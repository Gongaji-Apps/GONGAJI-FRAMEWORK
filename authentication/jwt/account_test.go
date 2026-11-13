package jwt

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

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

// ServiceModule: role di luar OpenRoles wajib memegang izin modul (atau wildcard).
func TestAccountStrategy_ServiceModule(t *testing.T) {
	izin := map[string]map[string]bool{
		"USER":          {},
		"MENTOR":        {"ngaji.meeting.manage": true},
		"ADMINISTRATOR": {"article.article.manage": true},
		"SUPERADMIN":    {"*": true},
		"ARTICLEX":      {"articlex.sesuatu": true},
	}
	jalankan := func(role string, mod func(*AccountStrategyConfig)) error {
		s, token := accountStrategy(t, map[string]*Account{"u-1": {UUID: "u-1", Role: role}}, nil, func(c *AccountStrategyConfig) {
			c.SingleDeviceRoles = []string{}
			c.ServiceModule = "article"
			c.Permissions = func(ctx context.Context, r string) (map[string]bool, error) { return izin[r], nil }
			if mod != nil {
				mod(c)
			}
		})
		_, err := s.Authenticate(context.Background(), token)
		return err
	}

	for _, role := range []string{"USER", "ADMINISTRATOR", "SUPERADMIN"} {
		if err := jalankan(role, nil); err != nil {
			t.Errorf("%s harus lolos di modul article: %v", role, err)
		}
	}
	for _, role := range []string{"MENTOR", "ARTICLEX"} {
		err := jalankan(role, nil)
		var appErr *errors.AppError
		if !stderrors.As(err, &appErr) || appErr.Code != errors.Forbidden {
			t.Errorf("%s tanpa izin article.* harus 403, dapat %v", role, err)
		}
	}

	// OpenRoles kosong (bukan nil): USER pun butuh izin modul.
	if err := jalankan("USER", func(c *AccountStrategyConfig) { c.OpenRoles = []string{} }); err == nil {
		t.Error("OpenRoles kosong: USER tanpa izin modul harus ditolak")
	}
}

func TestNewAccountStrategy_ServiceModuleButuhPermissions(t *testing.T) {
	_, err := NewAccountStrategy(AccountStrategyConfig{
		Secret:        []byte("rahasia-uji"),
		Lookup:        func(ctx context.Context, uuid string) (*Account, error) { return nil, nil },
		ServiceModule: "article",
	})
	if err == nil {
		t.Error("ServiceModule tanpa Permissions harus ditolak saat konstruksi")
	}
}

// sesiStrategy: strategi dengan Sessions terpasang + token ber-klaim sid.
func sesiStrategy(t *testing.T, account *Account, sessions map[string]*Session, lookupErr error) (*Strategy, string) {
	t.Helper()
	s, _ := accountStrategy(t, map[string]*Account{"u-1": account}, nil, func(c *AccountStrategyConfig) {
		c.Sessions = func(ctx context.Context, id string) (*Session, error) {
			if lookupErr != nil {
				return nil, lookupErr
			}
			return sessions[id], nil
		}
	})
	token, err := s.Manager.Generate(Claims{SubjectUUID: "u-1", Extra: map[string]any{ClaimSessionID: "s-1", ClaimRole: "KLAIM-TIDAK-DIPERCAYA"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, token
}

func TestAccountStrategy_Sesi_RoleSesiMenggantikanRoleAkun(t *testing.T) {
	// Kolom lama akun = USER dengan token lain (login CMS tidak menimpa token aplikasi):
	// sesi CMS tetap sah dan role request = role sesi, bukan role akun atau klaim token.
	sessions := map[string]*Session{}
	s, token := sesiStrategy(t, &Account{UUID: "u-1", Role: "USER", Token: ptr("token-aplikasi")}, sessions, nil)
	sessions["s-1"] = &Session{UserUUID: "u-1", Role: "ADMINISTRATOR", Context: ContextCMS, TokenHash: HashToken(token), MemberActive: true}

	claims, err := s.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "ADMINISTRATOR" {
		t.Errorf("role = %q, mau role sesi ADMINISTRATOR", claims.Role)
	}
}

func TestAccountStrategy_Sesi_Ditolak(t *testing.T) {
	kasus := []struct {
		nama  string
		ubah  func(*Session)
		pesan string
	}{
		{"dicabut (logout)", func(x *Session) { x.Revoked = true }, MsgSessionEnded},
		{"dicabut karena login baru", func(x *Session) { x.Revoked, x.Superseded = true, true }, MsgOtherDevice},
		{"kedaluwarsa", func(x *Session) { x.ExpiresAt = time.Now().Add(-time.Minute) }, MsgSessionEnded},
		{"milik akun lain", func(x *Session) { x.UserUUID = "u-2" }, MsgSessionEnded},
		{"hash token beda", func(x *Session) { x.TokenHash = HashToken("token-lain") }, MsgSessionEnded},
		{"role sesi sudah dicabut dari akun", func(x *Session) { x.MemberActive = false }, MsgSessionEnded},
	}
	for _, k := range kasus {
		sessions := map[string]*Session{}
		s, token := sesiStrategy(t, &Account{UUID: "u-1", Role: "ADMINISTRATOR"}, sessions, nil)
		sesi := &Session{UserUUID: "u-1", Role: "ADMINISTRATOR", TokenHash: HashToken(token), MemberActive: true}
		k.ubah(sesi)
		sessions["s-1"] = sesi
		_, err := s.Authenticate(context.Background(), token)
		if err == nil {
			t.Errorf("%s: harus ditolak", k.nama)
			continue
		}
		wantUnauthorized(t, err, k.pesan)
	}

	// sesi tidak ada
	s, token := sesiStrategy(t, &Account{UUID: "u-1", Role: "ADMINISTRATOR"}, map[string]*Session{}, nil)
	_, err := s.Authenticate(context.Background(), token)
	wantUnauthorized(t, err, MsgSessionEnded)
}

func TestAccountStrategy_Sesi_GalatLookupTidakDianggapSah(t *testing.T) {
	s, token := sesiStrategy(t, &Account{UUID: "u-1", Role: "ADMINISTRATOR"}, nil, stderrors.New("db mati"))
	if _, err := s.Authenticate(context.Background(), token); err == nil || err.Error() != "db mati" {
		t.Errorf("galat lookup sesi harus diteruskan, dapat %v", err)
	}
}

func TestAccountStrategy_Sesi_JalurLamaTetapBerlaku(t *testing.T) {
	// Token TANPA sid pada strategi ber-Sessions → single-device lama.
	accounts := map[string]*Account{"u-1": {UUID: "u-1", Role: "USER", Token: ptr("token-lama")}}
	s, tokenLama := accountStrategy(t, accounts, nil, func(c *AccountStrategyConfig) {
		c.Sessions = func(context.Context, string) (*Session, error) { t.Error("Sessions tak boleh dipanggil"); return nil, nil }
	})
	_, err := s.Authenticate(context.Background(), tokenLama)
	wantUnauthorized(t, err, MsgOtherDevice)

	// Token ber-sid pada strategi TANPA Sessions (service belum naik versi) → jalur lama,
	// role dari akun; lolos bila token sama dengan kolom lama.
	s2, _ := accountStrategy(t, accounts, nil, nil)
	tokenSesi, err := s2.Manager.Generate(Claims{SubjectUUID: "u-1", Extra: map[string]any{ClaimSessionID: "s-1"}})
	if err != nil {
		t.Fatal(err)
	}
	accounts["u-1"].Token = &tokenSesi
	claims, err := s2.Authenticate(context.Background(), tokenSesi)
	if err != nil {
		t.Fatalf("token ber-sesi harus diterima service lama: %v", err)
	}
	if claims.Role != "USER" {
		t.Errorf("role = %q, mau role akun", claims.Role)
	}
}
