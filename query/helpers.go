package query

// Wheres membungkus kondisi SQL tanpa argumen menjadi []Where, untuk kondisi tetap
// seperti "user_active = true" atau "deleted_at IS NULL" yang tidak memuat nilai
// dari klien. Kondisi dengan nilai dari klien harus memakai Where{Query, Args}
// dengan placeholder ? supaya tidak rawan SQL injection.
func Wheres(conditions ...string) []Where {
	wheres := make([]Where, 0, len(conditions))

	for _, condition := range conditions {
		wheres = append(wheres, Where{Query: condition})
	}

	return wheres
}

// UUIDOrNil mengembalikan nil bila value kosong, selain itu value apa adanya.
// Kolom uuid Postgres menolak string kosong ("" bukan uuid yang sah), sedangkan
// NULL sebagai argumen cukup berarti "tidak ada" — mis. filter wishlist untuk
// admin yang tidak membawa identitas user.
func UUIDOrNil(value string) any {
	if value == "" {
		return nil
	}

	return value
}
