// Package jwt provides JWT token generation and verification helpers.
//
// The Manager handles signing/parsing primitives only — it does not
// know about your user database or session model. For request-time
// authentication that combines JWT verification with app-specific
// checks (user lookup, single-device enforcement, active flag, etc.),
// use Strategy from this package together with a custom Validator.
//
// Example: token generation
//
//	m := jwt.New(jwt.Config{
//	    Secret: []byte(os.Getenv("JWT_SECRET")),
//	    Issuer: "gongaji-api",
//	})
//
//	token, err := m.Generate(jwt.Claims{
//	    SubjectUUID: user.UUID,
//	    ExpiresAt:   time.Now().Add(24 * time.Hour),
//	    Extra:       map[string]any{"role": "ADMIN"},
//	})
//
// Example: parse + verify
//
//	claims, err := m.Parse(rawToken)
//	if err != nil {
//	    return errors.NewUnauthorized("Token tidak valid.")
//	}
package jwt

import (
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned by Parse when the token cannot be verified.
var ErrInvalidToken = errors.New("jwt: token is invalid or expired")

// Config controls the Manager.
type Config struct {
	// Secret is the HMAC key used to sign and verify tokens.
	// Required.
	Secret []byte

	// Issuer, when set, is embedded as the `iss` claim on Generate
	// and validated on Parse. Empty disables both.
	Issuer string

	// DefaultTTL is used by Generate when Claims.ExpiresAt is zero.
	// If both are zero, Generate returns an error.
	DefaultTTL time.Duration
}

// Manager generates and parses HMAC-SHA256-signed JWT tokens.
type Manager struct {
	cfg Config
}

// Claims is a framework-friendly representation of the standard JWT
// claims plus an Extra map for custom payload fields.
//
// On Generate:
//   - SubjectUUID is required and emitted as `sub`.
//   - ExpiresAt is required (or supply Config.DefaultTTL); emitted as `exp`.
//   - IssuedAt defaults to time.Now() when zero; emitted as `iat`.
//   - NotBefore is emitted as `nbf` only when non-zero.
//   - Issuer overrides Config.Issuer when non-empty; emitted as `iss`.
//   - Extra fields are merged into the payload. Reserved claim names
//     (sub, exp, iat, nbf, iss, aud, jti) are silently ignored to
//     prevent caller from clobbering them.
//
// On Parse, every populated standard claim is read back and Extra
// holds any non-reserved fields.
type Claims struct {
	SubjectUUID string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	NotBefore   time.Time
	Issuer      string
	Extra       map[string]any
}

// New constructs a Manager.
func New(cfg Config) (*Manager, error) {
	if len(cfg.Secret) == 0 {
		return nil, errors.New("jwt: Secret is required")
	}
	return &Manager{cfg: cfg}, nil
}

// reserved holds standard JWT claim names. Generate strips these
// from Claims.Extra to prevent caller-side accidents.
var reserved = map[string]struct{}{
	"sub": {}, "exp": {}, "iat": {}, "nbf": {}, "iss": {}, "aud": {}, "jti": {},
}

// Generate signs the given Claims and returns the encoded token string.
func (m *Manager) Generate(c Claims) (string, error) {
	if c.SubjectUUID == "" {
		return "", errors.New("jwt: SubjectUUID is required")
	}

	now := time.Now()
	exp := c.ExpiresAt
	if exp.IsZero() {
		if m.cfg.DefaultTTL <= 0 {
			return "", errors.New("jwt: ExpiresAt or Config.DefaultTTL is required")
		}
		exp = now.Add(m.cfg.DefaultTTL)
	}
	iat := c.IssuedAt
	if iat.IsZero() {
		iat = now
	}

	payload := gojwt.MapClaims{
		"sub": c.SubjectUUID,
		"exp": exp.Unix(),
		"iat": iat.Unix(),
	}
	if !c.NotBefore.IsZero() {
		payload["nbf"] = c.NotBefore.Unix()
	}
	if iss := c.Issuer; iss != "" {
		payload["iss"] = iss
	} else if m.cfg.Issuer != "" {
		payload["iss"] = m.cfg.Issuer
	}
	for k, v := range c.Extra {
		if _, isReserved := reserved[k]; isReserved {
			continue
		}
		payload[k] = v
	}

	tok := gojwt.NewWithClaims(gojwt.SigningMethodHS256, payload)
	signed, err := tok.SignedString(m.cfg.Secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// Parse verifies the token signature, expiration, and (when configured)
// issuer, and returns the decoded Claims.
//
// Parse returns ErrInvalidToken (wrappable) for any failure so callers
// can branch on errors.Is(err, jwt.ErrInvalidToken) without inspecting
// the string. The wrapped error includes the underlying reason.
func (m *Manager) Parse(raw string) (*Claims, error) {
	parser := gojwt.NewParser(gojwt.WithValidMethods([]string{"HS256"}))

	tok, err := parser.Parse(raw, func(t *gojwt.Token) (any, error) {
		return m.cfg.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !tok.Valid {
		return nil, ErrInvalidToken
	}
	mc, ok := tok.Claims.(gojwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected claims shape", ErrInvalidToken)
	}

	out := &Claims{Extra: map[string]any{}}

	if v, ok := mc["sub"].(string); ok {
		out.SubjectUUID = v
	}
	if v, ok := mc["iss"].(string); ok {
		out.Issuer = v
	}
	if t, ok := unixToTime(mc["exp"]); ok {
		out.ExpiresAt = t
	}
	if t, ok := unixToTime(mc["iat"]); ok {
		out.IssuedAt = t
	}
	if t, ok := unixToTime(mc["nbf"]); ok {
		out.NotBefore = t
	}

	if m.cfg.Issuer != "" && out.Issuer != m.cfg.Issuer {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalidToken)
	}

	for k, v := range mc {
		if _, isReserved := reserved[k]; isReserved {
			continue
		}
		out.Extra[k] = v
	}

	return out, nil
}

func unixToTime(v any) (time.Time, bool) {
	switch n := v.(type) {
	case float64:
		return time.Unix(int64(n), 0), true
	case int64:
		return time.Unix(n, 0), true
	case int:
		return time.Unix(int64(n), 0), true
	}
	return time.Time{}, false
}
