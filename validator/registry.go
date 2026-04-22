package validator

import "github.com/go-playground/validator/v10"

type ValidatorRegister func(v *validator.Validate)

var registries []ValidatorRegister

func Register(fn ValidatorRegister) {
	registries = append(registries, fn)
}

func runRegistrations(v *validator.Validate) {
	for _, r := range registries {
		r(v)
	}
}
