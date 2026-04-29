package validator

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/google/uuid"
)

func UUID(value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return errors.NewBadRequest("Format UUID tidak valid")
	}
	return nil
}
