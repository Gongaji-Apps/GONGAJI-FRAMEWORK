package query

type Pagination struct {
	Limit      int `json:"limit"  form:"limit"`
	Pagination int `json:"pagination" form:"pagination"`
}
