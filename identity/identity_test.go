package identity

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// dryDB: dialek postgres tanpa koneksi — query dibangun tapi tidak dikirim.
func dryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable"}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry db: %v", err)
	}
	return db
}

func TestFindByUUIDSelectsOnlyAccountColumns(t *testing.T) {
	var users []UserAccount
	stmt := FindByUUID(dryDB(t), "abc").Find(&users).Statement
	sql := stmt.SQL.String()

	for _, want := range []string{`FROM "public"."user_mstr"`, `user_uuid = $1`, `LIMIT $2`} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL tidak memuat %q: %s", want, sql)
		}
	}
	for _, col := range accountColumns {
		if !strings.Contains(sql, col) {
			t.Errorf("SQL tidak memilih kolom %q: %s", col, sql)
		}
	}
	for _, secret := range []string{"*", "user_password", "user_otp", "user_fcm", "user_sub", "user_telp", "user_address"} {
		if strings.Contains(sql, secret) {
			t.Errorf("SQL tidak boleh membaca %q: %s", secret, sql)
		}
	}
	if len(stmt.Vars) == 0 || stmt.Vars[0] != "abc" {
		t.Errorf("parameter uuid = %v, mau abc", stmt.Vars)
	}
}

func TestLookupNotFoundIsNilNil(t *testing.T) {
	// DryRun tidak mengembalikan baris → dianggap akun tidak ada (validator → 401).
	account, err := Lookup(dryDB(t))(context.Background(), "tidak-ada")
	if err != nil || account != nil {
		t.Fatalf("Lookup = (%v, %v), mau (nil, nil)", account, err)
	}
}

func TestAccountMapping(t *testing.T) {
	id := int64(7)
	email, role, token := "a@b.c", "USER", "tok"
	yes, no := true, false

	got := UserAccount{UserID: &id, UUID: "u1", FullName: "Fulan", Email: &email, Role: &role,
		Token: &token, Active: &no, Tester: &yes}.Account()

	if got.UUID != "u1" || got.FullName != "Fulan" || got.Email != email || got.Role != role {
		t.Errorf("identitas salah: %+v", got)
	}
	if got.UserID == nil || *got.UserID != 7 || got.Token == nil || *got.Token != token {
		t.Errorf("UserID/Token salah: %+v", got)
	}
	if got.Active == nil || *got.Active || !got.Tester {
		t.Errorf("Active/Tester salah: active=%v tester=%v", got.Active, got.Tester)
	}

	empty := UserAccount{UUID: "u2"}.Account()
	if empty.Email != "" || empty.Role != "" || empty.Tester || empty.Active != nil || empty.Token != nil {
		t.Errorf("kolom kosong harus jadi nilai nol (Active nil = aktif): %+v", empty)
	}
}

func TestUserAccountNeverSerialized(t *testing.T) {
	token := "rahasia"
	out, err := json.Marshal(UserAccount{UUID: "u1", FullName: "Fulan", Token: &token})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "{}" {
		t.Errorf("UserAccount bukan untuk respons API, JSON harus {}: %s", out)
	}
}
