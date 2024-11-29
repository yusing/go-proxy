package utils

import (
	"github.com/go-playground/validator/v10"
	E "github.com/yusing/go-proxy/internal/error"
)

var validate = validator.New()

var ErrValidationError = E.New("validation error")

func Validator() *validator.Validate {
	return validate
}
