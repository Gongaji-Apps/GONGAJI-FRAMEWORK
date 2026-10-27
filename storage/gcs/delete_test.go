package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// fakeGCS meniru dua endpoint JSON API yang dipakai delete: list objek (dengan
// paginasi) dan hapus objek. Mencatat puncak request hapus yang berjalan serentak.
type fakeGCS struct {
	mu       sync.Mutex
	objects  map[string]bool
	missing  map[string]bool // dilaporkan 404 saat dihapus
	failOn   string          // dilaporkan 500 saat dihapus
	inFlight int32
	peak     int32
	pageSize int
}

func (f *fakeGCS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.EscapedPath(), "/storage/v1/b/bucket/o")

	switch {
	case r.Method == http.MethodGet && path == "":
		f.list(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/"):
		name, _ := url.PathUnescape(strings.TrimPrefix(path, "/"))
		f.delete(w, name)
	default:
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotImplemented)
	}
}

func (f *fakeGCS) list(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	var names []string
	for n := range f.objects {
		if strings.HasPrefix(n, r.URL.Query().Get("prefix")) {
			names = append(names, n)
		}
	}
	f.mu.Unlock()

	// urutkan agar token halaman deterministik
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	// Seperti GCS asli, token halaman = nama objek terakhir (bukan offset), jadi
	// objek yang dihapus di tengah paginasi tak menggeser halaman berikutnya.
	after := r.URL.Query().Get("pageToken")
	var page []string
	for _, n := range names {
		if n > after {
			page = append(page, n)
		}
	}
	more := len(page) > f.pageSize
	if more {
		page = page[:f.pageSize]
	}

	items := []map[string]string{}
	for _, n := range page {
		items = append(items, map[string]string{"name": n, "bucket": "bucket"})
	}
	resp := map[string]any{"kind": "storage#objects", "items": items}
	if more {
		resp["nextPageToken"] = page[len(page)-1]
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *fakeGCS) delete(w http.ResponseWriter, name string) {
	now := atomic.AddInt32(&f.inFlight, 1)
	defer atomic.AddInt32(&f.inFlight, -1)
	for {
		p := atomic.LoadInt32(&f.peak)
		if now <= p || atomic.CompareAndSwapInt32(&f.peak, p, now) {
			break
		}
	}
	time.Sleep(5 * time.Millisecond) // beri kesempatan request lain menumpuk

	if name == f.failOn {
		http.Error(w, `{"error":{"code":500,"message":"boom"}}`, http.StatusInternalServerError)
		return
	}
	if f.missing[name] {
		http.Error(w, `{"error":{"code":404,"message":"No such object"}}`, http.StatusNotFound)
		return
	}

	f.mu.Lock()
	delete(f.objects, name)
	f.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func newFakeGCS(t *testing.T, fake *fakeGCS) *GCS {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	client, err := storage.NewClient(context.Background(),
		option.WithEndpoint(srv.URL+"/storage/v1/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return New(client, Config{Bucket: "bucket"})
}

func objects(prefix string, n int) map[string]bool {
	m := map[string]bool{}
	for i := 0; i < n; i++ {
		m[fmt.Sprintf("%s%03d.png", prefix, i)] = true
	}
	return m
}

func TestDeleteFolder_SemuaTerhapusParalelTerbatas(t *testing.T) {
	objs := objects("folder/", 120)
	objs["lain/tetap.png"] = true
	fake := &fakeGCS{objects: objs, pageSize: 50}
	g := newFakeGCS(t, fake)

	if err := g.DeleteFolder(context.Background(), "folder/"); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}

	if len(fake.objects) != 1 || !fake.objects["lain/tetap.png"] {
		t.Errorf("sisa objek = %v, mau hanya lain/tetap.png", fake.objects)
	}
	if fake.peak > deleteConcurrency {
		t.Errorf("puncak paralel %d melebihi batas %d", fake.peak, deleteConcurrency)
	}
	if fake.peak < 2 {
		t.Errorf("puncak paralel %d — penghapusan tidak paralel", fake.peak)
	}
}

func TestDeleteFolder_ObjekSudahHilangBukanGalat(t *testing.T) {
	fake := &fakeGCS{objects: objects("f/", 5), missing: map[string]bool{"f/002.png": true}, pageSize: 100}
	if err := newFakeGCS(t, fake).DeleteFolder(context.Background(), "f/"); err != nil {
		t.Fatalf("404 seharusnya diabaikan: %v", err)
	}
}

func TestDeleteFolder_GalatDeleteDikembalikan(t *testing.T) {
	fake := &fakeGCS{objects: objects("f/", 40), failOn: "f/003.png", pageSize: 10}
	err := newFakeGCS(t, fake).DeleteFolder(context.Background(), "f/")
	if err == nil || strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("mau galat Delete asli, dapat %v", err)
	}
}

func TestDeleteBatch_ParalelTerbatas(t *testing.T) {
	objs := objects("b/", 100)
	paths := make([]string, 0, len(objs))
	for n := range objs {
		paths = append(paths, n)
	}
	fake := &fakeGCS{objects: objs, pageSize: 100}

	if err := newFakeGCS(t, fake).DeleteBatch(context.Background(), paths); err != nil {
		t.Fatalf("DeleteBatch: %v", err)
	}
	if len(fake.objects) != 0 {
		t.Errorf("tersisa %d objek", len(fake.objects))
	}
	if fake.peak > deleteConcurrency {
		t.Errorf("puncak paralel %d melebihi batas %d", fake.peak, deleteConcurrency)
	}
}
