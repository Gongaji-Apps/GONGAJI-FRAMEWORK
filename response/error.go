package response

import (
	stderrors "errors"
	"net/http"
	"sync"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/logger"
	"github.com/gin-gonic/gin"
)

// Logger untuk galat tak terduga (bukan *AppError). Default dibuat malas lewat
// logger.New(); service boleh mengganti dengan SetLogger (mis. logger yang sama
// dengan main.go, atau buffer di test).
var (
	unexpectedLog     *logger.Logger
	unexpectedLogOnce sync.Once
	unexpectedLogMu   sync.RWMutex
)

// SetLogger mengganti logger yang dipakai Error untuk galat tak terduga.
func SetLogger(l *logger.Logger) {
	unexpectedLogMu.Lock()
	defer unexpectedLogMu.Unlock()
	unexpectedLog = l
}

func unexpectedLogger() *logger.Logger {
	unexpectedLogMu.RLock()
	l := unexpectedLog
	unexpectedLogMu.RUnlock()
	if l != nil {
		return l
	}
	unexpectedLogOnce.Do(func() {
		unexpectedLogMu.Lock()
		defer unexpectedLogMu.Unlock()
		if unexpectedLog == nil {
			unexpectedLog = logger.New()
		}
	})
	unexpectedLogMu.RLock()
	defer unexpectedLogMu.RUnlock()
	return unexpectedLog
}

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
	} else if err != nil {
		// Klien hanya menerima pesan umum; penyebab aslinya dicatat di sini agar
		// 500 bisa ditelusuri di Cloud Logging, dan dititipkan ke ctx.Errors untuk
		// middleware yang ingin memprosesnya.
		_ = ctx.Error(err)

		fields := map[string]any{"status_code": statusCode}
		if ctx.Request != nil {
			fields["method"] = ctx.Request.Method
			fields["path"] = ctx.Request.URL.Path
		}

		entry := unexpectedLogger().WithFields(fields)
		if ctx.Request != nil {
			entry = unexpectedLogger().WithCtx(ctx.Request.Context()).WithFields(fields)
		}
		entry.WithError(err).Error("galat tak terduga")
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
