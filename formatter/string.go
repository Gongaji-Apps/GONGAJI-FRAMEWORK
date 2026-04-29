package formatter

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	indoCaser           = cases.Title(language.Indonesian)
	cleanDescriptionRgx = regexp.MustCompile(`[^a-zA-Z0-9\s\-.,/()']+`)
)

func TitleCase(value string) string {
	return indoCaser.String(strings.ToLower(strings.TrimSpace(value)))
}

func WrapLike(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
}

func LimitString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func JoinToSentence(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " dan " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", dan " + items[len(items)-1]
	}
}

func CleanDescription(value string) string {
	return cleanDescriptionRgx.ReplaceAllString(value, "")
}

func EscapeSQL(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "'", "''"))
}
