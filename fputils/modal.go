package fputils

import "errors"

func NewErrModal() *ErrModal {
	return &ErrModal{
		skipOnError: false,
		err:         nil,
	}
}

type ErrModal struct {
	err         error
	skipOnError bool
}

func (e ErrModal) WithSkipOnError() *ErrModal {
	return &ErrModal{
		err:         e.err,
		skipOnError: true,
	}
}

func (e ErrModal) Run(fn func() error) ErrModal {
	if e.skipOnError && e.err != nil {
		return e
	}

	e.err = errors.Join(e.err, fn())
	return e
}

func (e ErrModal) Resolve() error {
	return e.err
}
