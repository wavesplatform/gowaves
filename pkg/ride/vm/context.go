package vm

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Context interface {
	Pop() ast.Expr
	Push(value ast.Expr) error
	State() types.SmartState
	Scheme() byte
}

type stack interface {
	Pop() ast.Expr
	Push(value ast.Expr)
}

type ContextImpl struct {
	stack  stack
	state  types.SmartState
	scheme proto.Scheme
}

func (a ContextImpl) Pop() ast.Expr {
	return a.stack.Pop()
}

func (a *ContextImpl) Push(value ast.Expr) error {
	a.stack.Push(value)
	return nil
}

func (a ContextImpl) State() types.SmartState {
	return a.state
}

func (a ContextImpl) Scheme() byte {
	return a.scheme
}

func NewContext(stack2 stack, state types.SmartState, chainID proto.Scheme) *ContextImpl {
	return &ContextImpl{
		stack:  stack2,
		state:  state,
		scheme: chainID,
	}
}
