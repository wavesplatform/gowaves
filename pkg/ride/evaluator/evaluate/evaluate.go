package evaluate

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

func Eval(e ast.Expr, s ast.Scope) (bool, error) {
	rs, err := e.Evaluate(s)
	if err != nil {
		return false, err
	}

	b, ok := rs.(*ast.BooleanExpr)
	if !ok {
		return false, errors.Errorf("expected evaluate return *BooleanExpr, but found %T", b)
	}

	return b.Value, nil
}
