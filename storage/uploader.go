package storage

import (
	"context"
	"mime/multipart"
)

func UploadFromMultipart(
	ctx context.Context,
	storage Storage,
	file multipart.FileHeader,
	path string,
) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")

	return storage.Upload(ctx, src, path, contentType)
}
