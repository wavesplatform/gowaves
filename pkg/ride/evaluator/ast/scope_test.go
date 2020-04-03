package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func newEmptyScopeV1() Scope {
	return NewScope(1, proto.MainNetScheme, mockstate.State{})
}

func newEmptyScopeV3() Scope {
	return NewScope(3, proto.MainNetScheme, mockstate.State{})
}

func newEmptyScopeV4() Scope {
	return NewScope(4, proto.MainNetScheme, mockstate.State{})
}

func newScopeWithState(s types.SmartState) Scope {
	return NewScope(3, proto.MainNetScheme, s)
}

func TestFuncScope_Clone(t *testing.T) {
	parent := newEmptyScopeV1()
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
	s := newEmptyScopeV1()
	assert.Equal(t, proto.MainNetScheme, s.Scheme())
}
