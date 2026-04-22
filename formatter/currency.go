package formatter

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var printer = message.NewPrinter(language.Indonesian)

func Rupiah(value int) string {
	return printer.Sprintf("%d", value)
}
