package validator

var messagesID = map[string]string{
	"required": "{field} wajib diisi",
	"email":    "Format {field} tidak valid",
	"min":      "{field} minimal {param}",
	"max":      "{field} maksimal {param}",
	"len":      "Panjang {field} harus {param}",
	"numeric":  "{field} harus berupa angka",
	"uuid":     "Format {field} tidak valid",
	"url":      "Format URL tidak valid",
	"oneof":    "{field} tidak sesuai pilihan",
	"gte":      "{field} harus ≥ {param}",
	"lte":      "{field} harus ≤ {param}",
	// Customize
	"notblank": "{field} tidak boleh hanya spasi",
	"min_int":  "{field} harus ≥ {param}",
}
