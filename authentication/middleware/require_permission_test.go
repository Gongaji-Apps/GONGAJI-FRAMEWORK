package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
	"github.com/gin-gonic/gin"
)

func runPerm(t *testing.T, codes map[string]bool, required string) int {
	t.Helper()
	r := gin.New()
	// Inject permission codes ke request context (meniru Auth).
	r.Use(func(c *gin.Context) {
		if codes != nil {
			c.Request = c.Request.WithContext(contextx.WithPermissionCodes(c.Request.Context(), codes))
		}
		c.Next()
	})
	r.GET("/x", RequirePermission(required), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRequirePermission_Allowed(t *testing.T) {
	if code := runPerm(t, map[string]bool{"ngaji.level.manage": true}, "ngaji.level.manage"); code != http.StatusOK {
		t.Fatalf("punya izin harus 200, got %d", code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	if code := runPerm(t, map[string]bool{"other": true}, "ngaji.level.manage"); code != http.StatusForbidden {
		t.Fatalf("tanpa izin harus 403, got %d", code)
	}
}

func TestRequirePermission_NoCodes(t *testing.T) {
	if code := runPerm(t, nil, "ngaji.level.manage"); code != http.StatusForbidden {
		t.Fatalf("tanpa permission codes harus 403, got %d", code)
	}
}

// Wildcard: role super (mis. SUPERADMIN) lolos izin apa pun tanpa bergantung isi tabel.
func TestRequirePermission_Wildcard(t *testing.T) {
	if code := runPerm(t, map[string]bool{contextx.PermissionAll: true}, "article.article.manage"); code != http.StatusOK {
		t.Fatalf("pemegang wildcard harus 200, got %d", code)
	}
	if code := runPerm(t, map[string]bool{contextx.PermissionAll: false}, "article.article.manage"); code != http.StatusForbidden {
		t.Fatalf("wildcard bernilai false bukan izin, got %d", code)
	}
}
