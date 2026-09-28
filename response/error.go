package response

import (
	stderrors "errors"
	"net/http"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/gin-gonic/gin"
)

func httpStatus(err error) int {
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		switch appErr.Code {
		case errors.NotFound:
			return http.StatusNotFound
		case errors.Conflict:
			return http.StatusConflict
		case errors.Unauthorized:
			return http.StatusUnauthorized
		case errors.Forbidden:
			return http.StatusForbidden
		case errors.BadRequest:
			return http.StatusBadRequest
		case errors.PaymentRequired:
			return http.StatusPaymentRequired
		case errors.ServiceUnavailable:
			return http.StatusServiceUnavailable
		case errors.InternalServerError:
			return http.StatusInternalServerError
		default:
			return http.StatusInternalServerError
		}
	}

	return http.StatusInternalServerError
}

// PesanGalatUmum dipakai bila galat bukan *AppError (galat tak terduga).
const PesanGalatUmum = "Terjadi kesalahan di server. Coba lagi beberapa saat lagi."

func Error(ctx *gin.Context, err error) {
	// Galat constraint DB mentah (mis. dari Exec di service) dipetakan ke
	// pesan manusiawi alih-alih 500 generik.
	if errors.IsDBConstraintError(err) {
		err = errors.NormalizeDBError(err, "")
	}
	statusCode := httpStatus(err)

	message := PesanGalatUmum
	var meta any

	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		message = appErr.Message
		meta = appErr.Meta
	}

	Send(
		ctx,
		statusCode,
		false,
		message,
		nil,
		nil,
		nil,
		meta,
	)
}
