package errors

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ========================================================
// ==================== DUPLICATE ==========================
// ========================================================

// IsDuplicateError checks if error is caused by unique constraint violation
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	// GORM generic duplicate error (future-proof)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	// PostgreSQL (pgx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	// ========================================================
	// ==================== MYSQL (OPTIONAL) ===================
	// ========================================================
	// Uncomment if using MySQL driver
	//
	// var mysqlErr *mysql.MySQLError
	// if errors.As(err, &mysqlErr) {
	//     return mysqlErr.Number == 1062
	// }

	return false
}

// ========================================================
// ==================== FOREIGN KEY ========================
// ========================================================

// IsForeignKeyError checks if error is caused by FK constraint
func IsForeignKeyError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}

	return false
}

// ========================================================
// ==================== NOT NULL ===========================
// ========================================================

// IsNotNullError checks if error is caused by NOT NULL constraint
func IsNotNullError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23502"
	}

	return false
}

// ========================================================
// ==================== CHECK CONSTRAINT ===================
// ========================================================

// IsCheckConstraintError checks if error is caused by CHECK constraint
func IsCheckConstraintError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23514"
	}

	return false
}
