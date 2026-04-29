package validator

import (
	"fmt"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
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
