package xdig

import (
	"go.uber.org/dig"
)

var container = dig.New()

type ProvideOption struct {
	opt  dig.ProvideOption
	isAs bool
}

func As[iface any]() ProvideOption {
	return ProvideOption{
		opt:  dig.As(func() *iface { return nil }()),
		isAs: true,
	}
}

func ProvideValue[V any](v V) func() error {
	return CProvideValue(container, v)
}

func CProvide(container *dig.Container, constructor any, opts ...ProvideOption) func() error {
	return func() error {
		digOpts := make([]dig.ProvideOption, 0, len(opts))
		asOpts := make([]dig.ProvideOption, 0)
		for _, opt := range opts {
			if opt.isAs {
				asOpts = append(asOpts, opt.opt)
				continue
			}
			digOpts = append(digOpts, opt.opt)
		}

		if len(asOpts) == 0 {
			return container.Provide(constructor, digOpts...)
		}

		// provide once without as for making concrete type always available
		if err := container.Provide(constructor, digOpts...); err != nil {
			return err
		}
		for _, asOpt := range asOpts {
			if err := container.Provide(constructor, append(digOpts, asOpt)...); err != nil {
				return err
			}
		}

		return nil
	}
}

func CProvideValue[V any](c *dig.Container, v V, opts ...ProvideOption) func() error {
	return CProvide(c, func() V { return v }, opts...)
}

func CObtain[V any](c *dig.Container) (V, error) {
	var v V

	err := c.Invoke(func(i V) { v = i })
	return v, err
}
