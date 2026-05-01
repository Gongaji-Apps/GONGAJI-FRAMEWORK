package jwt

import (
	"errors"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

func newManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(Config{Secret: []byte("test-secret"), Issuer: "test-iss"})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNew_RequiresSecret(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestGenerate_RoundTrip(t *testing.T) {
	m := newManager(t)
	exp := time.Now().Add(1 * time.Hour).Truncate(time.Second)

	tok, err := m.Generate(Claims{
		SubjectUUID: "user-1",
		ExpiresAt:   exp,
		Extra: map[string]any{
			"role":  "ADMIN",
			"scope": []string{"read", "write"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := m.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.SubjectUUID != "user-1" {
		t.Errorf("sub = %q", parsed.SubjectUUID)
	}
	if !parsed.ExpiresAt.Equal(exp) {
		t.Errorf("exp = %v, want %v", parsed.ExpiresAt, exp)
	}
	if parsed.Issuer != "test-iss" {
		t.Errorf("iss = %q, want %q", parsed.Issuer, "test-iss")
	}
	if parsed.Extra["role"] != "ADMIN" {
		t.Errorf("extra[role] = %v", parsed.Extra["role"])
	}
}

func TestGenerate_RequiresSubject(t *testing.T) {
	m := newManager(t)
	if _, err := m.Generate(Claims{ExpiresAt: time.Now().Add(time.Hour)}); err == nil {
		t.Fatal("expected error for missing SubjectUUID")
	}
}

func TestGenerate_RequiresExpiry(t *testing.T) {
	m := newManager(t)
	if _, err := m.Generate(Claims{SubjectUUID: "u"}); err == nil {
		t.Fatal("expected error for missing ExpiresAt and DefaultTTL")
	}
}

func TestGenerate_DefaultTTL(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s"), DefaultTTL: 5 * time.Minute})
	tok, err := m.Generate(Claims{SubjectUUID: "u"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := m.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(parsed.ExpiresAt); d <= 0 || d > 5*time.Minute+time.Second {
		t.Errorf("default TTL not applied; expires in %v", d)
	}
}

func TestGenerate_ReservedExtraIgnored(t *testing.T) {
	m := newManager(t)
	tok, err := m.Generate(Claims{
		SubjectUUID: "u",
		ExpiresAt:   time.Now().Add(time.Hour),
		Extra: map[string]any{
			"sub":  "hacker", // attempt to override
			"role": "USER",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := m.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.SubjectUUID != "u" {
		t.Errorf("Reserved 'sub' override should be ignored; got %q", parsed.SubjectUUID)
	}
	if parsed.Extra["role"] != "USER" {
		t.Errorf("non-reserved field missing")
	}
	if _, ok := parsed.Extra["sub"]; ok {
		t.Errorf("sub should not appear in Extra after parse")
	}
}

func TestParse_Expired(t *testing.T) {
	m := newManager(t)
	tok, err := m.Generate(Claims{
		SubjectUUID: "u",
		ExpiresAt:   time.Now().Add(-1 * time.Minute), // already expired
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Parse(tok)
	if err == nil {
		t.Fatal("expected expired error")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("err should wrap ErrInvalidToken, got %v", err)
	}
}

func TestParse_BadSignature(t *testing.T) {
	m1, _ := New(Config{Secret: []byte("alpha")})
	m2, _ := New(Config{Secret: []byte("beta")})

	tok, _ := m1.Generate(Claims{SubjectUUID: "u", ExpiresAt: time.Now().Add(time.Hour)})

	if _, err := m2.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("cross-secret parse should fail with ErrInvalidToken, got %v", err)
	}
}

func TestParse_WrongAlg_RejectedAtParser(t *testing.T) {
	// Signing with HS512 even though Manager only allows HS256.
	tok := gojwt.NewWithClaims(gojwt.SigningMethodHS512, gojwt.MapClaims{
		"sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	m := newManager(t)
	if _, err := m.Parse(signed); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("wrong-alg token should be rejected, got %v", err)
	}
}

func TestParse_IssuerMismatch(t *testing.T) {
	signer, _ := New(Config{Secret: []byte("s"), Issuer: "service-a"})
	verifier, _ := New(Config{Secret: []byte("s"), Issuer: "service-b"})

	tok, _ := signer.Generate(Claims{SubjectUUID: "u", ExpiresAt: time.Now().Add(time.Hour)})

	_, err := verifier.Parse(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("mismatched issuer should fail, got %v", err)
	}
	if !strings.Contains(err.Error(), "issuer") {
		t.Errorf("error should mention issuer, got %v", err)
	}
}

func TestParse_NotBefore(t *testing.T) {
	m := newManager(t)
	tok, err := m.Generate(Claims{
		SubjectUUID: "u",
		ExpiresAt:   time.Now().Add(time.Hour),
		NotBefore:   time.Now().Add(time.Hour), // not yet valid
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("nbf in future should reject, got %v", err)
	}
}

func TestParse_GarbageString(t *testing.T) {
	m := newManager(t)
	if _, err := m.Parse("not.a.token"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("garbage should fail, got %v", err)
	}
}

func TestGenerate_PerCallIssuerOverride(t *testing.T) {
	m, _ := New(Config{Secret: []byte("s"), Issuer: "default"})
	tok, _ := m.Generate(Claims{
		SubjectUUID: "u",
		ExpiresAt:   time.Now().Add(time.Hour),
		Issuer:      "override",
	})

	// Verifier expects "override"
	v, _ := New(Config{Secret: []byte("s"), Issuer: "override"})
	if _, err := v.Parse(tok); err != nil {
		t.Errorf("override issuer should be embedded; got %v", err)
	}
}
