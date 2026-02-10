package validator

import (
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func Struct(data interface{}) error {
	return validate.Struct(data)
}

func Var(field interface{}, tag string) error {
	return validate.Var(field, tag)
}
