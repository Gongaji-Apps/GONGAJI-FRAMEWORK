package result

import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"

type Object_Result[T any] struct {
	Data *T `json:"data"`
}

type Array_Result[T any] struct {
	Data       []T             `json:"data"`
	Data_Total int64           `json:"data_total"`
	Pagination pagination.Meta `json:"pagination"`
}
