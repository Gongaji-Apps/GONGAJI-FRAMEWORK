package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type user struct {
	ID   int
	Name string
}

func TestMemory_SetGet_RoundTrip(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	want := user{ID: 1, Name: "Budi"}
	if err := m.Set(ctx, "u:1", want, 0); err != nil {
		t.Fatal(err)
	}

	var got user
	if err := m.Get(ctx, "u:1", &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMemory_Get_Miss(t *testing.T) {
	m := NewMemory()
	var got user
	err := m.Get(context.Background(), "absent", &got)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMemory_Get_NilDestSkipsUnmarshal(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if err := m.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Get(ctx, "k", nil); err != nil {
		t.Errorf("Get with nil dest should be no-op, got %v", err)
	}
}

func TestMemory_TTL_Expires(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	if err := m.Set(ctx, "k", "v", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Still present immediately
	if ok, _ := m.Exists(ctx, "k"); !ok {
		t.Fatal("key should be present before TTL")
	}

	time.Sleep(80 * time.Millisecond)

	if ok, _ := m.Exists(ctx, "k"); ok {
		t.Errorf("key should be expired")
	}
	var dest string
	if err := m.Get(ctx, "k", &dest); !errors.Is(err, ErrNotFound) {
		t.Errorf("expired Get should return ErrNotFound, got %v", err)
	}
}

func TestMemory_TTL_ZeroNeverExpires(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if err := m.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if ok, _ := m.Exists(ctx, "k"); !ok {
		t.Errorf("ttl=0 should never expire")
	}
}

func TestMemory_Delete(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.Set(ctx, "k", "v", 0)
	if err := m.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := m.Exists(ctx, "k"); ok {
		t.Errorf("key should be gone after Delete")
	}
}

func TestMemory_Delete_AbsentIsNoop(t *testing.T) {
	m := NewMemory()
	if err := m.Delete(context.Background(), "absent"); err != nil {
		t.Errorf("Delete of absent key should be no-op, got %v", err)
	}
}

func TestMemory_Exists(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if ok, _ := m.Exists(ctx, "absent"); ok {
		t.Error("absent key should not exist")
	}
	_ = m.Set(ctx, "k", "v", 0)
	if ok, _ := m.Exists(ctx, "k"); !ok {
		t.Error("present key should exist")
	}
}

func TestMemory_Flush(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.Set(ctx, "a", "1", 0)
	_ = m.Set(ctx, "b", "2", 0)
	if err := m.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := m.Exists(ctx, "a"); ok {
		t.Error("a should be gone")
	}
	if ok, _ := m.Exists(ctx, "b"); ok {
		t.Error("b should be gone")
	}
}

func TestMemory_Sweep(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.Set(ctx, "ephemeral", "x", 50*time.Millisecond)
	_ = m.Set(ctx, "permanent", "x", 0)

	time.Sleep(80 * time.Millisecond)
	removed := m.Sweep()
	if removed != 1 {
		t.Errorf("Sweep removed = %d, want 1", removed)
	}
	if ok, _ := m.Exists(ctx, "permanent"); !ok {
		t.Error("permanent should survive Sweep")
	}
}

func TestMemory_ContextCancelled(t *testing.T) {
	m := NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Set(ctx, "k", "v", 0); err == nil {
		t.Error("Set with cancelled ctx should error")
	}
	var d string
	if err := m.Get(ctx, "k", &d); err == nil {
		t.Error("Get with cancelled ctx should error")
	}
}

func TestMemory_ConcurrentAccess(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = m.Set(ctx, "k", "v", 0)
		}()
		go func() {
			defer wg.Done()
			var d string
			_ = m.Get(ctx, "k", &d)
		}()
	}
	wg.Wait()
}

func TestMemory_StructWithSlice(t *testing.T) {
	type record struct {
		Tags []string
		Meta map[string]int
	}
	m := NewMemory()
	ctx := context.Background()
	want := record{Tags: []string{"a", "b"}, Meta: map[string]int{"x": 1}}
	if err := m.Set(ctx, "r", want, 0); err != nil {
		t.Fatal(err)
	}
	var got record
	if err := m.Get(ctx, "r", &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 2 || got.Meta["x"] != 1 {
		t.Errorf("got %+v", got)
	}
}
