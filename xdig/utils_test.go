package xdig_test

import (
	"testing"

	"github.com/raphoester/x/xdig"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

func TestProvideAsWithConcreteTypeAndInterface(t *testing.T) {
	c := dig.New()

	err := xdig.CProvide(c, makeExampleType, xdig.As[exampleInterface]())()
	require.NoError(t, err)

	err = c.Invoke(func(
		t *exampleType,
		i exampleInterface,
	) {
		t.ExampleMethod()
	})

	require.NoError(t, err)
}

func TestProvideValueWithConcreteTypeAndInterface(t *testing.T) {
	c := dig.New()

	typeValue := makeExampleType()
	err := xdig.CProvideValue(c, typeValue, xdig.As[exampleInterface]())()
	require.NoError(t, err)

	err = c.Invoke(func(
		t *exampleType,
		i exampleInterface,
	) {
		t.ExampleMethod()
	})
	require.NoError(t, err)
}

func makeExampleType() *exampleType {
	return &exampleType{}
}

type exampleType struct{}

func (e exampleType) ExampleMethod() {}

type exampleInterface interface {
	ExampleMethod()
}
