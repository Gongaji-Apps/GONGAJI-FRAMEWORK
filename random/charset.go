package random

const (
	Alpha             = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaUpper        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaLower        = "abcdefghijklmnopqrstuvwxyz"
	Numeric           = "0123456789"
	AlphaNumeric      = Alpha + Numeric
	AlphaNumericUpper = AlphaUpper + Numeric
	AlphaNumericLower = AlphaLower + Numeric
	Character         = "!@#$%^&*()-_=+[]{}|;:,.<>/?"
	All               = AlphaNumeric + Character
)
