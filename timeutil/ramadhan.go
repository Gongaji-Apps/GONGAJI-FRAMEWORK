package timeutil

import "time"

var ramadhanStart = time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC)
var ramadhanEnd = time.Date(2026, 3, 20, 23, 59, 59, 0, time.UTC)

func IsRamadhan(date time.Time) bool {
	return !date.Before(ramadhanStart) && !date.After(ramadhanEnd)
}
