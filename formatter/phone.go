package formatter

func NormalizeIndonesiaPhone(number string) string {
	if len(number) > 0 && number[0] == '0' {
		return "62" + number[1:]
	}
	return number
}