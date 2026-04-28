package normalizer

import "strings"

func EmptyStringToNil(s *string) *string {
	if s == nil {
		return nil
	}
	if *s == "" {
		return nil
	}
	return s
}

func TrimAndNil(s *string) *string {
	if s == nil {
		return nil
	}

	str := strings.TrimSpace(*s)
	if str == "" {
		return nil
	}

	return &str
}

func UpperTrim(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
