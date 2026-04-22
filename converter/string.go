package converter

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func StringToInt(value string) (*int, error) {
	result, err := strconv.Atoi(value)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error]12 Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data String ke Integer. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return &result, nil
}

func StringToFloat32(value string) (*float64, error) {
	result, err := strconv.ParseFloat(value, 32)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data String ke Float32. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return &result, nil
}

func StringToFloat64(value string) (*float64, error) {
	result, err := strconv.ParseFloat(value, 64)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data String ke Float64. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return &result, nil
}

func StringToBool(value string) (*bool, error) {
	result, err := strconv.ParseBool(value)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data String ke Boolean. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return &result, nil
}

func StringToDate(value string) (*time.Time, error) {
	result, err := time.Parse("2006-01-02", value)

	if err != nil {
		return nil, fmt.Errorf("[Internal Server Error] Oops! Kami mengalami masalah saat melakukan Konversi Tipe Data String ke Date. %s", os.Getenv("ADDITIONAL_ERR_500"))
	}

	return &result, nil
}
