package binding

import (
	"encoding/json"
	stderrors "errors"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/normalizer"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
	"github.com/gin-gonic/gin"
)

// Pesan ringkas galat validasi — rincian per medan ada di meta.
const (
	validationMessageID = "Periksa kembali isian Anda."
	validationMessageEN = "Please check your input."
)

// ================================================
// ==================== HELPER ====================
// ================================================

func bindError(ctx *gin.Context, err error) error {
	lang := ctx.GetHeader("Accept-Language")
	en := strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "en")
	msg := validationMessageID
	if en {
		msg = validationMessageEN
	}

	meta := validator.FormatValidationError(lang, err)
	if len(meta) == 0 {
		// Bukan galat tag validasi: JSON rusak / tipe tak cocok.
		var typeErr *json.UnmarshalTypeError
		if stderrors.As(err, &typeErr) && typeErr.Field != "" {
			field := typeErr.Field
			if i := strings.LastIndex(field, "."); i >= 0 {
				field = field[i+1:]
			}
			label := errors.FieldLabel(field)
			pesan := label + " memiliki format yang tidak sesuai."
			if en {
				pesan = label + " has an invalid format."
			}
			meta = []validator.ValidationError{{Field: strings.ToLower(field), Message: pesan}}
		} else {
			if en {
				msg = "The submitted data could not be read. Refresh the page and try again."
			} else {
				msg = "Data yang dikirim tidak dapat dibaca. Muat ulang halaman lalu coba lagi."
			}
		}
	}

	return errors.NewBadRequestValidation(msg, meta)
}

// ==============================================
// ==================== JSON ====================
// ==============================================

func JSON[T any](ctx *gin.Context) (T, error) {
	var payload T

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return payload, bindError(ctx, err)
	}

	normalizer.NormalizeStruct(&payload)

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

	normalizer.NormalizeStruct(&payload)

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

	normalizer.NormalizeStruct(&payload)

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
