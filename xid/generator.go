package xid

import "github.com/google/uuid"

type Generator interface {
	Generate() string
}

type RandomGenerator struct {
}

func (r RandomGenerator) Generate() string {
	return uuid.New().String()
}

type CustomGenerator struct {
	GenFunc func() string
}

func (c CustomGenerator) Generate() string { return c.GenFunc() }

func NewDefaultFixedGenerator() Generator {
	return CustomGenerator{GenFunc: func() string { return "deadbeef-defa-defa-defa-deadbeefcafe" }}
}
