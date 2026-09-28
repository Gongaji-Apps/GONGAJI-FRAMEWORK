package validator

// Pesan validasi untuk PENGGUNA (bukan log): {field} diganti label manusiawi
// (errors.FieldLabel — terdaftar via errors.RegisterFieldLabels, atau
// diturunkan dari snake_case). Kunci berakhiran ".str" dipakai bila nilai
// yang divalidasi berupa teks (min/max/len = jumlah karakter).
var messagesID = map[string]string{
	"required":         "{field} wajib diisi.",
	"required_if":      "{field} wajib diisi.",
	"required_with":    "{field} wajib diisi.",
	"required_without": "{field} wajib diisi.",
	"email":            "Format {field} tidak valid. Contoh: nama@contoh.com.",
	"min":              "{field} minimal {param}.",
	"min.str":          "{field} minimal {param} karakter.",
	"max":              "{field} maksimal {param}.",
	"max.str":          "{field} maksimal {param} karakter.",
	"len":              "{field} harus berjumlah {param}.",
	"len.str":          "{field} harus {param} karakter.",
	"numeric":          "{field} harus berupa angka.",
	"number":           "{field} harus berupa angka.",
	"boolean":          "{field} harus bernilai ya atau tidak.",
	"uuid":             "{field} tidak valid. Pilih ulang dari daftar.",
	"url":              "{field} harus berupa alamat web yang valid (diawali https://).",
	"oneof":            "{field} tidak sesuai pilihan yang tersedia.",
	"gte":              "{field} tidak boleh kurang dari {param}.",
	"gte.str":          "{field} minimal {param} karakter.",
	"lte":              "{field} tidak boleh lebih dari {param}.",
	"lte.str":          "{field} maksimal {param} karakter.",
	"gt":               "{field} harus lebih dari {param}.",
	"lt":               "{field} harus kurang dari {param}.",
	"datetime":         "Format {field} tidak valid.",
	"e164":             "Format {field} tidak valid.",
	"dive":             "{field} tidak valid.",
	// Customize
	"notblank":         "{field} tidak boleh hanya berisi spasi.",
	"min_int":          "{field} tidak boleh kurang dari {param}.",
	"max_int":          "{field} tidak boleh lebih dari {param}.",
	"numeric_nullable": "{field} hanya boleh berisi angka.",
	"max_file_size":    "Ukuran {field} melebihi batas {param} MB.",
	"image":            "{field} harus berupa gambar.",
	"unique":           "{field} sudah dipakai.",
	// Cadangan untuk tag yang belum punya pesan.
	"_default": "{field} tidak valid.",
}
