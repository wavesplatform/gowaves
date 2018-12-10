package ast

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"io"
)

//
//func mapEval(e Exprs, s Scope) (Exprs, error) {
//	out := make(Exprs, len(e))
//	for i, row := range e {
//		rs, err := row.Evaluate(s.Clone())
//		if err != nil {
//			return nil, errors.Wrapf(err, "error evaluate %d param", i)
//		}
//		out[i] = rs
//	}
//
//	return out, nil
//}

func NativeGtLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeGtLong", func(i int64, i2 int64) Expr {
		return NewBoolean(i > i2)
	}, s, e)
}

func NativeGeLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeGeLong", func(i int64, i2 int64) Expr {
		return NewBoolean(i >= i2)
	}, s, e)
}

func NativeEq(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("NativeEq: invalid params, expected 2, passed %d", l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, "NativeEq evaluate first param")
	}
	second, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, "NativeEq evaluate second param")
	}

	b, err := first.Eq(second)
	return NewBoolean(b), err
}

func NativeGetList(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("NATIVE_GET_LIST: invalid params, expected 2, passed %d", l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, err
	}

	second, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, err
	}

	lst, ok := first.(Exprs)
	if !ok {
		return nil, errors.Errorf("NATIVE_GET_LIST: expected first argument Exprs, got %T", first)
	}

	lng, ok := second.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("NATIVE_GET_LIST: expected second argument *LongExpr, got %T", second)
	}

	if lng.Value < 0 || lng.Value >= int64(len(lst)) {
		return nil, errors.Errorf("NATIVE_GET_LIST: invalid index %d, len %d", lng.Value, len(lst))
	}

	return lst[lng.Value], nil

}

func NativeIsinstanceof(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("NATIVE_GET_LIST: invalid params, expected 2, passed %d", l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, err
	}

	second, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, err
	}

	obj, ok := first.(*ObjectExpr)
	if !ok {
		return nil, errors.Errorf("NATIVE_ISINSTANCEOF: expected first agrumant to be *ObjectExpr, got %T", first)
	}

	str, ok := second.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("NATIVE_ISINSTANCEOF: expected secod agrumant to be *StringExpr, got %T", second)
	}

	val, err := obj.Get(InstanceFieldName)
	if err != nil {
		return nil, errors.Wrap(err, "NATIVE_ISINSTANCEOF")
	}

	strVal, ok := val.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("NATIVE_ISINSTANCEOF: object field %s should be *StringExpr, but found %T", InstanceFieldName, val)
	}

	return NewBoolean(strVal.Value == str.Value), nil
}

func NativeSumLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeSumLong", func(i int64, i2 int64) Expr {
		return NewLong(i + i2)
	}, s, e)
}

func NativeSubLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeSubLong", func(i int64, i2 int64) Expr {
		return NewLong(i - i2)
	}, s, e)
}

func NativeMulLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeMulLong", func(i int64, i2 int64) Expr {
		return NewLong(i * i2)
	}, s, e)
}

func NativeDivLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeDivLong", func(i int64, i2 int64) Expr {
		return NewLong(i / i2)
	}, s, e)
}

func mathLong(funcName string, f func(int64, int64) Expr, s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.Eval(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	first, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}

	second, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}

	return f(first.Value, second.Value), nil
}

// bytes
// signature
// public key
func NativeSigVerify(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 3 {
		return nil, errors.Errorf("NativeSigVerify: invalid params, expected 2, passed %d", l)
	}

	rs, err := e.Eval(s)
	if err != nil {
		return nil, errors.Wrap(err, "NativeSigVerify")
	}

	bytesExpr, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeSigVerify: first argument expects to be *BytesExpr, found %T", rs[0])
	}

	signatureExpr, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeSigVerify: second argument expects to be *BytesExpr, found %T", rs[1])
	}

	pkExpr, ok := rs[2].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeSigVerify: third argument expects to be *BytesExpr, found %T", rs[2])
	}

	pk, err := crypto.NewPublicKeyFromBytes(pkExpr.bytes)
	if err != nil {
		return nil, errors.Wrap(err, "NativeSigVerify")
	}

	signature, err := crypto.NewSignatureFromBytes(signatureExpr.bytes)
	if err != nil {
		return nil, errors.Wrap(err, "NativeSigVerify")
	}

	out := crypto.Verify(pk, signature, bytesExpr.bytes)
	return NewBoolean(out), nil
}

func USER_THROW(s Scope, e Exprs) (Expr, error) {
	return nil, ErrThrow
}

func UserAddressFromString(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 1 {
		return nil, errors.Errorf("UserAddressFromString: invalid params, expected 1, passed %d", l)
	}

	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, "UserAddressFromString")
	}

	str, ok := rs.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("UserAddressFromString: expected first argument to be *StringExpr, found %T", rs)
	}

	addr, err := NewAddressFromString(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, "UserAddressFromString")
	}

	return addr, nil
}

func writeNativeFunction(w io.Writer, id int16, e Exprs) {

	switch id {
	case 0:
		e[0].Write(w)
		fmt.Fprint(w, " == ")
		e[1].Write(w)
	case 103:
		e[0].Write(w)
		fmt.Fprint(w, " >= ")
		e[1].Write(w)

	case 401:
		e[0].Write(w)
		fmt.Fprint(w, "[")
		e[1].Write(w)
		fmt.Fprint(w, "]")
	case 500:
		fmt.Fprint(w, "sigVerify(")
		e[0].Write(w)
		fmt.Fprint(w, ", ")
		e[1].Write(w)
		fmt.Fprint(w, ", ")
		e[2].Write(w)
		fmt.Fprint(w, ")")
	default:
		fmt.Fprintf(w, "FUNCTION_%d(", id)

		for i, arg := range e {
			arg.Write(w)
			if i < len(e)-1 {
				fmt.Fprint(w, ", ")
			}
		}

		fmt.Fprintf(w, ")")
	}

}
