package response

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"
	"github.com/gin-gonic/gin"
)

func Send(
	ctx *gin.Context,
	statusCode int,
	status bool,
	message string,
	data any,
	dataTotal *int64,
	pagination *pagination.Meta,
	meta any,
) {
	response := Response{
		StatusCode: statusCode,
		Status:     status,
		Message:    message,
		Data:       data,
		DataTotal:  dataTotal,
		Pagination: pagination,
		Meta:       meta,
	}

	ctx.JSON(response.StatusCode, response)
}
