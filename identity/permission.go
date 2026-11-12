package identity

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/jwt"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
)

// PermissionTable adalah tabel pasangan role–izin bersama semua service (RBAC).
const PermissionTable = "auth.role_permission"

// DefaultPermissionTTL: perubahan izin sebuah role berlaku paling lambat selama ini.
// Pemeriksaan akun (token single-device, akun nonaktif) TIDAK di-cache.
const DefaultPermissionTTL = 60 * time.Second

// DefaultSuperRoles: role yang selalu memegang semua izin (wildcard), tanpa membaca
// tabel — sehingga tak bisa terkunci akibat salah isi tabel izin.
var DefaultSuperRoles = []string{"SUPERADMIN"}

// PermissionConfig mengatur Permissions. Nilai nol = bawaan.
type PermissionConfig struct {
	// TTL cache per role. 0 = DefaultPermissionTTL.
	TTL time.Duration
	// SuperRoles: nil = DefaultSuperRoles; slice kosong (bukan nil) = tak ada role super.
	SuperRoles []string
	// OnError dipanggil bila tabel izin gagal dibaca (mis. untuk log). Opsional.
	OnError func(role string, err error)
}

// Permissions mengembalikan jwt.PermissionLoader yang membaca izin role dari
// auth.role_permission, di-cache per role, untuk dipasang di
// jwt.AccountStrategyConfig.Permissions:
//
//	strategy, err := jwt.NewAccountStrategy(jwt.AccountStrategyConfig{
//	    Secret:      []byte(os.Getenv("SECRET_KEY")),
//	    Lookup:      identity.Lookup(db),
//	    Permissions: identity.Permissions(db, identity.PermissionConfig{}),
//	})
//
// Route lalu dijaga middleware.RequirePermission("article.article.manage"): siapa
// yang memegang izin itu adalah DATA, bukan kode.
//
// Gagal baca tabel TIDAK menggagalkan request: role itu mendapat izin kosong
// (RequirePermission → 403, route lain tetap jalan), galat tidak di-cache, dan
// request berikutnya mencoba lagi. Loader dipanggil untuk SETIAP request
// terautentikasi (termasuk role USER), jadi galat di sini tak boleh mematikan aplikasi.
func Permissions(db *gorm.DB, cfg PermissionConfig) jwt.PermissionLoader {
	load := func(ctx context.Context, role string) (map[string]bool, error) {
		var rows []permissionRow
		if err := permissionQuery(db.WithContext(ctx), role).Find(&rows).Error; err != nil {
			return nil, err
		}
		set := make(map[string]bool, len(rows))
		for _, row := range rows {
			set[row.Code] = true
		}
		return set, nil
	}
	return newPermissionLoader(cfg, load, time.Now)
}

// permissionRow: satu baris auth.role_permission (hanya kolom yang dibaca).
type permissionRow struct {
	Code string `gorm:"column:permission_code"`
}

// permissionQuery: kode izin satu role.
func permissionQuery(db *gorm.DB, role string) *gorm.DB {
	return db.Table(PermissionTable).Select("permission_code").Where("role_code = ?", role)
}

type permissionEntry struct {
	codes   map[string]bool
	expires time.Time
}

// newPermissionLoader: inti ber-cache; load & now disuntikkan untuk uji.
//
// Map yang dikembalikan dipakai bersama lintas request — aman karena seluruh pemakai
// (RequirePermission, contextx.HasPermission) hanya membaca.
func newPermissionLoader(
	cfg PermissionConfig,
	load func(ctx context.Context, role string) (map[string]bool, error),
	now func() time.Time,
) jwt.PermissionLoader {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultPermissionTTL
	}

	superRoles := cfg.SuperRoles
	if superRoles == nil {
		superRoles = DefaultSuperRoles
	}
	super := make(map[string]bool, len(superRoles))
	for _, role := range superRoles {
		super[role] = true
	}
	all := map[string]bool{contextx.PermissionAll: true}

	var mu sync.Mutex
	entries := map[string]permissionEntry{}

	return func(ctx context.Context, role string) (map[string]bool, error) {
		if super[role] {
			return all, nil
		}

		mu.Lock()
		entry, ok := entries[role]
		mu.Unlock()
		if ok && now().Before(entry.expires) {
			return entry.codes, nil
		}

		codes, err := load(ctx, role)
		if err != nil {
			if cfg.OnError != nil {
				cfg.OnError(role, err)
			}
			return map[string]bool{}, nil
		}

		mu.Lock()
		entries[role] = permissionEntry{codes: codes, expires: now().Add(ttl)}
		mu.Unlock()

		return codes, nil
	}
}
