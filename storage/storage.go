package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, reader io.Reader, path string, contentType string) (string, error)
	Delete(ctx context.Context, path string) error
	DeleteBatch(ctx context.Context, paths []string) error
	DeleteFolder(ctx context.Context, prefix string) error
	Copy(ctx context.Context, src, dst string) error
	Move(ctx context.Context, src, dst string) error
	SignedURL(path string, expire time.Duration) (string, error)
}
