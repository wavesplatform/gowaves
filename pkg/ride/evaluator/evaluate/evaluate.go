package evaluate

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func Eval(e ast.Expr, s ast.Scope) (bool, error) {
	rs, err := e.Evaluate(s)
	if err != nil {
		if _, ok := err.(ast.Throw); ok {
			// maybe log error
			return false, nil
		}
		return false, err
	}

	b, ok := rs.(*ast.BooleanExpr)
	if !ok {
		return false, errors.Errorf("expected evaluate return *BooleanExpr, but found %T", b)
	}

	return b.Value, nil
}

func Verify(scheme byte, state types.SmartState, script *ast.Script, transaction proto.Transaction) (bool, error) {
	txVars, err := ast.NewVariablesFromTransaction(scheme, transaction)
	if err != nil {
		return false, err
	}

	height, err := state.NewestHeight()
	if err != nil {
		return false, err
	}

	funcsV2 := ast.VarFunctionsV2
	varsV2 := ast.VariablesV2(txVars, height)

	scope := ast.NewScope(scheme, state, funcsV2, varsV2)

	return Eval(script.Verifier, scope)
}
