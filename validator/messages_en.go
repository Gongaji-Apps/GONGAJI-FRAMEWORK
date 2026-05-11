package validator

var messagesEN = map[string]string{
	"required":         "{field} is required",
	"notblank":         "{field} cannot be blank",
	"email":            "{field} must be a valid email",
	"min":              "{field} must be at least {param}",
	"max":              "{field} must be at most {param}",
	"len":              "{field} must be {param} characters",
	"numeric":          "{field} must be a number",
	"uuid":             "{field} must be a valid UUID",
	"url":              "{field} must be a valid URL",
	"oneof":            "{field} must be one of the allowed values",
	"gte":              "{field} must be ≥ {param}",
	"lte":              "{field} must be ≤ {param}",
	"numeric_nullable": "{field} must not contain characters other than numbers",
	"max_file_size":    "{field} size exceeds {param}MB limit",
	"image":            "{field} must be an image",
	"gt":               "{field} must be greater than {param}",
	"lt":               "{field} must be less than {param}",
	"unique":           "{field} already exists",
}
