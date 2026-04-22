package converter

import (
	"encoding/base64"
)

func BytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
