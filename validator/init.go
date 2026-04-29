package validator

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func InitValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		runRegistrations(v)
	}
}

// Deprecated: use InitValidator instead.
func Init_Validator() {
	InitValidator()
}
