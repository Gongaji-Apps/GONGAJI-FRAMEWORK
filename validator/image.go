package validator

import (
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

func ValidateImage(contentType string, size int64, maxMB int64) error {
	if !strings.HasPrefix(contentType, "image/") {
		return errors.NewBadRequest("invalid_image")
	}

	if size > maxMB*1024*1024 {
		return errors.NewBadRequest("image_too_large")
	}

	return nil
}
