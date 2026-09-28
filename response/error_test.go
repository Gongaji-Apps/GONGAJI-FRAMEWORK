package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
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
