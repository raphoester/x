package xvalidator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

func ForbidZeroValues(v ...any) error {
	v10 := validator.New()
	for i, val := range v {
		if err := v10.Var(val, "required"); err != nil {
			return fmt.Errorf("value at index %d is required", i)
		}
	}
	return nil
}
