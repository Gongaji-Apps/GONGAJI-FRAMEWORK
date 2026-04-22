package converter

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var rupiahPrinter = message.NewPrinter(language.Indonesian)

func Float64ToRupiah(value float64) string {
	return "Rp " + rupiahPrinter.Sprintf("%.2f", value)
}
