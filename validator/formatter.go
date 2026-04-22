package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func FormatValidationError(language string, err error) []ValidationError {
	var errors []ValidationError

	translator := NewTranslator(language)

	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			errors = append(errors, ValidationError{
				Field:   strings.ToLower(e.Field()),
				Message: translator.Translate(e),
			})
		}
	}

	return errors
}
