package evaluate

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
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

func Verify(scheme byte, state types.SmartState, script *ast.Script, object map[string]ast.Expr, this, lastBlock ast.Expr) (bool, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return false, err
	}
	scope := ast.NewScope(script.Version, scheme, state)
	scope.SetTransaction(object)
	scope.SetThis(this)
	scope.SetLastBlockInfo(lastBlock)
	scope.SetHeight(height)

	return Eval(script.Verifier, scope)
}
