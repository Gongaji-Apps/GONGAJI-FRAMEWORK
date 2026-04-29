package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type echoBody struct {
	Hello string `json:"hello"`
}

func newTestClient(srv *httptest.Server, retry *RetryConfig) *Client {
	return New(Config{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
		Retry:   retry,
	})
}

func TestClient_GetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(echoBody{Hello: "world"})
	}))
	defer srv.Close()

	c := newTestClient(srv, nil)

	var result echoBody
	if err := c.Get(context.Background(), "/echo", &result); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Hello != "world" {
		t.Fatalf("got %q, want %q", result.Hello, "world")
	}
}

func TestClient_PostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var in echoBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(in)
	}))
	defer srv.Close()

	c := newTestClient(srv, nil)
	var result echoBody
	if err := c.Post(context.Background(), "/echo", echoBody{Hello: "ping"}, &result); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if result.Hello != "ping" {
		t.Fatalf("got %q, want %q", result.Hello, "ping")
	}
}

func TestClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"bad input"}`)
	}))
	defer srv.Close()

	c := newTestClient(srv, &RetryConfig{MaxAttempts: 1})
	err := c.Get(context.Background(), "/x", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	httpErr, ok := AsHTTPError(err)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", httpErr.StatusCode)
	}
	if !strings.Contains(string(httpErr.Body), "bad input") {
		t.Errorf("Body missing payload, got %q", httpErr.Body)
	}
}

func TestClient_RetriesOn5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(echoBody{Hello: "ok"})
	}))
	defer srv.Close()

	c := newTestClient(srv, &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   2,
		JitterFactor: 0,
	})

	var result echoBody
	if err := c.Get(context.Background(), "/", &result); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 calls, got %d", got)
	}
}

func TestClient_DoesNotRetry4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestClient(srv, &RetryConfig{MaxAttempts: 5, InitialDelay: time.Millisecond})
	_ = c.Get(context.Background(), "/", nil)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 call (no retry on 4xx), got %d", got)
	}
}

func TestClient_RespectsRetryAfter(t *testing.T) {
	var calls int32
	var firstAt, secondAt time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			firstAt = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		secondAt = time.Now()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, &RetryConfig{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond,
		Multiplier:   2,
		JitterFactor: 0,
	})

	if err := c.Get(context.Background(), "/", nil); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls, got %d", got)
	}
	if elapsed := secondAt.Sub(firstAt); elapsed < 900*time.Millisecond {
		t.Errorf("Retry-After not honored: elapsed %v", elapsed)
	}
}

func TestClient_PostForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", got)
		}
		_ = r.ParseForm()
		_ = json.NewEncoder(w).Encode(map[string]string{"got": r.FormValue("name")})
	}))
	defer srv.Close()

	c := newTestClient(srv, nil)
	values := url.Values{}
	values.Set("name", "alice")

	var result map[string]string
	if err := c.PostForm(context.Background(), "/", values, &result); err != nil {
		t.Fatalf("PostForm error: %v", err)
	}
	if result["got"] != "alice" {
		t.Errorf("got %q, want alice", result["got"])
	}
}

func TestClient_WithAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer xyz" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, nil).WithAuth("Bearer", "xyz")
	if err := c.Get(context.Background(), "/", nil); err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestClient_WithHeader_DoesNotMutateOriginal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Header.Get("X-Tenant")))
	}))
	defer srv.Close()

	base := newTestClient(srv, nil)
	scoped := base.WithHeader("X-Tenant", "acme")

	var got []byte
	if err := scoped.Get(context.Background(), "/", &got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "acme" {
		t.Errorf("scoped tenant = %q, want acme", got)
	}

	got = nil
	if err := base.Get(context.Background(), "/", &got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "" {
		t.Errorf("base tenant should be empty, got %q", got)
	}
}

func TestClient_ContextCancellationStopsRetry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv, &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   2,
		JitterFactor: 0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := c.Get(ctx, "/", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		if _, ok := AsHTTPError(err); !ok {
			t.Logf("got error: %v (acceptable as long as retry stopped)", err)
		}
	}
	if got := atomic.LoadInt32(&calls); got > 2 {
		t.Errorf("expected retry to stop on context cancel, got %d calls", got)
	}
}

func TestResolveURL(t *testing.T) {
	c := New(Config{BaseURL: "https://api.example.com/"})
	cases := map[string]string{
		"":                      "https://api.example.com",
		"/path":                 "https://api.example.com/path",
		"path":                  "https://api.example.com/path",
		"https://other.com/foo": "https://other.com/foo",
	}
	for in, want := range cases {
		if got := c.resolveURL(in); got != want {
			t.Errorf("resolveURL(%q) = %q, want %q", in, got, want)
		}
	}
}
