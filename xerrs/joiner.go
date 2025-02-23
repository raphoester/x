package xerrs

import "errors"

func NewJoiner() *Joiner {
	return &Joiner{}
}

type Joiner struct {
	err error
}

func (j *Joiner) Join(err error) {
	if j.err == nil {
		j.err = errors.Join(j.err, err)
	}
}

func (j *Joiner) Err() error {
	return j.err
}
