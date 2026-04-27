package response

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"
	"github.com/gin-gonic/gin"
)

func Send(
	ctx *gin.Context,
	status_code int,
	status bool,
	message string,
	data any,
	data_total *int64,
	pagination *pagination.Meta,
	meta any,
) {
	response := Response{
		Status_Code: status_code,
		Status:      status,
		Message:     message,
		Data:        data,
		Data_Total:  data_total,
		Pagination:  pagination,
		Meta:        meta,
	}

	ctx.JSON(response.Status_Code, response)
}
