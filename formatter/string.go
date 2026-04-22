package formatter

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var indoCaser = cases.Title(language.Indonesian)

func TitleCase(value string) string {
	return indoCaser.String(strings.ToLower(strings.TrimSpace(value)))
}

func WrapLike(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
}
