package result

import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"

type ObjectResult[T any] struct {
	Data *T `json:"data"`
}

type ArrayResult[T any] struct {
	Data       []T             `json:"data"`
	DataTotal  int64           `json:"data_total"`
	Pagination pagination.Meta `json:"pagination"`
}

// Deprecated: use ObjectResult instead.
type Object_Result[T any] = ObjectResult[T]

// Deprecated: use ArrayResult instead.
type Array_Result[T any] = ArrayResult[T]
