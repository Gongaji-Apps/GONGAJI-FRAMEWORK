package jwt

import (
	"context"
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
	authutils "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/utils"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/gin-gonic/gin"
)

// Validator is the application hook that runs after JWT signature and
// expiry have been verified. Use it to perform domain-specific checks:
//
//   - Look up the user in your DB
//   - Verify the token is still associated with this user
//     (single-device enforcement)
//   - Check user_active flag, role, permissions
//   - Populate AuthClaims (SubjectUUID, Role, PermissionCodes, Extra)
//
// Return any *errors.AppError to short-circuit auth with that response.
//
// The rawToken is provided so you can compare it to a stored token
// per user (single-device pattern).
type Validator func(ctx context.Context, claims *Claims, rawToken string) (*middleware.AuthClaims, error)

// Strategy implements middleware.AuthStrategy by delegating signature/
// expiry checks to Manager and app-specific checks to Validator.
//
// If Validator is nil, the strategy returns AuthClaims with only
// SubjectUUID populated from the token.
//
// Strategy.Name defaults to "JWT" but is overridable for cases where
// multiple JWT signers coexist (e.g. internal vs partner tokens).
type Strategy struct {
	Manager      *Manager
	Validator    Validator
	StrategyName string // optional override; defaults to "JWT"
}

// Compile-time check.
var _ middleware.AuthStrategy = (*Strategy)(nil)

// Name returns the strategy identifier surfaced into AuthClaims.
func (s *Strategy) Name() string {
	if s.StrategyName != "" {
		return s.StrategyName
	}
	return "JWT"
}

// CanHandle returns true when the request carries an "Authorization: Bearer ..."
// header.
func (s *Strategy) CanHandle(c *gin.Context) bool {
	return strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ")
}

// ExtractToken pulls the bearer token from the request.
func (s *Strategy) ExtractToken(c *gin.Context) (string, error) {
	tok, err := authutils.ExtractBearer(c)
	if err != nil {
		return "", errors.NewUnauthorized("[Unauthorized] Token bearer tidak ditemukan.")
	}
	return tok, nil
}

// Authenticate parses the JWT, runs the app-level Validator, and
// returns AuthClaims for the request.
func (s *Strategy) Authenticate(ctx context.Context, raw string) (*middleware.AuthClaims, error) {
	if s.Manager == nil {
		return nil, errors.NewInternalServerError("[Internal Server Error] JWT manager tidak dikonfigurasi.")
	}

	claims, err := s.Manager.Parse(raw)
	if err != nil {
		return nil, errors.NewUnauthorized("[Unauthorized] Token tidak valid atau sudah kadaluarsa. Mohon login ulang.")
	}

	if s.Validator != nil {
		return s.Validator(ctx, claims, raw)
	}

	// Default: no app-level checks. Just propagate the subject.
	return &middleware.AuthClaims{
		SubjectUUID: claims.SubjectUUID,
	}, nil
}
