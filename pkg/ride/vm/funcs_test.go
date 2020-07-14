package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

func TestGteLong(t *testing.T) {
	s := NewStack()
	s.PushL(10)
	s.PushL(6)

	err := GteLong(s)
	require.NoError(t, err)

	require.NoError(t, with(s, func(i bool) error {
		require.True(t, i)
		return nil
	}))

	s.PushL(5)
	s.PushL(5)
	_ = GteLong(s)
	require.Equal(t, ast.NewBoolean(true), s.Pop())

	s.PushL(4)
	s.PushL(5)
	_ = GteLong(s)
	require.Equal(t, ast.NewBoolean(false), s.Pop())
}

func TestIsInstanceOf(t *testing.T) {
	s := NewStack()
	s.Push(ast.NewLong(5))
	s.Push(ast.NewString(ast.NewLong(5).InstanceOf()))
	require.NoError(t, IsInstanceOf(s))
	require.Equal(t, ast.NewBoolean(true), s.Pop())

	s.Push(ast.NewString(""))
	s.Push(ast.NewString(ast.NewLong(5).InstanceOf()))
	require.NoError(t, IsInstanceOf(s))
	require.Equal(t, ast.NewBoolean(false), s.Pop())
}
