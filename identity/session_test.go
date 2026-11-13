package identity

import (
	"strings"
	"testing"
	"time"
)

func TestSessionQuery(t *testing.T) {
	var rows []sessionRow
	stmt := sessionQuery(dryDB(t), "s-1").Find(&rows).Statement
	sql := stmt.SQL.String()

	for _, want := range []string{
		"FROM auth.user_session AS s",
		"LEFT JOIN auth.user_role AS r ON r.user_uuid = s.user_uuid AND r.role_code = s.role_code",
		"r.urole_active AS member_active",
		"s.session_id = $1",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL tidak memuat %q: %s", want, sql)
		}
	}
	// Token asli tidak pernah ada di tabel; yang dibaca hanya hash-nya.
	if !strings.Contains(sql, "session_token_hash") {
		t.Errorf("hash token harus dibaca: %s", sql)
	}
	if len(stmt.Vars) < 1 || stmt.Vars[0] != "s-1" {
		t.Errorf("parameter id sesi = %v", stmt.Vars)
	}
}

func TestSessionRowMapping(t *testing.T) {
	ya, tidak := true, false
	kini := time.Now()
	loginBaru, logout := RevokedByNewLogin, "logout"

	hidup := sessionRow{UserUUID: "u-1", Role: "ADMIN", Context: "CMS", TokenHash: "h", ExpiredDate: kini, MemberActive: &ya}.session()
	if hidup.Revoked || hidup.Superseded || !hidup.MemberActive || hidup.Role != "ADMIN" || hidup.Context != "CMS" || !hidup.ExpiresAt.Equal(kini) {
		t.Errorf("sesi hidup salah dipetakan: %+v", hidup)
	}

	if s := (sessionRow{RevokedDate: &kini, RevokedReason: &loginBaru, MemberActive: &ya}).session(); !s.Revoked || !s.Superseded {
		t.Errorf("dicabut karena login baru harus Superseded: %+v", s)
	}
	if s := (sessionRow{RevokedDate: &kini, RevokedReason: &logout, MemberActive: &ya}).session(); !s.Revoked || s.Superseded {
		t.Errorf("logout bukan Superseded: %+v", s)
	}
	// Keanggotaan dihapus (LEFT JOIN → NULL) atau dinonaktifkan → tidak aktif.
	if s := (sessionRow{MemberActive: nil}).session(); s.MemberActive {
		t.Error("keanggotaan hilang harus MemberActive=false")
	}
	if s := (sessionRow{MemberActive: &tidak}).session(); s.MemberActive {
		t.Error("keanggotaan nonaktif harus MemberActive=false")
	}
}
