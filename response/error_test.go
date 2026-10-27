package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

func kirim(t *testing.T, err error) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	Error(ctx, err)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

func TestError_AppErrorTerbungkus(t *testing.T) {
	code, body := kirim(t, fmt.Errorf("lapisan: %w", errors.NewNotFound("Batch tidak ditemukan.")))
	if code != http.StatusNotFound || body["message"] != "Batch tidak ditemukan." {
		t.Fatalf("dapat %d %v", code, body["message"])
	}
}

func TestError_ConstraintMentahJadiManusiawi(t *testing.T) {
	code, body := kirim(t, fmt.Errorf("exec: %w", &pgconn.PgError{Code: "23505", TableName: "slot_mstr"}))
	msg, _ := body["message"].(string)
	if code != http.StatusConflict || strings.Contains(msg, "slot_mstr") || strings.Contains(msg, "[") {
		t.Fatalf("dapat %d %q", code, msg)
	}
}

func TestError_GalatUmum(t *testing.T) {
	code, body := kirim(t, fmt.Errorf("boom"))
	if code != http.StatusInternalServerError || body["message"] != PesanGalatUmum {
		t.Fatalf("dapat %d %v", code, body["message"])
	}
}

// tangkapLog mengarahkan logger galat tak terduga ke buffer selama satu test.
func tangkapLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	l := logger.New()
	l.SetOutput(&buf)
	SetLogger(l)
	t.Cleanup(func() { SetLogger(nil) })
	return &buf
}

func TestError_GalatUmumDicatat(t *testing.T) {
	buf := tangkapLog(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/category/create", nil)

	Error(ctx, fmt.Errorf("storage: invalid_grant"))

	out := buf.String()
	if !strings.Contains(out, "storage: invalid_grant") || !strings.Contains(out, "/v1/category/create") || !strings.Contains(out, `"level":"error"`) {
		t.Fatalf("log tidak memuat penyebab/path: %s", out)
	}
	if len(ctx.Errors) != 1 {
		t.Fatalf("ctx.Errors = %d, harap 1", len(ctx.Errors))
	}
	// Klien tetap hanya menerima pesan umum.
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != PesanGalatUmum {
		t.Fatalf("pesan ke klien bocor: %v", body["message"])
	}
}

func TestError_AppErrorTidakDicatat(t *testing.T) {
	buf := tangkapLog(t)
	code, _ := kirim(t, errors.NewNotFound("Data tidak ditemukan."))
	if code != http.StatusNotFound || buf.Len() != 0 {
		t.Fatalf("dapat %d, log=%q (harap kosong)", code, buf.String())
	}
}
