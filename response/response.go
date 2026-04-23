package response

import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"

type Meta struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Status_Code int              `json:"status_code"`
	Status      bool             `json:"status"`
	Message     string           `json:"message"`
	Data        any              `json:"data,omitempty"`
	Data_Total  *int64           `json:"data_total,omitempty"`
	Pagination  *pagination.Meta `json:"pagination,omitempty"`
	Token       *string          `json:"token,omitempty"`
	Errors      any              `json:"errors,omitempty"`
	Error_Code  *string          `json:"error_code,omitempty"`
	Meta        Meta             `json:"meta"`
}
