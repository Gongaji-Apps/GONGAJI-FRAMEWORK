package errors

import "fmt"

// ========================================================
// ==================== NORMALIZER =========================
// ========================================================

func NormalizeDBError(err error, table_name string) error {
	if err == nil {
		return nil
	}

	switch {
	case IsDuplicateError(err):
		return NewConflict("[Duplicate]")
	case IsForeignKeyError(err):
		return NewConflict("[Foreign Key]")
	case IsNotNullError(err):
		return NewBadRequest("[Not Null]")
	case IsCheckConstraintError(err):
		return NewBadRequest("[Check Constraint]")
	default:
		return NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat menyimpan Data %s.", table_name))
	}
}
