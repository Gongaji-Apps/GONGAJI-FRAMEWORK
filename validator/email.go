package validator

import (
	"errors"
	"net/mail"
)

var ErrInvalidEmail = errors.New("format email tidak valid")

func Email(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}
