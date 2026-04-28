package validator

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

func minInt(fl validator.FieldLevel) bool {
	field := fl.Field()

	// handle pointer (*string)
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return true // omitempty case
		}
		field = field.Elem()
	}

	val := strings.TrimSpace(field.String())

	// kosong → skip (biar omitempty jalan)
	if val == "" {
		return true
	}

	num, err := strconv.Atoi(val)
	if err != nil {
		return false
	}

	param := fl.Param()
	min, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	return num >= min
}

func maxInt(fl validator.FieldLevel) bool {
	field := fl.Field()

	// handle pointer (*string)
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return true // omitempty case
		}
		field = field.Elem()
	}

	val := strings.TrimSpace(field.String())

	// kosong → skip (biar omitempty jalan)
	if val == "" {
		return true
	}

	num, err := strconv.Atoi(val)
	if err != nil {
		return false
	}

	param := fl.Param()
	min, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	return num <= min
}

func init() {
	Register(func(v *validator.Validate) {
		v.RegisterValidation("min_int", minInt)
		v.RegisterValidation("max_int", maxInt)
	})
}
