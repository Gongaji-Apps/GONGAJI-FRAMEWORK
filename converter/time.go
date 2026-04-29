package converter

import (
	"time"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

func TimeToString(t time.Time, layout string) string {
	return t.Format(layout)
}

func StringToTime(value, layout string) (*time.Time, error) {
	result, err := time.Parse(layout, value)
	if err != nil {
		return nil, errors.NewBadRequest("Format waktu tidak valid")
	}
	return &result, nil
}
