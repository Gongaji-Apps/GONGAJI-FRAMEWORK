package validator

import (
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/go-playground/validator/v10"
)

func File(contentType string, size int64, maxMB int64, allowedTypes []string) error {
	if len(allowedTypes) > 0 {
		allowed := false
		for _, t := range allowedTypes {
			if strings.HasPrefix(contentType, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.NewBadRequest(fmt.Sprintf("Tipe file tidak didukung: %s", contentType))
		}
	}

	if maxMB > 0 && size > maxMB*1024*1024 {
		return errors.NewBadRequest(fmt.Sprintf("Ukuran file melebihi batas %d MB", maxMB))
	}

	return nil
}

func maxFileSize(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(multipart.FileHeader)

	if !ok {
		return false
	}

	maxMB, err := strconv.ParseInt(fl.Param(), 10, 64)

	if err != nil {
		return false
	}

	maxBytes := maxMB * 1024 * 1024

	return file.Size <= maxBytes
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("max_file_size", maxFileSize)
	})
}
