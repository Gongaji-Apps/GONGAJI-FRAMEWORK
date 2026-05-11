package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var ErrDuplicate = errors.New("Afwan, Terdapat Data Duplicate pada data anda.")

func CheckDuplicate(values []string) error {
	seen := make(map[string]struct{}, len(values))

	for _, v := range values {
		if _, exists := seen[v]; exists {
			return ErrDuplicate
		}
		seen[v] = struct{}{}
	}
	return nil
}

func uniqueStringSlice(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() != reflect.Slice {
		return false
	}

	seen := make(map[string]struct{})

	for i := 0; i < field.Len(); i++ {
		val := field.Index(i)

		if val.Kind() != reflect.String {
			return false
		}

		s := strings.TrimSpace(val.String())

		if _, exists := seen[s]; exists {
			return false
		}

		seen[s] = struct{}{}
	}

	return true
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("unique", uniqueStringSlice)
	})
}
