package jwt

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/gin-gonic/gin"
)

func newGinCtx(authHeader string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	c.Request = req
	return c
}

func TestStrategy_Name(t *testing.T) {
	s := &Strategy{}
	if got := s.Name(); got != "JWT" {
		t.Errorf("default name = %q, want JWT", got)
	}
	s.StrategyName = "JWT_PARTNER"
	if got := s.Name(); got != "JWT_PARTNER" {
		t.Errorf("override name = %q", got)
	}
}

func TestStrategy_CanHandle(t *testing.T) {
	s := &Strategy{}
	if s.CanHandle(newGinCtx("")) {
		t.Error("should not handle missing header")
	}
	if s.CanHandle(newGinCtx("Basic abc")) {
		t.Error("should not handle non-bearer scheme")
	}
	if !s.CanHandle(newGinCtx("Bearer abc")) {
		t.Error("should handle Bearer header")
	}
}

func TestStrategy_ExtractToken(t *testing.T) {
	s := &Strategy{}
	tok, err := s.ExtractToken(newGinCtx("Bearer my-token"))
	if err != nil {
		t.Fatal(err)
	}
	if tok != "my-token" {
		t.Errorf("token = %q", tok)
	}

	if _, err := s.ExtractToken(newGinCtx("")); err == nil {
		t.Error("missing header should error")
	}
}

func TestStrategy_Authenticate_NoManager(t *testing.T) {
	s := &Strategy{}
	if _, err := s.Authenticate(context.Background(), "any"); err == nil {
		t.Error("expected error when Manager is nil")
	}
}

func TestStrategy_Authenticate_DefaultValidator(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s")})
	tok, _ := m.Generate(Claims{
		SubjectUUID: "user-42",
		ExpiresAt:   time.Now().Add(time.Hour),
		Extra:       map[string]any{"role": "ADMIN"},
	})

	s := &Strategy{Manager: m}

	claims, err := s.Authenticate(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SubjectUUID != "user-42" {
		t.Errorf("SubjectUUID = %q", claims.SubjectUUID)
	}
}

func TestStrategy_Authenticate_BadToken(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s")})
	s := &Strategy{Manager: m}

	if _, err := s.Authenticate(context.Background(), "not.a.token"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestStrategy_Authenticate_CustomValidator(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s")})
	tok, _ := m.Generate(Claims{
		SubjectUUID: "user-42",
		ExpiresAt:   time.Now().Add(time.Hour),
		Extra:       map[string]any{"role": "ADMIN"},
	})

	var seenRaw string
	var seenClaims *Claims

	s := &Strategy{
		Manager: m,
		Validator: func(ctx context.Context, c *Claims, raw string) (*middleware.AuthClaims, error) {
			seenRaw = raw
			seenClaims = c
			role, _ := c.Extra["role"].(string)
			return &middleware.AuthClaims{
				SubjectUUID: c.SubjectUUID,
				Role:        role,
				PermissionCodes: map[string]bool{
					"ANYTHING": true,
				},
			}, nil
		},
	}

	claims, err := s.Authenticate(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	if seenRaw != tok {
		t.Errorf("validator did not receive raw token")
	}
	if seenClaims == nil || seenClaims.SubjectUUID != "user-42" {
		t.Errorf("validator did not receive parsed claims")
	}
	if claims.Role != "ADMIN" {
		t.Errorf("role = %q", claims.Role)
	}
	if !claims.PermissionCodes["ANYTHING"] {
		t.Errorf("permission not propagated")
	}
}

func TestStrategy_Authenticate_ValidatorRejects(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s")})
	tok, _ := m.Generate(Claims{
		SubjectUUID: "user-42",
		ExpiresAt:   time.Now().Add(time.Hour),
	})

	s := &Strategy{
		Manager: m,
		Validator: func(ctx context.Context, c *Claims, raw string) (*middleware.AuthClaims, error) {
			return nil, errors.NewUnauthorized("user nonaktif")
		},
	}

	_, err := s.Authenticate(context.Background(), tok)
	if err == nil {
		t.Fatal("expected error from validator")
	}
	appErr, ok := err.(*errors.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != errors.Unauthorized {
		t.Errorf("code = %q", appErr.Code)
	}
}
