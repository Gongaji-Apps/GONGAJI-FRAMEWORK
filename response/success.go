package response

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"

	"github.com/gin-gonic/gin"
)

const defaultSuccessMessage = "Permintaan berhasil diproses."

func Success(ctx *gin.Context, data any) {
	Send(
		ctx,
		http.StatusOK,
		true,
		defaultSuccessMessage,
		data,
		nil,
		nil,
		nil,
	)
}

func Created(ctx *gin.Context, data any) {
	Send(
		ctx,
		http.StatusCreated,
		true,
		"Data berhasil dibuat.",
		data,
		nil,
		nil,
		nil,
	)
}

func Updated(ctx *gin.Context, data any) {
	Send(
		ctx,
		http.StatusOK,
		true,
		"Data berhasil diperbarui.",
		data,
		nil,
		nil,
		nil,
	)
}

func Deleted(ctx *gin.Context) {
	Send(
		ctx,
		http.StatusOK,
		true,
		"Data berhasil dihapus.",
		nil,
		nil,
		nil,
		nil,
	)
}

func NoContent(ctx *gin.Context) {
	ctx.Status(http.StatusNoContent)
}

func SuccessWithMessage(ctx *gin.Context, message string, data any) {
	Send(
		ctx,
		http.StatusOK,
		true,
		message,
		data,
		nil,
		nil,
		nil,
	)
}

func SuccessWithMeta(ctx *gin.Context, data any, meta any) {
	Send(
		ctx,
		http.StatusOK,
		true,
		defaultSuccessMessage,
		data,
		nil,
		nil,
		meta,
	)
}

func SuccessWithCache(ctx *gin.Context, data any, etag string, maxAge time.Duration) {
	setCacheHeaders(ctx, etag, maxAge)
	Send(
		ctx,
		http.StatusOK,
		true,
		defaultSuccessMessage,
		data,
		nil,
		nil,
		nil,
	)
}

func SuccessObject[T any](
	ctx *gin.Context,
	result result.ObjectResult[T],
) {
	Send(
		ctx,
		http.StatusOK,
		true,
		defaultSuccessMessage,
		result.Data,
		nil,
		nil,
		nil,
	)
}

func SuccessArray[T any](
	ctx *gin.Context,
	result result.ArrayResult[T],
) {
	Send(
		ctx,
		http.StatusOK,
		true,
		defaultSuccessMessage,
		result.Data,
		&result.DataTotal,
		&result.Pagination,
		nil,
	)
}

func SuccessArrayWithCache[T any](
	ctx *gin.Context,
	result result.ArrayResult[T],
	etag string,
	maxAge time.Duration,
) {
	setCacheHeaders(ctx, etag, maxAge)
	SuccessArray(ctx, result)
}

func setCacheHeaders(ctx *gin.Context, etag string, maxAge time.Duration) {
	expireTime := time.Now().Add(maxAge).UTC().Format(http.TimeFormat)
	ctx.Header("Cache-Control", fmt.Sprintf("public, stale-while-revalidate=%d", int(maxAge.Seconds())))
	ctx.Header("Expires", expireTime)
	if etag != "" {
		ctx.Header("ETag", etag)
	}
}
