package validator

import (
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

func numericNullable(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if strings.TrimSpace(value) == "" {
		return true
	}

	_, err := strconv.ParseInt(value, 10, 64)

	return err == nil
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("numeric_nullable", numericNullable)
	})
}
