package validator

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidPhone = errors.New("format phone tidak valid")
	phoneRegex      = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
)

func Phone(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return ErrInvalidPhone
	}
	return nil
}