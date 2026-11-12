package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
)

func TestPermissionQuery(t *testing.T) {
	var rows []permissionRow
	stmt := permissionQuery(dryDB(t), "ADMINISTRATOR").Find(&rows).Statement
	sql := stmt.SQL.String()

	for _, want := range []string{`SELECT "permission_code"`, `FROM "auth"."role_permission"`, `role_code = $1`} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL tidak memuat %q: %s", want, sql)
		}
	}
	if len(stmt.Vars) != 1 || stmt.Vars[0] != "ADMINISTRATOR" {
		t.Errorf("parameter role = %v", stmt.Vars)
	}
}

// penghitung: loader palsu yang mencatat berapa kali tabel "dibaca".
type penghitung struct {
	n     int
	codes map[string]bool
	err   error
}

func (p *penghitung) load(context.Context, string) (map[string]bool, error) {
	p.n++
	return p.codes, p.err
}

func TestPermissionsCachePerRoleDenganTTL(t *testing.T) {
	sekarang := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	p := &penghitung{codes: map[string]bool{"article.article.manage": true}}
	loader := newPermissionLoader(PermissionConfig{TTL: time.Minute}, p.load, func() time.Time { return sekarang })

	for i := 0; i < 3; i++ {
		codes, err := loader(context.Background(), "ADMINISTRATOR")
		if err != nil || !codes["article.article.manage"] {
			t.Fatalf("izin tidak termuat: %v %v", codes, err)
		}
	}
	if p.n != 1 {
		t.Fatalf("dalam TTL tabel harus dibaca 1x, terbaca %d", p.n)
	}

	if _, _ = loader(context.Background(), "ADMIN"); p.n != 2 {
		t.Fatalf("role lain punya entri cache sendiri, terbaca %d", p.n)
	}

	// Izin dicabut di tabel: baru terlihat setelah TTL lewat.
	p.codes = map[string]bool{}
	sekarang = sekarang.Add(59 * time.Second)
	if codes, _ := loader(context.Background(), "ADMINISTRATOR"); !codes["article.article.manage"] {
		t.Fatal("sebelum TTL lewat cache masih berlaku")
	}
	sekarang = sekarang.Add(2 * time.Second)
	if codes, _ := loader(context.Background(), "ADMINISTRATOR"); codes["article.article.manage"] {
		t.Fatal("setelah TTL lewat pencabutan izin harus berlaku")
	}
}

func TestPermissionsRoleSuperSelaluWildcardTanpaBacaTabel(t *testing.T) {
	p := &penghitung{err: errors.New("tabel izin tidak ada")}
	loader := newPermissionLoader(PermissionConfig{}, p.load, time.Now)

	codes, err := loader(context.Background(), "SUPERADMIN")
	if err != nil || !codes[contextx.PermissionAll] {
		t.Fatalf("SUPERADMIN harus memegang wildcard: %v %v", codes, err)
	}
	if p.n != 0 {
		t.Fatalf("role super tak boleh bergantung tabel, terbaca %d", p.n)
	}

	// Slice kosong (bukan nil) mematikan role super.
	tanpa := newPermissionLoader(PermissionConfig{SuperRoles: []string{}}, (&penghitung{codes: map[string]bool{}}).load, time.Now)
	if codes, _ := tanpa(context.Background(), "SUPERADMIN"); codes[contextx.PermissionAll] {
		t.Fatal("SuperRoles kosong: SUPERADMIN tak boleh otomatis wildcard")
	}
}

// Tabel gagal dibaca: izin kosong (403 di RequirePermission), request tidak gagal,
// galat dilaporkan dan TIDAK di-cache.
func TestPermissionsGagalBacaTidakMematikanDanTidakDiCache(t *testing.T) {
	p := &penghitung{err: errors.New("relation auth.role_permission does not exist")}
	var dilaporkan []string
	loader := newPermissionLoader(PermissionConfig{
		OnError: func(role string, err error) { dilaporkan = append(dilaporkan, role) },
	}, p.load, time.Now)

	for i := 0; i < 2; i++ {
		codes, err := loader(context.Background(), "USER")
		if err != nil {
			t.Fatalf("galat tabel tak boleh menggagalkan request: %v", err)
		}
		if len(codes) != 0 {
			t.Fatalf("gagal baca = izin kosong, dapat %v", codes)
		}
	}
	if p.n != 2 || len(dilaporkan) != 2 {
		t.Fatalf("galat tak boleh di-cache & harus dilaporkan: baca=%d lapor=%d", p.n, len(dilaporkan))
	}

	// Pulih: begitu tabel terbaca, izin langsung berlaku.
	p.err, p.codes = nil, map[string]bool{"x": true}
	if codes, _ := loader(context.Background(), "USER"); !codes["x"] {
		t.Fatal("setelah tabel pulih izin harus termuat")
	}
}

func TestHasPermissionMenghormatiWildcard(t *testing.T) {
	ctx := contextx.WithPermissionCodes(context.Background(), map[string]bool{contextx.PermissionAll: true})
	if !contextx.HasPermission(ctx, "apa.saja") {
		t.Error("wildcard harus lolos izin apa pun")
	}
	ctx = contextx.WithPermissionCodes(context.Background(), map[string]bool{"a": true})
	if !contextx.HasPermission(ctx, "a") || contextx.HasPermission(ctx, "b") {
		t.Error("izin biasa: hanya kode yang dipegang")
	}
	if contextx.HasPermission(context.Background(), "a") {
		t.Error("tanpa izin di context = tidak punya")
	}
}
