package converter

import (
	"errors"
	"reflect"
	"strings"
)

var ErrNotStruct = errors.New("input must be struct")

func StructToMap(input any) (map[string]any, error) {
	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	typ := val.Type()
	result := make(map[string]any)

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		name := strings.ToLower(fieldType.Name)

		// skip preload
		if strings.Contains(name, "preload") {
			continue
		}

		if isZeroValue(field) {
			continue
		}

		result[name] = field.Interface()
	}

	return result, nil
}

func isZeroValue(v reflect.Value) bool {
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}
