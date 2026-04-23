package response

import (
	"net/http"

	frameworkErr "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

func mapHTTPStatus(err error) int {
	switch err {
	case frameworkErr.ErrBadRequest:
		return http.StatusBadRequest
	case frameworkErr.ErrUnauthorized:
		return http.StatusUnauthorized
	case frameworkErr.ErrForbidden:
		return http.StatusForbidden
	case frameworkErr.ErrNotFound:
		return http.StatusNotFound
	case frameworkErr.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
