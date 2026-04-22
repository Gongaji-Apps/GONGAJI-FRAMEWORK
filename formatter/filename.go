package formatter

import (
	"fmt"
	"mime/multipart"
	"strings"
)

func WithOriginalExtension(file multipart.FileHeader, filename string) string {
	if dot := strings.LastIndex(file.Filename, "."); dot != -1 {
		return fmt.Sprintf("%s.%s", filename, file.Filename[dot+1:])
	}
	return filename
}

func WithOriginalExtensionBase64(base64Header string, filename string) string {
	switch {
	case strings.Contains(base64Header, "image/jpeg"):
		return fmt.Sprintf("%s.%s", filename, "jpg")
	case strings.Contains(base64Header, "image/webp"):
		return fmt.Sprintf("%s.%s", filename, "webp")
	default:
		return fmt.Sprintf("%s.%s", filename, "png")
	}
}
