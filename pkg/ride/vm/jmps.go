package vm

import "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"

type Jmps struct {
	stack *Stack
}

func NewJmps() *Jmps {
	return &Jmps{stack: NewStack()}
}

func (a *Jmps) Push(to int) {
	a.stack.PushL(int64(to))
}

func (a *Jmps) Pop() int {
	return int(a.stack.Pop().(*ast.LongExpr).Value)
}

func (a *Jmps) PopSafe() (int, bool) {
	if a.stack.Len() > 0 {
		return int(a.stack.Pop().(*ast.LongExpr).Value), true
	}
	return 0, false
}
