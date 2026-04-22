package validator

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func Init_Validator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		runRegistrations(v)
	}
}
