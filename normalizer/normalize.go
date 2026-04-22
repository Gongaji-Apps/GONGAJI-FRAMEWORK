package normalizer

import (
	"reflect"
	"strings"
)

func NormalizeStruct(input interface{}) {
	val := reflect.ValueOf(input)

	if val.Kind() != reflect.Ptr || val.IsNil() {
		return
	}

	normalizeValue(val.Elem())
}

func normalizeValue(v reflect.Value) {
	switch v.Kind() {

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			typeField := v.Type().Field(i)

			if !field.CanSet() {
				continue
			}

			tag := typeField.Tag.Get("normalize")
			applyNormalize(field, tag)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			normalizeValue(v.Index(i))
		}
	}
}

func applyNormalize(field reflect.Value, tag string) {
	switch field.Kind() {

	case reflect.Ptr:
		if field.IsNil() {
			return
		}

		elem := field.Elem()

		if elem.Kind() == reflect.String {
			str := strings.TrimSpace(elem.String())

			if str == "" && hasOption(tag, "nil") {
				field.Set(reflect.Zero(field.Type()))
			} else {
				elem.SetString(str)
			}
			return
		}

		normalizeValue(elem)

	case reflect.String:
		field.SetString(strings.TrimSpace(field.String()))

	case reflect.Struct:
		normalizeValue(field)

	case reflect.Slice, reflect.Array:
		normalizeValue(field)
	}
}

func hasOption(tag string, opt string) bool {
	for _, t := range strings.Split(tag, ",") {
		if t == opt {
			return true
		}
	}
	return false
}
