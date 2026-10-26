package gcs

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Klien tanpa autentikasi cukup untuk memeriksa Writer: NewWriter belum mengirim
// apa pun ke jaringan sampai Write/Close dipanggil.
func newTestGCS(t *testing.T, cfg Config) *GCS {
	t.Helper()

	client, err := storage.NewClient(context.Background(), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("storage.NewClient: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return New(client, cfg)
}

func TestNewWriterSetsCacheControl(t *testing.T) {
	g := newTestGCS(t, Config{
		Bucket:       "bucket",
		CacheControl: "public, max-age=31536000, immutable",
	})

	w := g.newWriter(context.Background(), "files/a.png", "image/png")

	if w.CacheControl != "public, max-age=31536000, immutable" {
		t.Errorf("CacheControl = %q", w.CacheControl)
	}
	if w.ContentType != "image/png" {
		t.Errorf("ContentType = %q", w.ContentType)
	}
	// w.Bucket baru terisi dari respons server setelah Close, jadi yang diperiksa Name saja.
	if w.Name != "files/a.png" {
		t.Errorf("Name = %q", w.Name)
	}
}

func TestNewWriterWithoutCacheControlKeepsDefault(t *testing.T) {
	g := newTestGCS(t, Config{Bucket: "bucket"})

	w := g.newWriter(context.Background(), "files/a.png", "image/png")

	if w.CacheControl != "" {
		t.Errorf("CacheControl = %q, want empty (default GCS)", w.CacheControl)
	}
}
