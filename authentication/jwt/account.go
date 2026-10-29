package jwt

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

// Account adalah potret akun yang dibutuhkan pemeriksaan standar setelah tanda tangan
// JWT sah. Service memetakan baris user_mstr-nya sendiri ke struct ini di dalam
// AccountLookup; tipe kolom yang berbeda antar service (email/role pointer atau bukan)
// diselesaikan di sana, bukan di framework.
type Account struct {
	UUID     string
	UserID   *int64 // opsional: kunci alternatif BIGINT (public.user_mstr.user_id)
	FullName string
	Email    string
	Role     string
	Token    *string         // token single-device yang tersimpan di DB; nil = belum login
	Active   *bool           // nil = dianggap aktif
	Tester   bool            // akun penguji (lolos gating tertentu di service)
	Links    map[string]bool // tautan wajib per role, mis. "Partner": user_partner != nil
	Extra    map[string]any  // diteruskan apa adanya ke AuthClaims.Extra
}

// AccountLookup memuat akun berdasarkan subject UUID dari token. Kembalikan (nil, nil)
// bila akun tidak ada; galat *errors.AppError NotFound juga diperlakukan sebagai tidak
// ada. Galat lain diteruskan apa adanya (mis. DB mati → 500).
type AccountLookup func(ctx context.Context, subjectUUID string) (*Account, error)

// PermissionLoader memuat kode izin untuk sebuah role (RBAC). Opsional.
type PermissionLoader func(ctx context.Context, role string) (map[string]bool, error)

// DefaultSingleDeviceRoles: role yang tokennya harus sama dengan yang tersimpan di DB
// (satu perangkat aktif). Admin/CMS boleh masuk dari banyak perangkat.
var DefaultSingleDeviceRoles = []string{"USER"}

// AccountStrategyConfig menyusun Strategy dengan Validator standar GoNgaji.
type AccountStrategyConfig struct {
	// Secret dan DefaultTTL/Issuer diteruskan ke Manager (lihat Config).
	Secret     []byte
	DefaultTTL time.Duration // 0 = DefaultAccountTTL
	Issuer     string

	// Lookup wajib.
	Lookup AccountLookup

	// SingleDeviceRoles: nil = DefaultSingleDeviceRoles; slice kosong (bukan nil)
	// mematikan pemeriksaan single-device.
	SingleDeviceRoles []string

	// BoundRoles memetakan role → nama tautan di Account.Links yang wajib true,
	// mis. {"ADMINISTRATOR_LEARNING_BRANCH": "Partner", "ADMINISTRATOR_DONATION": "Foundation"}.
	// Nama tautan dipakai di pesan galat ("...belum tertaut ke Partner manapun").
	BoundRoles map[string]string

	// Permissions (opsional) mengisi AuthClaims.PermissionCodes untuk middleware.RequirePermission.
	Permissions PermissionLoader

	// StrategyName opsional (lihat Strategy.StrategyName).
	StrategyName string
}

// DefaultAccountTTL dipakai Generate bila DefaultTTL tidak diisi.
const DefaultAccountTTL = 24 * time.Hour

// Pesan 401 standar — satu kata-kata untuk semua service.
const (
	MsgSessionEnded   = "Sesi Anda berakhir. Silakan masuk kembali."
	MsgAccountMissing = "Akun Anda tidak ditemukan. Silakan masuk kembali."
	MsgOtherDevice    = "Sesi Anda berakhir karena akun ini masuk di perangkat lain. Silakan masuk kembali."
	MsgInactive       = "Akun Anda telah dinonaktifkan. Hubungi pengelola."
	msgUnboundFormat  = "Akun Anda belum tertaut ke %s manapun. Hubungi pengelola."
)

// NewAccountStrategy membangun Strategy yang, setelah Manager memverifikasi tanda
// tangan dan kedaluwarsa, menjalankan pemeriksaan standar:
//
//  1. subject UUID harus ada di token dan akun harus ditemukan (401, bukan 404);
//  2. single-device untuk SingleDeviceRoles: token bearer harus sama dengan Account.Token;
//  3. Account.Active false → 401;
//  4. role di BoundRoles wajib punya tautan terkait (Account.Links[nama] == true);
//  5. Permissions (bila ada) mengisi PermissionCodes.
//
// Hasilnya AuthClaims lengkap (UUID, UserID, nama, email, role, tester, izin, extra).
//
// Contoh:
//
//	strategy, err := jwt.NewAccountStrategy(jwt.AccountStrategyConfig{
//	    Secret:     []byte(os.Getenv("SECRET_KEY")),
//	    DefaultTTL: time.Duration(env.Int("EXPIRED_TOKEN", 24)) * time.Hour,
//	    BoundRoles: map[string]string{"ADMINISTRATOR_LEARNING_BRANCH": "Partner"},
//	    Lookup: func(ctx context.Context, uuid string) (*jwt.Account, error) {
//	        user, err := repo.FindByUUID(ctx, uuid)
//	        if err != nil || user == nil {
//	            return nil, err
//	        }
//	        return &jwt.Account{UUID: user.Uuid, Role: user.Role, Token: user.Token, Active: user.Active,
//	            Links: map[string]bool{"Partner": user.Partner != nil}}, nil
//	    },
//	})
func NewAccountStrategy(cfg AccountStrategyConfig) (*Strategy, error) {
	if cfg.Lookup == nil {
		return nil, stderrors.New("jwt: AccountStrategyConfig.Lookup is required")
	}

	ttl := cfg.DefaultTTL
	if ttl <= 0 {
		ttl = DefaultAccountTTL
	}

	manager, err := New(Config{Secret: cfg.Secret, Issuer: cfg.Issuer, DefaultTTL: ttl})
	if err != nil {
		return nil, err
	}

	singleDevice := make(map[string]bool)
	roles := cfg.SingleDeviceRoles
	if roles == nil {
		roles = DefaultSingleDeviceRoles
	}
	for _, role := range roles {
		singleDevice[role] = true
	}

	validator := func(ctx context.Context, claims *Claims, raw string) (*middleware.AuthClaims, error) {
		if claims.SubjectUUID == "" {
			return nil, errors.NewUnauthorized(MsgSessionEnded)
		}

		account, err := cfg.Lookup(ctx, claims.SubjectUUID)
		if err != nil {
			var appErr *errors.AppError
			if stderrors.As(err, &appErr) && appErr.Code == errors.NotFound {
				return nil, errors.NewUnauthorized(MsgAccountMissing)
			}
			return nil, err
		}
		if account == nil {
			return nil, errors.NewUnauthorized(MsgAccountMissing)
		}

		if singleDevice[account.Role] {
			if account.Token == nil || *account.Token != raw {
				return nil, errors.NewUnauthorized(MsgOtherDevice)
			}
		}

		if account.Active != nil && !*account.Active {
			return nil, errors.NewUnauthorized(MsgInactive)
		}

		if link, bound := cfg.BoundRoles[account.Role]; bound && !account.Links[link] {
			return nil, errors.NewUnauthorized(fmt.Sprintf(msgUnboundFormat, link))
		}

		permissions := map[string]bool{}
		if cfg.Permissions != nil && account.Role != "" {
			permissions, err = cfg.Permissions(ctx, account.Role)
			if err != nil {
				return nil, err
			}
			if permissions == nil {
				permissions = map[string]bool{}
			}
		}

		extra := account.Extra
		if extra == nil {
			extra = map[string]any{}
		}

		return &middleware.AuthClaims{
			SubjectUUID:     account.UUID,
			SubjectUserID:   account.UserID,
			SubjectFullName: account.FullName,
			SubjectEmail:    account.Email,
			SubjectTester:   account.Tester,
			Role:            account.Role,
			PermissionCodes: permissions,
			Extra:           extra,
		}, nil
	}

	return &Strategy{Manager: manager, Validator: validator, StrategyName: cfg.StrategyName}, nil
}
