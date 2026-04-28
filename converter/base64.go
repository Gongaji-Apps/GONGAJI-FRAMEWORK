package converter

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func Base64ToBytes(value string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(value)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data Base64 ke Byte. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return result, nil
}

// DecodeBase64 decodes a base64 string or data URI (e.g. "data:image/png;base64,...").
// Returns the decoded bytes, the detected content type, and any error.
func DecodeBase64(input string) ([]byte, string, error) {
	contentType := "application/octet-stream"

	if strings.Contains(input, ",") {
		parts := strings.SplitN(input, ",", 2)
		meta := parts[0]
		data := parts[1]

		if strings.Contains(meta, "data:") {
			meta = strings.TrimPrefix(meta, "data:")
			meta = strings.TrimSuffix(meta, ";base64")
			contentType = meta
		}

		input = data
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, "", err
	}

	return decoded, contentType, nil
}
