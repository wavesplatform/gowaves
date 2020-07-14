package vm

import (
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

type Stack struct {
	stack []ast.Expr
}

func NewStack() *Stack {
	return &Stack{stack: make([]ast.Expr, 0, 8)}
}

func (a *Stack) Push(s ast.Expr) {
	a.stack = append(a.stack, s)
}

func (a *Stack) PushL(i int64) {
	a.Push(ast.NewLong(i))
}

func (a *Stack) Pop() ast.Expr {
	last := a.stack[len(a.stack)-1]
	a.stack = a.stack[:len(a.stack)-1]
	return last
}

func (a *Stack) Len() int {
	return len(a.stack)
}
