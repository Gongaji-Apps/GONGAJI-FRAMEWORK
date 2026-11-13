package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Klaim tambahan token ber-sesi (multi-role, opsi 3). Token lama tidak memuatnya dan
// tetap diperiksa lewat jalur lama (role dari akun, single-device lewat Account.Token).
const (
	// ClaimSessionID: id baris sesi (auth.user_session.session_id).
	ClaimSessionID = "sid"
	// ClaimRole: peran aktif sesi. HANYA informasi untuk klien — yang berlaku adalah
	// role pada baris sesi, bukan klaim ini.
	ClaimRole = "role"
	// ClaimContext: konteks login (ContextApp / ContextCMS). Informasi untuk klien.
	ClaimContext = "ctx"
)

// Konteks sesi: satu akun boleh punya sesi aplikasi dan sesi CMS sekaligus, masing-
// masing dengan peran aktifnya sendiri.
const (
	ContextApp = "APP"
	ContextCMS = "CMS"
)

// Session adalah potret satu sesi login: milik siapa, peran aktifnya, dan apakah masih
// berlaku. Dimuat SessionLookup dari auth.user_session (+ keanggotaan auth.user_role).
type Session struct {
	UserUUID  string
	Role      string    // peran aktif sesi; menggantikan Account.Role untuk request ini
	Context   string    // ContextApp / ContextCMS
	TokenHash string    // HashToken(token) saat sesi dibuat
	ExpiresAt time.Time // nol = ikut kedaluwarsa token
	Revoked   bool      // dicabut (logout, login baru di konteks yang sama, role berubah)
	// Superseded: dicabut karena akun masuk lagi di konteks yang sama (pesan
	// "masuk di perangkat lain", sama seperti single-device lama).
	Superseded bool
	// MemberActive: akun masih memegang Role (baris auth.user_role aktif). False →
	// sesi ditolak walau belum dicabut: mencabut role berlaku seketika.
	MemberActive bool
}

// SessionLookup memuat sesi berdasarkan id dari klaim ClaimSessionID. Kembalikan
// (nil, nil) bila sesi tidak ada. Galat diteruskan apa adanya (DB mati → 500): sesi
// yang tak bisa diperiksa TIDAK dianggap sah.
type SessionLookup func(ctx context.Context, sessionID string) (*Session, error)

// HashToken mengembalikan SHA-256 (hex) dari token. Yang disimpan di tabel sesi adalah
// hash ini, bukan tokennya: isi tabel yang bocor tidak bisa dipakai untuk masuk.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
