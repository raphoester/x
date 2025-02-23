package xvalidator

import "github.com/go-playground/validator"

func Struct(in any) error {
	v := validator.New()
	return v.Struct(in)
}
