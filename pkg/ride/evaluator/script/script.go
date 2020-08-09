package messages

import (
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/vm"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type Script struct {
	Version    int
	HasBlockV2 bool
	HasArrays  bool
	Verifier   ast.Expr
	DApp       DApp
	dApp       bool
	VmCode     []byte
}

func (a *Script) HasVerifier() bool {
	if a.IsDapp() {
		return a.DApp.Verifier != nil
	}
	return a.Verifier != nil
}

func (a *Script) IsDapp() bool {
	return a.dApp
}

func (a *Script) CallFunction(scheme proto.Scheme, state types.SmartState, tx *proto.InvokeScriptWithProofs, this, lastBlock ast.Expr) (bool, []proto.ScriptAction, error) {
	if !a.IsDapp() {
		return false, nil, errors.New("can't call Script.CallFunction on non DApp")
	}
	txObj, err := ast.NewVariablesFromTransaction(scheme, tx)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to convert transaction")
	}
	name := tx.FunctionCall.Name
	if name == "" && tx.FunctionCall.Default {
		name = "default"
	}
	fn, ok := a.DApp.CallableFuncs[name]
	if !ok {
		return false, nil, errors.Errorf("Callable function named '%s' not found", name)
	}
	invoke, err := a.buildInvocation(scheme, tx)
	if err != nil {
		return false, nil, err
	}
	height, err := state.AddingBlockHeight()
	if err != nil {
		return false, nil, err
	}
	scope := ast.NewScope(a.Version, scheme, state)
	scope.SetThis(this)
	scope.SetLastBlockInfo(lastBlock)
	scope.SetHeight(height)
	scope.SetTransaction(txObj)

	// assign of global vars and function
	for _, expr := range a.DApp.Declarations {
		_, err = expr.Evaluate(scope)
		if err != nil {
			return true, nil, errors.Wrap(err, "Script.CallFunction")
		}
	}

	if len(fn.FuncDecl.Args) != len(tx.FunctionCall.Arguments) {
		return true, nil, errors.Errorf("invalid func '%s' args count, expected %d, got %d", fn.FuncDecl.Name, len(fn.FuncDecl.Args), len(tx.FunctionCall.Arguments))
	}
	// pass function arguments
	curScope := scope.Clone()
	for i, arg := range tx.FunctionCall.Arguments {
		argExpr, err := protoArgToArgExpr(arg)
		if err != nil {
			return true, nil, errors.Wrap(err, "Script.CallFunction")
		}
		curScope.AddValue(fn.FuncDecl.Args[i], argExpr)
	}
	// invocation type
	curScope.AddValue(fn.AnnotationInvokeName, invoke)

	rs, err := fn.FuncDecl.Body.Evaluate(curScope)
	if err != nil {
		return true, nil, errors.Wrap(err, "Script.CallFunction")
	}

	switch t := rs.(type) {
	case *ast.WriteSetExpr:
		actions, err := t.ToActions()
		return true, actions, err
	case *ast.TransferSetExpr:
		actions, err := t.ToActions()
		return true, actions, err
	case *ast.ScriptResultExpr:
		actions, err := t.ToActions()
		return true, actions, err
	case ast.Exprs:
		res := make([]proto.ScriptAction, 0, len(t))
		for _, e := range t {
			ae, ok := e.(ast.Actionable)
			if !ok {
				return true, nil, errors.Errorf("Script.CallFunction: fail to convert result to action")
			}
			action, err := ae.ToAction(tx.ID)
			if err != nil {
				return true, nil, errors.Wrap(err, "Script.CallFunction: fail to convert result to action")
			}
			res = append(res, action)
		}
		return true, res, nil
	default:
		return true, nil, errors.Errorf("Script.CallFunction: unexpected result type '%T'", t)
	}
}

func (a *Script) Verify(scheme byte, state types.SmartState, object map[string]ast.Expr, this, lastBlock ast.Expr) (ast.Result, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return ast.Result{}, err
	}
	if a.IsDapp() {
		if a.DApp.Verifier == nil {
			return ast.Result{}, errors.New("verify function not defined")
		}
		scope := ast.NewScope(a.Version, scheme, state)
		scope.SetThis(this)
		scope.SetLastBlockInfo(lastBlock)
		scope.SetHeight(height)

		fn := a.DApp.Verifier
		// pass function arguments
		curScope := scope //.Clone()
		// annotated tx type
		curScope.AddValue(fn.AnnotationInvokeName, ast.NewObject(object))
		// here should be only assign of vars and function
		for _, expr := range a.DApp.Declarations {
			_, err = expr.Evaluate(curScope)
			if err != nil {
				return ast.Result{}, errors.Wrap(err, "Script.Verify")
			}
		}
		return EvalAsResult(fn.FuncDecl.Body, curScope)
	} else {
		scope := ast.NewScope(a.Version, scheme, state)
		scope.SetTransaction(object)
		scope.SetThis(this)
		scope.SetLastBlockInfo(lastBlock)
		scope.SetHeight(height)

		if a.Version == 1 && len(a.VmCode) > 0 {
			vmScope := vm.BuildScope(state, scheme, a.Version)
			vmScope.AddTransaction(object)
			zap.S().Debugf("object=== %+v", object)
			zap.S().Debugf("object=== %q", object)
			vmScope.SetHeight(height)
			rs, err := vm.EvaluateExpressionAsBoolean(a.VmCode, vmScope)
			if err != nil {
				zap.S().Debugf("EvaluateExpressionAsBoolean: %v, %s", rs, err)
				return ast.Result{}, err
			}
		}

		scriptrs, err := EvalAsResult(a.Verifier, scope)
		zap.S().Debugf("EvalAsResult: %v, %s", scriptrs, err)

		zap.S().Debugf("Object: %s", base58.Encode(object["id"].(*ast.BytesExpr).Value))

		return scriptrs, err
	}
}

//type object map[string]ast.Expr

func (a *Script) buildInvocation(scheme proto.Scheme, tx *proto.InvokeScriptWithProofs) (*ast.InvocationExpr, error) {
	fields := ast.Object{}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	fields["caller"] = ast.NewAddressFromProtoAddress(addr)
	fields["callerPublicKey"] = ast.NewBytes(tx.SenderPK.Bytes())

	switch a.Version {
	case 4:
		payments := ast.NewExprs(nil)
		for _, p := range tx.Payments {
			payments = append(ast.NewExprs(ast.NewAttachedPaymentExpr(ast.MakeOptionalAsset(p.Asset), ast.NewLong(int64(p.Amount)))), payments...)
		}
		fields["payments"] = payments
	default:
		fields["payment"] = ast.NewUnit()
		if len(tx.Payments) > 0 {
			fields["payment"] = ast.NewAttachedPaymentExpr(ast.MakeOptionalAsset(tx.Payments[0].Asset), ast.NewLong(int64(tx.Payments[0].Amount)))
		}
	}
	fields["transactionId"] = ast.NewBytes(tx.ID.Bytes())
	fields["fee"] = ast.NewLong(int64(tx.Fee))
	fields["feeAssetId"] = ast.MakeOptionalAsset(tx.FeeAsset)

	return ast.NewInvocation(fields), nil
}

func protoArgToArgExpr(arg proto.Argument) (ast.Expr, error) {
	switch a := arg.(type) {
	case *proto.IntegerArgument:
		return &ast.LongExpr{Value: a.Value}, nil
	case *proto.BooleanArgument:
		return &ast.BooleanExpr{Value: a.Value}, nil
	case *proto.StringArgument:
		return &ast.StringExpr{Value: a.Value}, nil
	case *proto.BinaryArgument:
		return &ast.BytesExpr{Value: a.Value}, nil
	default:
		return nil, errors.New("unknown argument type")
	}
}

func (a *Script) Eval(s ast.Scope) (ast.Result, error) {
	return EvalAsResult(a.Verifier, s)
}

func EvalAsResult(e ast.Expr, s ast.Scope) (ast.Result, error) {
	rs, err := e.Evaluate(s)
	if err != nil {
		if throw, ok := err.(ast.Throw); ok {
			return ast.Result{
				Throw:   true,
				Message: throw.Message,
			}, nil
		}
		return ast.Result{}, err
	}
	b, ok := rs.(*ast.BooleanExpr)
	if !ok {
		return ast.Result{}, errors.Errorf("expected evaluate return *BooleanExpr, but found %T", b)
	}
	return ast.Result{Value: b.Value}, nil
}
