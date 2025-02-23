package xtime

import "time"

type Provider interface {
	Now() time.Time
}

type RealProvider struct{}

func (RealProvider) Now() time.Time {
	return time.Now()
}

type CustomProvider struct {
	NowFunc func() time.Time
}

func (f CustomProvider) Now() time.Time {
	return f.NowFunc()
}

func NewDefaultFixedProvider() *CustomProvider {
	return &CustomProvider{
		NowFunc: func() time.Time {
			return time.Date(2024, time.October, 10, 0, 0, 0, 0, time.Local)
		},
	}
}
