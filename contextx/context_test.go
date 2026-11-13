package contextx

import (
	"context"
	"testing"
)

func TestPrivileged(t *testing.T) {
	ctx := WithSubjectUUID(WithRoleCode(context.Background(), "USER"), "u-1")

	privileged := Privileged(ctx)

	if got := GetRoleCode(privileged); got != "" {
		t.Errorf("role harus kosong, dapat %q", got)
	}

	if got := GetSubjectUUID(privileged); got != "u-1" {
		t.Errorf("identitas subjek harus tetap terbawa, dapat %q", got)
	}

	if got := GetRoleCode(ctx); got != "USER" {
		t.Errorf("ctx asal tidak boleh berubah, dapat %q", got)
	}
}

func TestCoversModuleDanIsRestricted(t *testing.T) {
	dengan := func(role string, codes map[string]bool) context.Context {
		return WithPermissionCodes(WithRoleCode(context.Background(), role), codes)
	}

	kasus := []struct {
		nama     string
		ctx      context.Context
		terbatas bool
	}{
		{"USER tanpa izin", dengan("USER", map[string]bool{}), true},
		{"USER ber-izin pakai modul (tetap terbatas)", dengan("USER", map[string]bool{"article.catalog.view": true}), true},
		{"role asing ber-izin modul lain", dengan("MENTOR", map[string]bool{"ngaji.meeting.manage": true}), true},
		{"awalan mirip bukan modul yang sama", dengan("X", map[string]bool{"articlex.a": true}), true},
		{"izin bernilai false", dengan("X", map[string]bool{"article.article.manage": false}), true},
		{"pengelola modul", dengan("ADMINISTRATOR", map[string]bool{"article.master.manage": true}), false},
		{"wildcard", dengan("SUPERADMIN", map[string]bool{PermissionAll: true}), false},
		{"konteks internal (role kosong)", context.Background(), false},
		{"Privileged", Privileged(dengan("USER", map[string]bool{})), false},
	}
	for _, k := range kasus {
		if got := IsRestricted(k.ctx, "article"); got != k.terbatas {
			t.Errorf("%s: IsRestricted = %v, mau %v", k.nama, got, k.terbatas)
		}
	}

	if !HasModulePermission(dengan("A", map[string]bool{"article.x": true}), "article") {
		t.Error("HasModulePermission harus true bila memegang izin modul")
	}
}
