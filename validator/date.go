package validator

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

func validDate(fl validator.FieldLevel) bool {
	value := strings.TrimSpace(fl.Field().String())

	if value == "" {
		return false
	}

	_, err := time.Parse("2006-01-02", value)

	return err == nil
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("date", validDate)
	})
}