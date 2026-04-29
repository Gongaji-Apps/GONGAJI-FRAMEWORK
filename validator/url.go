package validator

import (
	"net/url"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

func URL(rawURL string) error {
	if rawURL == "" {
		return errors.NewBadRequest("URL tidak boleh kosong")
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return errors.NewBadRequest("Format URL tidak valid")
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.NewBadRequest("Format URL tidak valid")
	}
	return nil
}
