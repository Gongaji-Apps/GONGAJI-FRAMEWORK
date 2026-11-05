// Package identity menyediakan potret baris public.user_mstr yang dibutuhkan service
// pembaca (bukan pemilik tabel), supaya struktur user_mstr tidak lagi disalin di tiap
// service.
//
// Tabel user_mstr dimiliki service penulis (public, authentication, subscription) yang
// tetap memakai model lengkapnya sendiri. Service lain cukup:
//
//   - UserAccount + Lookup: kolom minimal untuk validator JWT standar
//     (jwt.NewAccountStrategy). Hanya kolom ini yang dibaca dari DB — kolom rahasia
//     (password, OTP, FCM, dst.) tidak pernah dimuat, dan perubahan kolom lain di
//     user_mstr tidak menyentuh service pembaca.
//
// Contoh (middleware service):
//
//	strategy, err := jwt.NewAccountStrategy(jwt.AccountStrategyConfig{
//	    Secret:     []byte(os.Getenv("SECRET_KEY")),
//	    DefaultTTL: time.Duration(env.Int("EXPIRED_TOKEN", 24)) * time.Hour,
//	    Lookup:     identity.Lookup(db),
//	})
package identity

import (
	"context"
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/jwt"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
)

// TableName adalah tabel user bersama semua service.
const TableName = "public.user_mstr"

// UserAccount adalah kolom user_mstr yang dibutuhkan pemeriksaan token: identitas,
// role, token single-device, status aktif, dan penanda akun penguji.
//
// Tidak untuk respons API: Token disembunyikan dari JSON, dan struct ini sengaja tidak
// punya kolom profil (email/telp/alamat ditampilkan lewat model service sendiri).
type UserAccount struct {
	UserID   *int64  `gorm:"column:user_id" json:"-"`
	UUID     string  `gorm:"column:user_uuid" json:"-"`
	FullName string  `gorm:"column:user_full_name" json:"-"`
	Email    *string `gorm:"column:user_email" json:"-"`
	Role     *string `gorm:"column:user_role" json:"-"`
	Token    *string `gorm:"column:user_token" json:"-"`
	Active   *bool   `gorm:"column:user_active" json:"-"`
	Tester   *bool   `gorm:"column:user_tester" json:"-"`
}

// TableName memetakan UserAccount ke public.user_mstr untuk GORM.
func (UserAccount) TableName() string { return TableName }

// accountColumns: daftar SELECT eksplisit — kolom di luar daftar ini tidak pernah dibaca.
var accountColumns = []string{
	"user_id", "user_uuid", "user_full_name", "user_email",
	"user_role", "user_token", "user_active", "user_tester",
}

// Account memetakan baris ke jwt.Account (pointer kosong → nilai nol; Tester nil → false).
func (u UserAccount) Account() *jwt.Account {
	return &jwt.Account{
		UUID:     u.UUID,
		UserID:   u.UserID,
		FullName: u.FullName,
		Email:    deref(u.Email),
		Role:     deref(u.Role),
		Token:    u.Token,
		Active:   u.Active,
		Tester:   u.Tester != nil && *u.Tester,
	}
}

// FindByUUID membangun query satu akun berdasarkan user_uuid dengan kolom UserAccount saja.
func FindByUUID(db *gorm.DB, subjectUUID string) *gorm.DB {
	return db.Model(&UserAccount{}).Select(accountColumns).Where("user_uuid = ?", subjectUUID).Limit(1)
}

// Lookup mengembalikan jwt.AccountLookup yang membaca UserAccount dari db.
// Akun tidak ada → (nil, nil) → 401 "Akun Anda tidak ditemukan"; galat DB lain
// dinormalisasi (errors.NormalizeDBError) dan diteruskan (→ 500).
func Lookup(db *gorm.DB) jwt.AccountLookup {
	return func(ctx context.Context, subjectUUID string) (*jwt.Account, error) {
		var users []UserAccount
		if err := FindByUUID(db.WithContext(ctx), subjectUUID).Find(&users).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, errors.NormalizeDBError(err, "User")
		}
		if len(users) == 0 {
			return nil, nil
		}
		return users[0].Account(), nil
	}
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
