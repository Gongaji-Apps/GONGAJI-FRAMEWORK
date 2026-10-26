package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
)

type Config struct {
	Bucket  string
	BaseURL string

	// CacheControl, bila diisi, dipasang sebagai metadata Cache-Control pada setiap
	// objek yang di-Upload. Kosong = perilaku lama: GCS memakai default-nya sendiri,
	// yaitu "public, max-age=3600" untuk objek publik — klien mengunduh ulang tiap jam.
	//
	// Nilai panjang seperti "public, max-age=31536000, immutable" hanya aman kalau
	// path objek selalu unik (UUID/kode acak) dan tak pernah ditimpa dengan nama sama.
	CacheControl string
}

type GCS struct {
	client *storage.Client
	cfg    Config
}

func New(client *storage.Client, cfg Config) *GCS {
	return &GCS{client: client, cfg: cfg}
}

func (g *GCS) Upload(ctx context.Context, r io.Reader, path string, contentType string) (string, error) {
	w := g.newWriter(ctx, path, contentType)

	if _, err := io.Copy(w, r); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", g.cfg.BaseURL, g.cfg.Bucket, path), nil
}

// newWriter menyiapkan Writer beserta metadatanya. Dipisah dari Upload sebagai seam
// test: membuat Writer belum menyentuh jaringan, jadi atributnya bisa diperiksa hermetik.
func (g *GCS) newWriter(ctx context.Context, path string, contentType string) *storage.Writer {
	w := g.client.Bucket(g.cfg.Bucket).Object(path).NewWriter(ctx)
	w.ContentType = contentType

	if g.cfg.CacheControl != "" {
		w.CacheControl = g.cfg.CacheControl
	}

	return w
}

func (g *GCS) Delete(ctx context.Context, path string) error {
	return g.client.Bucket(g.cfg.Bucket).Object(path).Delete(ctx)
}

// SignedURL menandatangani URL objek (V4, GET) memakai kredensial si klien.
//
// Lewat BucketHandle.SignedURL, BUKAN storage.SignedURL paket: versi paket wajib
// diberi GoogleAccessID+PrivateKey eksplisit, dan tanpa itu selalu gagal
// "missing required credentials" — bug yang selama ini memaksa service konsumen
// (GONGAJI-API-NGAJI internal/storage/signer.go) menyalin logika penandatanganan
// sendiri. BucketHandle menurunkannya otomatis dari kredensial klien
// (option.WithCredentialsJSON / ADC).
func (g *GCS) SignedURL(path string, expire time.Duration) (string, error) {
	return g.client.Bucket(g.cfg.Bucket).SignedURL(path, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expire),
	})
}

// deleteConcurrency membatasi Delete paralel. Tanpa batas, batch besar memicu
// ribuan goroutine + request serentak dan terkena rate limit GCS (429).
const deleteConcurrency = 16

func (g *GCS) DeleteBatch(ctx context.Context, paths []string) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(deleteConcurrency)

	for _, p := range paths {
		path := p
		eg.Go(func() error {
			return g.Delete(ctx, path)
		})
	}

	return eg.Wait()
}

// DeleteFolder menghapus semua objek ber-prefix secara paralel (dibatasi
// deleteConcurrency), sambil terus membaca halaman listing berikutnya. Objek yang
// sudah terhapus lebih dulu (ErrObjectNotExist) tidak dianggap gagal.
func (g *GCS) DeleteFolder(ctx context.Context, prefix string) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(deleteConcurrency)

	it := g.client.Bucket(g.cfg.Bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			// Delete yang gagal membatalkan ctx sehingga listing ikut gagal;
			// kembalikan galat Delete aslinya, bukan "context canceled".
			if werr := eg.Wait(); werr != nil {
				return werr
			}
			return err
		}

		name := attrs.Name
		eg.Go(func() error {
			err := g.Delete(ctx, name)
			if errors.Is(err, storage.ErrObjectNotExist) {
				return nil
			}
			return err
		})
	}

	return eg.Wait()
}

func (g *GCS) Copy(ctx context.Context, src, dst string) error {
	source := g.client.Bucket(g.cfg.Bucket).Object(src)
	target := g.client.Bucket(g.cfg.Bucket).Object(dst)

	_, err := target.CopierFrom(source).Run(ctx)
	return err
}

func (g *GCS) Move(ctx context.Context, src, dst string) error {
	if err := g.Copy(ctx, src, dst); err != nil {
		return err
	}

	return g.Delete(ctx, src)
}

func (g *GCS) Exists(ctx context.Context, path string) (bool, error) {
	_, err := g.client.Bucket(g.cfg.Bucket).Object(path).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *GCS) GetURL(path string) string {
	return fmt.Sprintf("%s/%s/%s", g.cfg.BaseURL, g.cfg.Bucket, path)
}
