package vm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
	//"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

type Scope struct {
	fns        map[string]Func
	calculated map[string]ast.Expr
	scheme     byte
	state      types.SmartState
}

func merge(x map[string]ast.Expr, y map[string]ast.Expr) map[string]ast.Expr {
	out := make(map[string]ast.Expr)
	for k, v := range x {
		out[k] = v
	}
	for k, v := range y {
		out[k] = v
	}
	return out
}

func NewScope(state types.SmartState, fns map[string]Func, calculated map[string]ast.Expr, scheme byte) *Scope {
	return &Scope{fns: fns, calculated: calculated, scheme: scheme, state: state}
}

func (a *Scope) AddTransaction(tx ast.Object) {
	a.calculated["tx"] = ast.NewObject(tx)
	a.calculated["txId"] = tx["id"]
}

func (a *Scope) SetHeight(height proto.Height) {
	a.calculated["height"] = ast.NewLong(int64(height))
}

func (a *Scope) Call(name string, s *Stack) error {
	fn, ok := a.fns[name]
	if !ok {
		return errors.Errorf("function named '%s' not found", name)
	}
	err := fn(NewContext(s, a.state, a.scheme))
	if err != nil {
		return errors.Wrap(err, name)
	}
	return nil
}

func (a *Scope) PushVariable(name string, position int) {

}

// -1 if no position
//func (a *Scope) Variable(name string) (value StackValue, position int, err error) {
//
//}

func (a *Scope) Calculated(name string) (value ast.Expr, exists bool) {
	v, ok := a.calculated[name]
	return v, ok
}

func BuildScope(state types.SmartState, scheme proto.Scheme, version int) *Scope {

	vars, funcs := expressionsV1()
	return NewScope(state, funcs, vars, scheme)

	//var v func(int) bool
	//switch version {
	//case 1, 2:
	//	v = func(int) bool {
	//		return true
	//	}
	//default:
	//	v = func(l int) bool {
	//		return l <= maxMessageLengthV3
	//	}
	//}
	//
	//out := newScopeImpl(scheme, state, v)

	/*
		var e map[string]Expr
		switch version {
		case 1:
			e = expressionsV1()
			return out.withExprs(e)
		case 2:
			e = expressionsV2()
			return out.withExprs(e)
		case 3:
			e = expressionsV3()
			return out.withExprs(e)
		default:
			e = expressionsV4()
			return out.withExprs(e)
		}*/
}

func expressionsV1() (map[string]ast.Expr, map[string]Func) {
	//return VariablesV1(), functionsV2()
	return make(map[string]ast.Expr), functionsV2()
}

func functionsV2() map[string]Func {
	return map[string]Func{
		"0":       Eq,
		"1":       IsInstanceOf,
		"2":       NativeThrow,
		"100":     NativeSumLong,
		"101":     NativeSubLong,
		"102":     NativeGtLong,
		"103":     GteLong,
		"104":     NativeMulLong,
		"105":     NativeDivLong,
		"106":     NativeModLong,
		"300":     NativeConcatStrings,
		"400":     NativeSizeList,
		"420":     NativeLongToString,
		"401":     NativeGetList,
		"411":     NativeStringToBytes,
		"500":     SigVerifyV2,
		"501":     limitedKeccak256(0),
		"600":     NativeToBase58,
		"601":     NativeFromBase58,
		"1000":    NativeTransactionByID,
		"1040":    NativeDataIntegerFromArray,
		"1043":    NativeDataStringFromArray,
		"1050":    NativeDataIntegerFromState,
		"1051":    NativeDataBooleanFromState,
		"1053":    NativeDataStringFromState,
		"1060":    NativeAddressFromRecipient,
		"1100":    NativeCreateList,
		"$getter": GetterFn,

		"Address":              UserAddress,
		"extract":              UserExtract,
		"isDefined":            UserIsDefined,
		"!":                    UserUnaryNot,
		"!=":                   UserFunctionNeq,
		"wavesBalance":         UserWavesBalanceV3,
		"addressFromPublicKey": UserAddressFromPublicKey,
		"addressFromString":    UserAddressFromString,
	}
}
