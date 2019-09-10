package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func newEmptyScope() Scope {
	return NewScope(proto.MainNetScheme, mockstate.State{}, EmptyFunctions(), nil)
}

func newScopeWithState(s types.SmartState) Scope {
	return NewScope(proto.MainNetScheme, s, EmptyFunctions(), nil)
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

func TestScopeImpl_Scheme(t *testing.T) {
	s := newEmptyScope()
	assert.Equal(t, proto.MainNetScheme, s.Scheme())
}
