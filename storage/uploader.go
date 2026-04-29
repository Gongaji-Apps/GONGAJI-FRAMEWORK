package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/converter"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
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

func UploadFromBase64(
	ctx context.Context,
	storage Storage,
	base64Data string,
	path string,
) (string, error) {
	decoded, contentType, err := converter.DecodeBase64(base64Data)
	if err != nil {
		return "", errors.NewBadRequest("Format base64 tidak valid")
	}
	return storage.Upload(ctx, bytes.NewReader(decoded), path, contentType)
}

func UploadWithValidation(
	ctx context.Context,
	storage Storage,
	file multipart.FileHeader,
	path string,
	allowedMimePrefix string,
	maxSizeKB int64,
) (string, error) {
	contentType := file.Header.Get("Content-Type")

	if allowedMimePrefix != "" && !strings.HasPrefix(contentType, allowedMimePrefix) {
		return "", errors.NewBadRequest(fmt.Sprintf("Tipe file tidak didukung: %s", contentType))
	}

	if maxSizeKB > 0 && file.Size > maxSizeKB*1024 {
		return "", errors.NewBadRequest(fmt.Sprintf("Ukuran file melebihi batas %d KB", maxSizeKB))
	}

	return UploadFromMultipart(ctx, storage, file, path)
}
