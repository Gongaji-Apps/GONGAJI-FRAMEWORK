package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSuccessWithCache_Header(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	SuccessWithCache(ctx, map[string]any{"a": 1}, "v1-abc", 90*time.Second)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "public, max-age=90" {
		t.Errorf("Cache-Control = %q, want %q", got, "public, max-age=90")
	}
	if got := w.Header().Get("ETag"); got != "v1-abc" {
		t.Errorf("ETag = %q", got)
	}
	if w.Header().Get("Expires") == "" {
		t.Error("Expires kosong")
	}
}
