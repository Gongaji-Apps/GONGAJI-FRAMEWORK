package random

import (
	"crypto/rand"
	"errors"
	"math/big"
)

func String(length int, charset string) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid length")
	}

	if charset == "" {
		charset = AlphaNumeric
	}

	result := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
