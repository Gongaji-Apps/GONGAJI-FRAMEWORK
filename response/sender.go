package response

import (
	"github.com/gin-gonic/gin"

	commonErr "github.com/Gongaji-Apps/GONGAJI-COMMON/errors"
)

func Success(ctx *gin.Context, data interface{}) {
	ctx.JSON(200, Response{
		Meta: Meta{
			Code:    200,
			Message: "success",
		},
		Data: data,
	})
}

func Error(ctx *gin.Context, err error) {
	status := mapHTTPStatus(err)

	appErr, ok := err.(*commonErr.AppError)
	message := "Internal server error"

	if ok {
		message = appErr.Message
	}

	ctx.JSON(status, Response{
		Meta: Meta{
			Code:    status,
			Message: message,
		},
	})
}
