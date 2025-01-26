package utils

import (
	"github.com/go-playground/validator/v10"
	E "github.com/yusing/go-proxy/internal/error"
)

var validate = validator.New()

var ErrValidationError = E.New("validation error")

type CustomValidator interface {
	Validate() E.Error
}

func Validator() *validator.Validate {
	return validate
}

func MustRegisterValidation(tag string, fn validator.Func) {
	err := validate.RegisterValidation(tag, fn)
	if err != nil {
		panic(err)
	}
}
