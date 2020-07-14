package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/op"
)

func TestStack(t *testing.T) {
	s := NewStack()
	s.PushL(1)
	s.PushL(2)
	require.EqualValues(t, ast.NewLong(2), s.Pop())
	require.EqualValues(t, ast.NewLong(1), s.Pop())
}

func TestOpCodeBuilder_StackPushBytes(t *testing.T) {
	bts := op.NewOpCodeBuilder().StackPushBytes([]byte{1, 2, 3}).Code()
	rs, err := EvaluateExpression(bts, nil)
	require.NoError(t, err)
	require.Equal(t, rs, ast.NewBytes([]byte{1, 2, 3}))
}
