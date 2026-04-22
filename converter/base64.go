package converter

import (
	"encoding/base64"
	"fmt"
	"os"
)

func Base64ToBytes(value string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(value)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data Base64 ke Byte. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return result, nil
}
