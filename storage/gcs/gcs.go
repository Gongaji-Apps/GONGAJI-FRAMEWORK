package gcs

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type Config struct {
	Bucket  string
	BaseURL string
}

type GCS struct {
	client *storage.Client
	cfg    Config
}

func New(client *storage.Client, cfg Config) *GCS {
	return &GCS{client: client, cfg: cfg}
}

func (g *GCS) Upload(ctx context.Context, r io.Reader, path string, contentType string) (string, error) {
	w := g.client.Bucket(g.cfg.Bucket).Object(path).NewWriter(ctx)
	w.ContentType = contentType

	if _, err := io.Copy(w, r); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", g.cfg.BaseURL, g.cfg.Bucket, path), nil
}

func (g *GCS) Delete(ctx context.Context, path string) error {
	return g.client.Bucket(g.cfg.Bucket).Object(path).Delete(ctx)
}

func (g *GCS) SignedURL(path string, expire time.Duration) (string, error) {
	return storage.SignedURL(
		g.cfg.Bucket,
		path,
		&storage.SignedURLOptions{
			Scheme:  storage.SigningSchemeV4,
			Method:  "GET",
			Expires: time.Now().Add(expire),
		},
	)
}

func (g *GCS) DeleteBatch(ctx context.Context, paths []string) error {
	ch := make(chan error, len(paths))

	for _, p := range paths {
		go func(path string) {
			ch <- g.Delete(ctx, path)
		}(p)
	}

	for i := 0; i < len(paths); i++ {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}

func (g *GCS) DeleteFolder(ctx context.Context, prefix string) error {
	it := g.client.Bucket(g.cfg.Bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if err := g.client.
			Bucket(g.cfg.Bucket).
			Object(attrs.Name).
			Delete(ctx); err != nil {
			return err
		}
	}

	return nil
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
