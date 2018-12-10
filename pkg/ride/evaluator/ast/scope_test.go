package ast

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func newEmptyScope() Scope {
	return NewScope(EmptyFuncScope(), nil)
}

func TestFuncScope_Clone(t *testing.T) {

	parent := newEmptyScope()
	parent.AddValue("x", NewBoolean(true))
	e, _ := parent.Value("x")
	assert.Equal(t, NewBoolean(true), e)

	child := parent.Clone()
	e2, _ := child.Value("x")
	assert.Equal(t, NewBoolean(true), e2)

	child.AddValue("x", NewBoolean(false))
	e3, _ := child.Value("x")
	assert.Equal(t, NewBoolean(false), e3)

	parent.Value("x")
	e4, _ := parent.Value("x")
	assert.Equal(t, NewBoolean(true), e4)

	// add value to child, parent has no value
	child.AddValue("y", NewLong(5))
	_, ok := parent.Value("y")
	assert.Equal(t, false, ok)

}
