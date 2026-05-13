package response

import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/pagination"

type Response struct {
	StatusCode    int              `json:"status_code"`
	Status        bool             `json:"status"`
	Message       string           `json:"message"`
	Data          any              `json:"data,omitempty"`
	DataTotal     *int64           `json:"data_total,omitempty"`
	Pagination    *pagination.Meta `json:"pagination,omitempty"`
	Meta          any              `json:"meta,omitempty"`
	VersionStatus string           `json:"version_status"` // DELETED
}
