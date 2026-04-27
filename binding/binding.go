package binding

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/normalizer"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
	"github.com/gin-gonic/gin"
)

const validationMessage = "Validation Error!"

// ================================================
// ==================== HELPER ====================
// ================================================

func bindError(ctx *gin.Context, err error) error {
	lang := ctx.GetHeader("Accept-Language")

	return errors.NewBadRequestValidation(
		validationMessage,
		validator.FormatValidationError(lang, err),
	)
}

// ==============================================
// ==================== JSON ====================
// ==============================================

func JSON[T any](ctx *gin.Context) (T, error) {
	var payload T

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return payload, bindError(ctx, err)
	}

	return payload, nil
}

// ===============================================
// ==================== QUERY ====================
// ===============================================

func Query[T any](ctx *gin.Context) (T, error) {
	var payload T

	if err := ctx.ShouldBindQuery(&payload); err != nil {
		return payload, bindError(ctx, err)
	}

	return payload, nil
}

// =============================================
// ==================== URI ====================
// =============================================

func URI[T any](ctx *gin.Context) (T, error) {
	var payload T

	if err := ctx.ShouldBindUri(&payload); err != nil {
		return payload, bindError(ctx, err)
	}

	return payload, nil
}

// ==============================================
// ==================== FORM ====================
// ==============================================

func Form[T any](ctx *gin.Context) (T, error) {
	var payload T

	if err := ctx.ShouldBind(&payload); err != nil {
		return payload, bindError(ctx, err)
	}

	normalizer.NormalizeStruct(&payload)

	return payload, nil
}
