package xvalidator

import (
	"errors"
	"net/mail"
)

func Email(value string) error {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return errors.New("invalid email")
	}

	return nil
}
