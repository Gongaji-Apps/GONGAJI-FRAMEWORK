package validator

import (
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/go-playground/validator/v10"
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

func imageFile(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(*multipart.FileHeader)

	if !ok || file == nil {
		return false
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))

	allowed := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	return allowed[ext]
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("image", imageFile)
	})
}
