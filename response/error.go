package response

import (
	"net/http"

	commonErr "github.com/Gongaji-Apps/GONGAJI-COMMON/errors"
)

func mapHTTPStatus(err error) int {
	switch err {
	case commonErr.ErrBadRequest:
		return http.StatusBadRequest
	case commonErr.ErrUnauthorized:
		return http.StatusUnauthorized
	case commonErr.ErrForbidden:
		return http.StatusForbidden
	case commonErr.ErrNotFound:
		return http.StatusNotFound
	case commonErr.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}