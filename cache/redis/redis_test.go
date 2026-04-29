package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	frameworkCache "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache"
	goredis "github.com/redis/go-redis/v9"
)

type product struct {
	ID    int
	Name  string
	Price float64
}

func newTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewFromClient(client, ""), mr
}

func TestRedis_SetGet_RoundTrip(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	want := product{ID: 1, Name: "Buku", Price: 25_000}
	if err := c.Set(ctx, "p:1", want, 0); err != nil {
		t.Fatal(err)
	}

	var got product
	if err := c.Get(ctx, "p:1", &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRedis_Get_Miss(t *testing.T) {
	c, _ := newTestCache(t)
	var got product
	err := c.Get(context.Background(), "absent", &got)
	if !errors.Is(err, frameworkCache.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRedis_Get_NilDestSkipsUnmarshal(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, "k", nil); err != nil {
		t.Errorf("Get with nil dest should succeed, got %v", err)
	}
}

func TestRedis_TTL_Expires(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, "k"); !ok {
		t.Fatal("key should exist before TTL")
	}

	// Advance miniredis's clock past the TTL.
	mr.FastForward(2 * time.Minute)

	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Errorf("key should be expired after FastForward")
	}
}

func TestRedis_TTL_NeverExpires(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(24 * time.Hour)
	if ok, _ := c.Exists(ctx, "k"); !ok {
		t.Errorf("ttl=0 should never expire")
	}
}

func TestRedis_Delete(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "k", "v", 0)
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, "k"); ok {
		t.Errorf("key should be gone")
	}
}

func TestRedis_Delete_AbsentIsNoop(t *testing.T) {
	c, _ := newTestCache(t)
	if err := c.Delete(context.Background(), "absent"); err != nil {
		t.Errorf("Delete of absent should be no-op, got %v", err)
	}
}

func TestRedis_Flush(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "a", "1", 0)
	_ = c.Set(ctx, "b", "2", 0)
	if err := c.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.Exists(ctx, "a"); ok {
		t.Error("a should be gone")
	}
}

func TestRedis_KeyPrefix(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c := NewFromClient(client, "tenant1:")
	ctx := context.Background()

	if err := c.Set(ctx, "user:1", "alice", 0); err != nil {
		t.Fatal(err)
	}
	// Underlying store should hold the prefixed key.
	got, err := mr.Get("tenant1:user:1")
	if err != nil {
		t.Fatalf("expected prefixed key in miniredis, got err: %v", err)
	}
	// Stored value is JSON-encoded.
	if got != `"alice"` {
		t.Errorf("stored value = %q, want \"alice\"", got)
	}

	var s string
	if err := c.Get(ctx, "user:1", &s); err != nil {
		t.Fatal(err)
	}
	if s != "alice" {
		t.Errorf("got %q", s)
	}
}

func TestRedis_NewFromClient_CloseIsNoop(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c := NewFromClient(client, "")
	if err := c.Close(); err != nil {
		t.Errorf("Close on borrowed client should be no-op, got %v", err)
	}
	// Client should still be usable.
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Errorf("client should still be usable, got %v", err)
	}
}

func TestRedis_StoredAsJSON_RoundTripsAcrossBackends(t *testing.T) {
	type record struct {
		Name string         `json:"name"`
		Tags []string       `json:"tags"`
		Meta map[string]int `json:"meta"`
	}
	c, _ := newTestCache(t)
	ctx := context.Background()
	want := record{Name: "x", Tags: []string{"a", "b"}, Meta: map[string]int{"n": 1}}

	if err := c.Set(ctx, "r", want, 0); err != nil {
		t.Fatal(err)
	}
	var got record
	if err := c.Get(ctx, "r", &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name || len(got.Tags) != 2 || got.Meta["n"] != 1 {
		t.Errorf("got %+v", got)
	}
}
