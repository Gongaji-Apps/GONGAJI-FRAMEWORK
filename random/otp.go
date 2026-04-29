package random

import "github.com/google/uuid"

func OTP(length int) (string, error) {
	return String(length, Numeric)
}

func Code(length int) (string, error) {
	return String(length, AlphaNumericUpper)
}

func UUID() string {
	return uuid.New().String()
}
