package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	frameworkError "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"
)

type BaseRepository[T any] struct {
	DB        *gorm.DB
	TableName string
	Debug     bool
}

func NewBaseRepository[T any](db *gorm.DB, table string) BaseRepository[T] {
	return BaseRepository[T]{
		DB:        db,
		TableName: table,
		Debug:     strings.EqualFold(os.Getenv("APP_DEBUG"), "true"),
	}
}

// ========================================================
// ==================== BASE GET QUERY ====================
// ========================================================
func (r *BaseRepository[T]) db(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.DB
	}

	q := tx.WithContext(ctx)

	if r.Debug {
		q = q.Debug()
	}

	return q
}

// ===========================================================
// ==================== BASE APPLY WHERES ====================
// ===========================================================
func (r *BaseRepository[T]) applyWheres(qb *gorm.DB, q query.Query) *gorm.DB {
	if q.Where != nil {
		qb = qb.Where(q.Where)
	}

	for _, w := range q.Wheres {
		if len(w.Args) > 0 {
			qb = qb.Where(w.Query, w.Args...)
		} else {
			qb = qb.Where(w.Query)
		}
	}

	return qb
}

// ============================================================
// ==================== BASE APPLY HAVINGS ====================
// ============================================================
func (r *BaseRepository[T]) applyHavings(qb *gorm.DB, q query.Query) *gorm.DB {
	for _, h := range q.Havings {
		qb = qb.Having(h)
	}

	return qb
}

// ==============================================================
// ==================== NORMALIZE PAGINATION ====================
// ==============================================================
func normalizePagination(p query.Pagination) query.Pagination {
	if p.Limit <= 0 {
		p.Limit = 100
	}

	if p.Limit > 1000 {
		p.Limit = 1000
	}

	if p.Pagination < 1 {
		p.Pagination = 1
	}

	return p
}

// ===============================================================
// ==================== BASE APPLY PAGINATION ====================
// ===============================================================
func (r *BaseRepository[T]) applyPagination(qb *gorm.DB, p query.Pagination) *gorm.DB {
	return qb.Limit(p.Limit).Offset((p.Pagination - 1) * p.Limit)
}

// ==========================================================
// ==================== BASE BUILD QUERY ====================
// ==========================================================
func (r *BaseRepository[T]) BaseBuildQuery(
	ctx context.Context,
	q query.Query,
	custom func(*gorm.DB) *gorm.DB,
) *gorm.DB {
	return r.BaseBuildQueryFrom(ctx, nil, q, custom)
}

// =================================================================
// ==================== BASE BUILD QUERY FROM ======================
// =================================================================
// BaseBuildQueryFrom is identical to BaseBuildQuery but starts
// from the provided base *gorm.DB instead of the stored connection.
// Pass nil to fall back to the default behavior.
// Use this in GORM preload callbacks to preserve the foreign-key
// WHERE constraint injected by GORM via the db1 parameter.
func (r *BaseRepository[T]) BaseBuildQueryFrom(
	ctx context.Context,
	base *gorm.DB,
	q query.Query,
	custom func(*gorm.DB) *gorm.DB,
) *gorm.DB {
	qb := r.db(ctx, base).Model(new(T))

	qb = r.applyHavings(qb, q)

	qb = r.applyWheres(qb, q)

	if q.Order != "" {
		qb = qb.Order(q.Order)
	}

	if custom != nil {
		qb = custom(qb)
	}

	return qb
}

// ========================================================
// ==================== BASE GET ARRAY ====================
// ========================================================
func (r *BaseRepository[T]) BaseGetArray(
	ctx context.Context,
	qb *gorm.DB,
	p query.Pagination,
) (*result.ArrayResult[T], error) {
	var data []T
	var total int64

	if err := qb.Count(&total).Error; err != nil {
		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.TableName))
	}

	p = normalizePagination(p)

	qb = r.applyPagination(qb, p)

	if err := qb.Find(&data).Error; err != nil {
		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.TableName))
	}

	totalPage := int(math.Ceil(float64(total) / float64(p.Limit)))

	return &result.ArrayResult[T]{
		Data:       data,
		DataTotal:  total,
		Pagination: pagination.New(p.Pagination, totalPage),
	}, nil
}

// =========================================================
// ==================== BASE GET OBJECT ====================
// =========================================================
func (r *BaseRepository[T]) BaseGetObject(
	ctx context.Context,
	qb *gorm.DB,
	mode string,
	notFoundError bool,
) (*result.ObjectResult[T], error) {

	var data T
	var err error

	switch strings.ToUpper(mode) {
	case "LAST":
		err = qb.Last(&data).Error
	default:
		err = qb.First(&data).Error
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if notFoundError {
				return nil, frameworkError.NewNotFound(fmt.Sprintf("Afwan, Data %s tidak ditemukan.", r.TableName))
			}

			return &result.ObjectResult[T]{}, nil
		}

		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.TableName))
	}

	return &result.ObjectResult[T]{Data: &data}, nil
}

// =====================================================
// ==================== BASE EXISTS ====================
// =====================================================
func (r *BaseRepository[T]) BaseExists(ctx context.Context, qb *gorm.DB) (bool, error) {
	var count int64
	if err := qb.Count(&count).Error; err != nil {
		return false, frameworkError.NewInternalServerError(
			fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mengecek Data %s", r.TableName),
		)
	}
	return count > 0, nil
}

// ====================================================
// ==================== BASE COUNT ====================
// ====================================================
func (r *BaseRepository[T]) BaseCount(ctx context.Context, qb *gorm.DB) (int64, error) {
	var count int64
	if err := qb.Count(&count).Error; err != nil {
		return 0, frameworkError.NewInternalServerError(
			fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat menghitung Data %s", r.TableName),
		)
	}
	return count, nil
}

// =====================================================
// ==================== BASE CREATE ====================
// =====================================================
func (r *BaseRepository[T]) BaseCreate(ctx context.Context, tx *gorm.DB, newData T) error {
	qb := r.db(ctx, tx)

	if err := qb.Create(&newData).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.TableName)
	}

	return nil
}

// ===========================================================
// ==================== BASE CREATE BATCH ====================
// ===========================================================
func (r *BaseRepository[T]) BaseCreateBatch(ctx context.Context, tx *gorm.DB, data []T) error {
	if len(data) == 0 {
		return nil
	}
	qb := r.db(ctx, tx)
	if err := qb.Create(&data).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.TableName)
	}
	return nil
}

// =====================================================
// ==================== BASE UPSERT ====================
// =====================================================
func (r *BaseRepository[T]) BaseUpsert(
	ctx context.Context,
	tx *gorm.DB,
	data T,
	conflictColumns []string,
	updateColumns []string,
) error {
	qb := r.db(ctx, tx)

	cols := make([]clause.Column, 0, len(conflictColumns))
	for _, c := range conflictColumns {
		cols = append(cols, clause.Column{Name: c})
	}

	onConflict := clause.OnConflict{Columns: cols}
	if len(updateColumns) > 0 {
		onConflict.DoUpdates = clause.AssignmentColumns(updateColumns)
	} else {
		onConflict.UpdateAll = true
	}

	if err := qb.Clauses(onConflict).Create(&data).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.TableName)
	}
	return nil
}

// =====================================================
// ==================== BASE UPDATE ====================
// =====================================================
func (r *BaseRepository[T]) BaseUpdate(ctx context.Context, tx *gorm.DB, q query.Query) error {
	qb := r.db(ctx, tx).Model(new(T))

	qb = r.applyWheres(qb, q)

	if err := qb.Updates(q.UpdateData).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.TableName)
	}
	return nil
}

// =====================================================
// ==================== BASE DELETE ====================
// =====================================================
func (r *BaseRepository[T]) BaseDelete(ctx context.Context, tx *gorm.DB, q query.Query) error {
	qb := r.db(ctx, tx)

	qb = r.applyWheres(qb, q)

	if err := qb.Delete(new(T)).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.TableName)
	}
	return nil
}

// ==========================================================
// ==================== BASE TRANSACTION ====================
// ==========================================================
func (r *BaseRepository[T]) BaseTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.DB.WithContext(ctx).Transaction(fn)
}
