package response

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"

	"github.com/gin-gonic/gin"
)

func Success(ctx *gin.Context, data any) {
	Send(
		ctx,
		200,
		true,
		"Permintaan berhasil diproses.",
		data,
		nil,
		nil,
		nil,
	)
}

func Created(ctx *gin.Context, data any) {
	Send(
		ctx,
		201,
		true,
		"Data berhasil dibuat.",
		data,
		nil,
		nil,
		nil,
	)
}

func Deleted(ctx *gin.Context) {
	Send(
		ctx,
		200,
		true,
		"Data berhasil dihapus.",
		nil,
		nil,
		nil,
		nil,
	)
}

func SuccessObject[T any](
	ctx *gin.Context,
	result result.Object_Result[T],
) {
	Send(
		ctx,
		200,
		true,
		"Permintaan berhasil diproses.",
		result.Data,
		nil,
		nil,
		nil,
	)
}

func SuccessArray[T any](
	ctx *gin.Context,
	result result.Array_Result[T],
) {
	Send(
		ctx,
		200,
		true,
		"Permintaan berhasil diproses.",
		result.Data,
		&result.Data_Total,
		&result.Pagination,
		nil,
	)
}
