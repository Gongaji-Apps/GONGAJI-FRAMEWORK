package query

// Request adalah paket parameter dari handler ke repository untuk endpoint daftar/detail:
// pagination, query (where/wheres/order/havings), dan relasi mana yang perlu di-preload.
// P adalah struct flag preload milik service (mis. shared.Preload), karena daftar relasi
// berbeda tiap domain. Service biasanya membuat alias:
//
//	type Query_Request = query.Request[Preload]
//
// sehingga tanda tangan repository/service yang sudah ada tidak berubah.
type Request[P any] struct {
	Pagination Pagination
	Query      Query
	Preload    P
}
