package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

func IsEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

func notBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("notblank", notBlank)
	})
}
