package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouterWithRole(roleCode any, roles []string) *gin.Engine {
	r := gin.New()
	r.GET("/protected",
		// Simulate Auth() having set role_code in the context.
		func(c *gin.Context) {
			if roleCode != nil {
				c.Set("role_code", roleCode)
			}
			c.Next()
		},
		AuthorizeRoles(roles...),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) },
	)
	return r
}

func runRequest(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestAuthorizeRoles_AllowedRoleMatches(t *testing.T) {
	r := newRouterWithRole("ADMIN", []string{"ADMIN", "STAFF"})
	w := runRequest(t, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthorizeRoles_CaseInsensitive(t *testing.T) {
	r := newRouterWithRole("admin", []string{"ADMIN"})
	w := runRequest(t, r)
	if w.Code != http.StatusOK {
		t.Errorf("case-insensitive match should pass; status = %d", w.Code)
	}
}

func TestAuthorizeRoles_RoleNotInList(t *testing.T) {
	r := newRouterWithRole("USER", []string{"ADMIN", "STAFF"})
	w := runRequest(t, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestAuthorizeRoles_MissingRoleCode(t *testing.T) {
	r := newRouterWithRole(nil, []string{"ADMIN"})
	w := runRequest(t, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestAuthorizeRoles_RoleCodeWrongType(t *testing.T) {
	// role_code set to non-string (programmer error in upstream code).
	r := newRouterWithRole(42, []string{"ADMIN"})
	w := runRequest(t, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-string role_code should be rejected; status = %d", w.Code)
	}
}

func TestAuthorizeRoles_EmptyRoleCode(t *testing.T) {
	r := newRouterWithRole("", []string{"ADMIN"})
	w := runRequest(t, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("empty role_code should be rejected; status = %d", w.Code)
	}
}

func TestAuthorizeRoles_EmptyRolesAlwaysRejects(t *testing.T) {
	// Defensive check: AuthorizeRoles() with no arguments must not silently allow everyone.
	r := newRouterWithRole("ADMIN", []string{})
	w := runRequest(t, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("empty role list should reject; status = %d", w.Code)
	}
}

func TestAuthorizeRoles_BodyHasFrameworkErrorShape(t *testing.T) {
	r := newRouterWithRole("USER", []string{"ADMIN"})
	w := runRequest(t, r)
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if status, _ := body["status"].(bool); status {
		t.Errorf("status should be false in error response")
	}
	if msg, _ := body["message"].(string); msg == "" {
		t.Errorf("message should be populated")
	}
}
