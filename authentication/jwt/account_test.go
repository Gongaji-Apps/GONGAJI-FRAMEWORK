package jwt

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

func ptr[T any](v T) *T { return &v }

func accountStrategy(t *testing.T, accounts map[string]*Account, lookupErr error, mod func(*AccountStrategyConfig)) (*Strategy, string) {
	t.Helper()
	cfg := AccountStrategyConfig{
		Secret:     []byte("rahasia-uji"),
		BoundRoles: map[string]string{"ADMINISTRATOR_LEARNING_BRANCH": "Partner"},
		Lookup: func(ctx context.Context, uuid string) (*Account, error) {
			if lookupErr != nil {
				return nil, lookupErr
			}
			return accounts[uuid], nil
		},
	}
	if mod != nil {
		mod(&cfg)
	}
	s, err := NewAccountStrategy(cfg)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Manager.Generate(Claims{SubjectUUID: "u-1"})
	if err != nil {
		t.Fatal(err)
	}
	return s, token
}

func wantUnauthorized(t *testing.T, err error, msg string) {
	t.Helper()
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != errors.Unauthorized {
		t.Fatalf("mau 401, dapat %v", err)
	}
	if msg != "" && appErr.Message != msg {
		t.Errorf("pesan = %q, mau %q", appErr.Message, msg)
	}
}

func TestNewAccountStrategy_LookupWajib(t *testing.T) {
	if _, err := NewAccountStrategy(AccountStrategyConfig{Secret: []byte("x")}); err == nil {
		t.Error("tanpa Lookup harus error")
	}
}

func TestAccountStrategy_Berhasil(t *testing.T) {
	s, token := accountStrategy(t, map[string]*Account{"u-1": {
		UUID: "u-1", UserID: ptr(int64(7)), FullName: "Fulan", Email: "f@x.id", Role: "ADMINISTRATOR",
		Active: ptr(true), Tester: true, Extra: map[string]any{"k": "v"},
	}}, nil, nil)

	claims, err := s.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SubjectUUID != "u-1" || *claims.SubjectUserID != 7 || claims.SubjectFullName != "Fulan" ||
		claims.SubjectEmail != "f@x.id" || claims.Role != "ADMINISTRATOR" || !claims.SubjectTester || claims.Extra["k"] != "v" {
		t.Errorf("AuthClaims tidak lengkap: %+v", claims)
	}
	if claims.PermissionCodes == nil {
		t.Error("PermissionCodes harus map kosong, bukan nil")
	}
}

func TestAccountStrategy_AkunTidakAda(t *testing.T) {
	s, token := accountStrategy(t, map[string]*Account{}, nil, nil)
	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, MsgAccountMissing)

	s2, token2 := accountStrategy(t, nil, errors.NewNotFound("User tidak ditemukan"), nil)
	_, err = s2.Authenticate(context.Background(), token2)
	wantUnauthorized(t, err, MsgAccountMissing)
}

func TestAccountStrategy_GalatLookupLainDiteruskan(t *testing.T) {
	s, token := accountStrategy(t, nil, errors.NewInternalServerError("db mati"), nil)
	_, err := s.Authenticate(context.Background(), token)
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != errors.InternalServerError {
		t.Fatalf("galat DB harus diteruskan apa adanya, dapat %v", err)
	}
}

func TestAccountStrategy_SingleDevice(t *testing.T) {
	accounts := map[string]*Account{"u-1": {UUID: "u-1", Role: "USER", Token: ptr("token-lama")}}
	s, token := accountStrategy(t, accounts, nil, nil)

	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, MsgOtherDevice)

	accounts["u-1"].Token = &token
	if _, err := s.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("token cocok harus lolos: %v", err)
	}

	// ADMIN tidak dikenai single-device
	accounts["u-1"] = &Account{UUID: "u-1", Role: "ADMINISTRATOR", Token: ptr("beda")}
	if _, err := s.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("admin tidak boleh kena single-device: %v", err)
	}

	// Dimatikan dengan slice kosong
	accounts["u-1"] = &Account{UUID: "u-1", Role: "USER", Token: nil}
	s3, token3 := accountStrategy(t, accounts, nil, func(c *AccountStrategyConfig) { c.SingleDeviceRoles = []string{} })
	if _, err := s3.Authenticate(context.Background(), token3); err != nil {
		t.Fatalf("single-device nonaktif harus lolos: %v", err)
	}
}

func TestAccountStrategy_Nonaktif(t *testing.T) {
	s, token := accountStrategy(t, map[string]*Account{"u-1": {UUID: "u-1", Role: "ADMINISTRATOR", Active: ptr(false)}}, nil, nil)
	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, MsgInactive)
}

func TestAccountStrategy_RoleTerikat(t *testing.T) {
	accounts := map[string]*Account{"u-1": {UUID: "u-1", Role: "ADMINISTRATOR_LEARNING_BRANCH", Links: map[string]bool{"Partner": false}}}
	s, token := accountStrategy(t, accounts, nil, nil)
	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, "Akun Anda belum tertaut ke Partner manapun. Hubungi pengelola.")

	accounts["u-1"].Links["Partner"] = true
	if _, err := s.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("tertaut harus lolos: %v", err)
	}
}

func TestAccountStrategy_Permissions(t *testing.T) {
	s, token := accountStrategy(t, map[string]*Account{"u-1": {UUID: "u-1", Role: "MENTOR"}}, nil, func(c *AccountStrategyConfig) {
		c.Permissions = func(ctx context.Context, role string) (map[string]bool, error) {
			if role != "MENTOR" {
				t.Errorf("role = %q", role)
			}
			return map[string]bool{"level.read": true}, nil
		}
	})
	claims, err := s.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !claims.PermissionCodes["level.read"] {
		t.Errorf("PermissionCodes = %v", claims.PermissionCodes)
	}

	s2, token2 := accountStrategy(t, map[string]*Account{"u-1": {UUID: "u-1", Role: "MENTOR"}}, nil, func(c *AccountStrategyConfig) {
		c.Permissions = func(ctx context.Context, role string) (map[string]bool, error) {
			return nil, errors.NewServiceUnavailable("RBAC tidak tersedia")
		}
	})
	if _, err := s2.Authenticate(context.Background(), token2); err == nil {
		t.Error("galat loader izin harus diteruskan")
	}
}

func TestAccountStrategy_TokenTanpaSubject(t *testing.T) {
	s, _ := accountStrategy(t, map[string]*Account{}, nil, nil)
	m, _ := New(Config{Secret: []byte("rahasia-uji"), DefaultTTL: DefaultAccountTTL})
	// Manager.Generate menolak SubjectUUID kosong, jadi uji lewat token yang subject-nya tidak dikenal
	token, _ := m.Generate(Claims{SubjectUUID: "tidak-ada"})
	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, MsgAccountMissing)
}
