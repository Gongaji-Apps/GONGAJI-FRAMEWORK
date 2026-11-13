package identity

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/jwt"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

// Tabel multi-role (opsi 3): keanggotaan role per akun dan sesi login per konteks.
const (
	UserRoleTable    = "auth.user_role"
	UserSessionTable = "auth.user_session"
)

// RevokedByNewLogin: alasan cabut sesi saat akun masuk lagi di konteks yang sama.
// Service penulis sesi (authentication) memakai nilai ini; pemeriksa menerjemahkannya
// menjadi pesan "masuk di perangkat lain".
const RevokedByNewLogin = "login_baru"

// sessionRow: kolom yang dibaca pemeriksa sesi — sesi + status keanggotaan role-nya.
type sessionRow struct {
	UserUUID      string     `gorm:"column:user_uuid"`
	Role          string     `gorm:"column:role_code"`
	Context       string     `gorm:"column:session_context"`
	TokenHash     string     `gorm:"column:session_token_hash"`
	ExpiredDate   time.Time  `gorm:"column:session_expired_date"`
	RevokedDate   *time.Time `gorm:"column:session_revoked_date"`
	RevokedReason *string    `gorm:"column:session_revoked_reason"`
	MemberActive  *bool      `gorm:"column:member_active"`
}

// sessionQuery: satu sesi berdasarkan id, digabung dengan keanggotaan role-nya
// (LEFT JOIN: keanggotaan yang sudah dihapus → member_active NULL → ditolak).
func sessionQuery(db *gorm.DB, sessionID string) *gorm.DB {
	return db.Table(UserSessionTable+" AS s").
		Select("s.user_uuid, s.role_code, s.session_context, s.session_token_hash, s.session_expired_date, s.session_revoked_date, s.session_revoked_reason, r.urole_active AS member_active").
		Joins("LEFT JOIN "+UserRoleTable+" AS r ON r.user_uuid = s.user_uuid AND r.role_code = s.role_code").
		Where("s.session_id = ?", sessionID).
		Limit(1)
}

func (row sessionRow) session() *jwt.Session {
	return &jwt.Session{
		UserUUID:     row.UserUUID,
		Role:         row.Role,
		Context:      row.Context,
		TokenHash:    row.TokenHash,
		ExpiresAt:    row.ExpiredDate,
		Revoked:      row.RevokedDate != nil,
		Superseded:   row.RevokedDate != nil && row.RevokedReason != nil && *row.RevokedReason == RevokedByNewLogin,
		MemberActive: row.MemberActive != nil && *row.MemberActive,
	}
}

// Sessions mengembalikan jwt.SessionLookup yang membaca auth.user_session untuk
// dipasang di jwt.AccountStrategyConfig.Sessions:
//
//	strategy, err := jwt.NewAccountStrategy(jwt.AccountStrategyConfig{
//	    Secret:   []byte(os.Getenv("SECRET_KEY")),
//	    Lookup:   identity.Lookup(db),
//	    Sessions: identity.Sessions(db),
//	})
//
// TIDAK di-cache: mencabut sesi (logout, login baru, role dicabut) harus berlaku pada
// request berikutnya. Biayanya satu query tambahan per request ber-token sesi. Sesi
// tidak ada → (nil, nil) → 401; galat DB dinormalisasi dan diteruskan (→ 500).
func Sessions(db *gorm.DB) jwt.SessionLookup {
	return func(ctx context.Context, sessionID string) (*jwt.Session, error) {
		var rows []sessionRow
		if err := sessionQuery(db.WithContext(ctx), sessionID).Find(&rows).Error; err != nil {
			return nil, errors.NormalizeDBError(err, "Sesi")
		}
		if len(rows) == 0 {
			return nil, nil
		}
		return rows[0].session(), nil
	}
}
