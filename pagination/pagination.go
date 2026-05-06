package pagination

type Meta struct {
	Current int  `json:"current"`
	Next    *int `json:"next"`
	Total   int  `json:"total"`
	Last    bool `json:"last"`
}