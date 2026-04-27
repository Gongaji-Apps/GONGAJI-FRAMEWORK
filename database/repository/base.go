package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"gorm.io/gorm"

	frameworkError "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"
)

type Base_Repository[T any] struct {
	DB         *gorm.DB
	Table_Name string
	Debug      bool
}

func New_Base_Repository[T any](db *gorm.DB, table string) Base_Repository[T] {
	return Base_Repository[T]{
		DB:         db,
		Table_Name: table,
		Debug:      strings.ToLower(os.Getenv("APP_DEBUG")) == "TRUE",
	}
}

// ========================================================
// ==================== BASE GET QUERY ====================
// ========================================================
func (r *Base_Repository[T]) db(ctx context.Context, tx *gorm.DB) *gorm.DB {
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
func (r *Base_Repository[T]) applyWheres(qb *gorm.DB, q query.Query) *gorm.DB {
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
func (r *Base_Repository[T]) applyHavings(qb *gorm.DB, q query.Query) *gorm.DB {
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
func (r *Base_Repository[T]) applyPagination(qb *gorm.DB, p query.Pagination) *gorm.DB {
	p = normalizePagination(p)

	return qb.Limit(p.Limit).Offset((p.Pagination - 1) * p.Limit)
}

// ==========================================================
// ==================== BASE BUILD QUERY ====================
// ==========================================================
func (r *Base_Repository[T]) Base_Build_Query(
	ctx context.Context,
	q query.Query,
	custom func(*gorm.DB) *gorm.DB,
) *gorm.DB {
	qb := r.db(ctx, nil).Model(new(T))

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
func (r *Base_Repository[T]) Base_Get_Array(
	ctx context.Context,
	qb *gorm.DB,
	p query.Pagination,
) (*result.Array_Result[T], error) {
	var data []T
	var total int64

	if err := qb.Count(&total).Error; err != nil {
		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.Table_Name))
	}

	qb = r.applyPagination(qb, p)

	if err := qb.Find(&data).Error; err != nil {
		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.Table_Name))
	}

	totalPage := int(math.Ceil(float64(total) / float64(p.Limit)))

	return &result.Array_Result[T]{
		Data:       data,
		Data_Total: total,
		Pagination: pagination.New(p.Pagination, totalPage),
	}, nil
}

// =========================================================
// ==================== BASE GET OBJECT ====================
// =========================================================
func (r *Base_Repository[T]) Base_Get_Object(
	ctx context.Context,
	qb *gorm.DB,
	mode string,
	notFoundError bool,
) (*result.Object_Result[T], error) {

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
				return nil, frameworkError.NewNotFound(fmt.Sprintf("Afwan, Data %s tidak ditemukan.", r.Table_Name))
			}

			return &result.Object_Result[T]{}, nil
		}

		return nil, frameworkError.NewInternalServerError(fmt.Sprintf("[Internal Server Error] Afwan, Kami mengalami masalah saat mendapatkan Data %s", r.Table_Name))
	}

	return &result.Object_Result[T]{Data: &data}, nil
}

// =====================================================
// ==================== BASE CREATE ====================
// =====================================================
func (r *Base_Repository[T]) Base_Create(ctx context.Context, tx *gorm.DB, newData T) error {
	qb := r.db(ctx, tx)

	// Create
	if err := qb.Create(&newData).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.Table_Name)
	}

	return nil
}

// =====================================================
// ==================== BASE UPDATE ====================
// =====================================================
func (r *Base_Repository[T]) Base_Update(ctx context.Context, tx *gorm.DB, v_query query.Query) error {
	qb := r.db(ctx, tx).Model(new(T))

	// Apply Wheres
	qb = r.applyWheres(qb, v_query)

	// Update
	if err := qb.Updates(v_query.Update_Data).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.Table_Name)
	}
	return nil
}

// =====================================================
// ==================== BASE DELETE ====================
// =====================================================
func (r *Base_Repository[T]) Base_Delete(ctx context.Context, tx *gorm.DB, v_query query.Query) error {
	qb := r.db(ctx, tx)

	// Apply Wheres
	qb = r.applyWheres(qb, v_query)

	// Delete
	if err := qb.Delete(new(T)).Error; err != nil {
		return frameworkError.NormalizeDBError(err, r.Table_Name)
	}
	return nil
}
