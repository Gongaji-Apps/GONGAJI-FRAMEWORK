package errors

import (
	stderrors "errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ========================================================
// ==================== NORMALIZER =========================
// ========================================================

// NormalizeDBError memetakan galat DB menjadi *AppError berpesan manusiawi.
// Urutan: pesan per-constraint terdaftar → pesan bawaan per jenis galat
// (memakai label tabel/kolom terdaftar bila ada). Nama tabel, kolom, dan
// constraint mentah tak pernah muncul di pesan.
func NormalizeDBError(err error, tableName string) error {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}

	var pgErr *pgconn.PgError
	hasPg := stderrors.As(err, &pgErr)
	table := tableName
	if hasPg && pgErr.TableName != "" {
		if pgErr.SchemaName != "" {
			table = pgErr.SchemaName + "." + pgErr.TableName
		} else {
			table = pgErr.TableName
		}
	}
	if hasPg {
		if msg := constraintMessage(pgErr.ConstraintName); msg != "" {
			switch pgErr.Code {
			case "23505", "23503":
				return NewConflict(msg)
			default:
				return NewBadRequest(msg)
			}
		}
	}

	label := labelAtauData(table)

	switch {
	case IsDuplicateError(err):
		return NewConflict(label + " yang sama sudah ada. Periksa kembali isian Anda.")
	case IsForeignKeyError(err):
		if hasPg && strings.Contains(pgErr.Message, "still referenced") {
			return NewConflict(label + " tidak bisa dihapus karena masih dipakai oleh data lain.")
		}
		return NewConflict("Data yang dipilih tidak ditemukan atau sudah dihapus. Muat ulang halaman lalu coba lagi.")
	case IsNotNullError(err):
		if hasPg && pgErr.ColumnName != "" {
			return NewBadRequest(FieldLabel(pgErr.ColumnName) + " wajib diisi.")
		}
		return NewBadRequest("Ada isian wajib yang masih kosong.")
	case IsCheckConstraintError(err):
		return NewBadRequest("Ada isian yang nilainya tidak sesuai aturan. Periksa kembali isian Anda.")
	default:
		return NewInternalServerError("Terjadi kesalahan saat menyimpan " + strings.ToLower(label) + ". Coba lagi beberapa saat lagi.")
	}
}

// IsDBConstraintError: true bila err adalah galat constraint Postgres yang
// bisa dipetakan ke pesan manusiawi (dipakai response.Error untuk galat DB
// mentah yang lolos dari service).
func IsDBConstraintError(err error) bool {
	return IsDuplicateError(err) || IsForeignKeyError(err) || IsNotNullError(err) || IsCheckConstraintError(err)
}
