package xdig

import (
	"go.uber.org/dig"
)

var container = dig.New()

func As[iface any]() dig.ProvideOption {
	return dig.As(func() *iface { return nil }())
}

func Provide(constructor any, opts ...dig.ProvideOption) func() error {
	return func() error { return container.Provide(constructor, opts...) }
}

func Invoke(function interface{}, opts ...dig.InvokeOption) error {
	return container.Invoke(function, opts...)
}
