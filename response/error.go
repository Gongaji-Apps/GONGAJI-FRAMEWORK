package response

import (
	"net/http"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/gin-gonic/gin"
)

func httpStatus(err error) int {
	if appErr, ok := err.(*errors.AppError); ok {
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

func Error(ctx *gin.Context, err error) {
	statusCode := httpStatus(err)

	message := "[Internal Server Error] Afwan, terjadi kesalahan pada sistem."
	var meta any

	if appErr, ok := err.(*errors.AppError); ok {
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
