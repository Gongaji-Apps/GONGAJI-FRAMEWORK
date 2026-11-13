package contextx

import (
	"context"
	"strings"
)

type key string

const (
	AuthTypeKey        key = "auth_type"
	CorrelationIDKey   key = "correlation_id"
	OverviewKey        key = "overview"
	PermissionCodesKey key = "permission_codes"
	RequestIDKey       key = "request_id"
	RoleCodeKey        key = "role_code"
	SubjectEmailKey    key = "subject_email"
	SubjectFullNameKey key = "subject_full_name"
	SubjectUUIDKey     key = "subject_uuid"
	SubjectUserIDKey   key = "subject_user_id"
	SubjectTesterKey   key = "subject_tester"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

func WithSubjectUUID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SubjectUUIDKey, id)
}

func GetSubjectUUID(ctx context.Context) string {
	return getString(ctx, SubjectUUIDKey)
}

func GetSubjectUUIDNil(ctx context.Context) *string {
	return getStringNil(ctx, SubjectUUIDKey)
}

// SubjectUserID adalah kunci alternatif BIGINT (public.user_mstr.user_id),
// dipakai untuk FK audit (created_by/updated_by) pada service yang memakai
// konvensi FK BIGINT alih-alih UUID.
func WithSubjectUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, SubjectUserIDKey, id)
}

func GetSubjectUserID(ctx context.Context) *int64 {
	if v, ok := ctx.Value(SubjectUserIDKey).(int64); ok {
		return &v
	}

	return nil
}

func WithSubjectTester(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, SubjectTesterKey, value)
}

func GetSubjectTester(ctx context.Context) bool {
	return getBool(ctx, SubjectTesterKey)
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, id)
}

func WithSubjectFullName(ctx context.Context, fullName string) context.Context {
	return context.WithValue(ctx, SubjectFullNameKey, fullName)
}

func WithSubjectEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, SubjectEmailKey, email)
}

func GetSubjectFullName(ctx context.Context) string {
	return getString(ctx, SubjectFullNameKey)
}

func GetSubjectEmail(ctx context.Context) string {
	return getString(ctx, SubjectEmailKey)
}

func WithRoleCode(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, RoleCodeKey, code)
}

func WithPermissionCodes(ctx context.Context, codes map[string]bool) context.Context {
	return context.WithValue(ctx, PermissionCodesKey, codes)
}

func WithAuthType(ctx context.Context, authType string) context.Context {
	return context.WithValue(ctx, AuthTypeKey, authType)
}

func GetRequestID(ctx context.Context) string {
	return getString(ctx, RequestIDKey)
}

func GetCorrelationID(ctx context.Context) string {
	return getString(ctx, CorrelationIDKey)
}

func GetRoleCode(ctx context.Context) string {
	return getString(ctx, RoleCodeKey)
}

func GetPermissionCodes(ctx context.Context) map[string]bool {
	if v, ok := ctx.Value(PermissionCodesKey).(map[string]bool); ok {
		return v
	}
	return nil
}

// PermissionAll adalah kode izin wildcard: pemegangnya lolos SEMUA pemeriksaan izin.
// Diisi pemuat izin untuk role super (mis. SUPERADMIN) supaya role itu tak bisa
// terkunci dari sistem akibat salah isi tabel izin.
const PermissionAll = "*"

// HasPermission melaporkan apakah subjek memegang izin code (atau wildcard
// PermissionAll). Pakai ini di handler/service alih-alih membaca map
// GetPermissionCodes langsung, supaya wildcard dihormati di semua tempat.
func HasPermission(ctx context.Context, code string) bool {
	codes := GetPermissionCodes(ctx)
	return codes != nil && (codes[code] || codes[PermissionAll])
}

// CoversModule melaporkan apakah kumpulan izin memuat wildcard PermissionAll atau
// setidaknya satu izin modul itu ("article" → "article.*").
func CoversModule(codes map[string]bool, module string) bool {
	if codes[PermissionAll] {
		return true
	}
	prefix := module + "."
	for code, held := range codes {
		if held && strings.HasPrefix(code, prefix) {
			return true
		}
	}
	return false
}

// HasModulePermission: subjek memegang setidaknya satu izin modul (atau wildcard).
func HasModulePermission(ctx context.Context, module string) bool {
	return CoversModule(GetPermissionCodes(ctx), module)
}

// IsRestricted melaporkan apakah subjek harus diperlakukan TERBATAS di modul ini:
// hanya data miliknya sendiri, hanya konten yang terbit/aktif, kolom internal
// disembunyikan. Pengganti pola lama `GetRoleCode(ctx) == "USER"`.
//
// Pola lama adalah daftar-tolak satu role: role lain yang bukan pengelola (mentor,
// admin produk lain, dst.) lolos tanpa batasan. Di sini kebalikannya — terbatas
// adalah BAWAAN, dan longgar hanya bagi pemegang izin modul:
//
//	if contextx.IsRestricted(ctx, "article") { payload.Article_Status = "PUBLISHED" }
//
// Role kosong TIDAK terbatas: itu konteks internal (Privileged, webhook, cron,
// rute tanpa login) yang perilakunya harus tetap seperti sebelumnya.
func IsRestricted(ctx context.Context, module string) bool {
	return GetRoleCode(ctx) != "" && !HasModulePermission(ctx, module)
}

func GetAuthType(ctx context.Context) string {
	return getString(ctx, AuthTypeKey)
}

func getString(ctx context.Context, k key) string {
	if v, ok := ctx.Value(k).(string); ok {
		return v
	}

	return ""
}

func getStringNil(ctx context.Context, k key) *string {
	if v, ok := ctx.Value(k).(string); ok {
		return &v
	}

	return nil
}

func getBool(ctx context.Context, k key) bool {
	if v, ok := ctx.Value(k).(bool); ok {
		return v
	}

	return false
}

func WithOverview(ctx context.Context, value bool) context.Context {
	return context.WithValue(ctx, OverviewKey, value)
}

func GetOverview(ctx context.Context) bool {
	return getBool(ctx, OverviewKey)
}

// Privileged mengembalikan ctx dengan role dikosongkan: hook AfterFind yang
// menyamarkan kolom sensitif untuk role USER tidak lagi berlaku, sehingga service
// bisa membaca nilai asli untuk validasi internal. Pengganti context.Background()
// di kode lama: deadline/cancel request dan identitas subjek tetap terbawa.
// Jangan meneruskan ctx hasil Privileged ke jalur yang membentuk respons klien.
func Privileged(ctx context.Context) context.Context {
	return WithRoleCode(ctx, "")
}
