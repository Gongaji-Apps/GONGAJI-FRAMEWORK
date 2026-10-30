package binding

import (
	"github.com/gin-gonic/gin"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/converter"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
)

// Preloadable dipenuhi struct domain yang punya ToPreload() — memetakan field
// `preload_*` dari query string ke struct flag preload service (P).
type Preloadable[P any] interface {
	ToPreload() P
}

// ListQuery mem-bind query string ke payload domain T DAN ke query.Pagination, lalu
// mengembalikan payload beserta query.Request[P] yang pagination-nya sudah terisi.
// Mengganti dua kali Query[...] + inisialisasi Request di tiap handler daftar.
//
// Pola handler:
//
//	payload, request, err := binding.ListQuery[domain.Category_Mstr, shared.Preload](ctx)
//	if err != nil { response.Error(ctx, err); return }
//	// ... kondisi khusus dari payload (ILIKE, role) → request.Query.Wheres, lalu kosongkan field-nya
//	if err := binding.FinishList(&request, payload); err != nil { response.Error(ctx, err); return }
func ListQuery[T any, P any](ctx *gin.Context) (T, query.Request[P], error) {
	var request query.Request[P]

	payload, err := Query[T](ctx)
	if err != nil {
		return payload, request, err
	}

	pagination, err := Query[query.Pagination](ctx)
	if err != nil {
		return payload, request, err
	}

	request.Pagination = pagination

	return payload, request, nil
}

// FinishList mengisi request.Query.Where dari field payload yang terisi
// (converter.StructToMap) dan request.Preload dari payload.ToPreload(). Panggil
// SETELAH kondisi khusus diproses dan field-nya dikosongkan, supaya tidak ikut
// menjadi kondisi `=` di WHERE.
func FinishList[T Preloadable[P], P any](request *query.Request[P], payload T) error {
	if err := FinishWhere(request, payload); err != nil {
		return err
	}

	request.Preload = payload.ToPreload()

	return nil
}

// FinishWhere seperti FinishList tanpa preload — untuk payload yang tidak punya ToPreload().
func FinishWhere[P any](request *query.Request[P], payload any) error {
	where, err := converter.StructToMap(payload)
	if err != nil {
		return err
	}

	request.Query.Where = where

	return nil
}
