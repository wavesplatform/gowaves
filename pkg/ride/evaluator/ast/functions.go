package ast

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"io"
	"math/big"
	"unicode/utf8"
)

const MaxBytesResult = 65536
const MaxStringResult = 32767
const DefaultThrowMessage = "Explicit script termination"

type Throw struct {
	Message string
}

func (a Throw) Error() string {
	return a.Message
}

func modDivision(x int64, y int64) int64 {
	return x - floorDiv(x, y)*y
}

func floorDiv(x int64, y int64) int64 {
	r := x / y
	if (x^y) < 0 && (r*y != x) {
		r--
	}
	return r
}

func NativeGtLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeGtLong", func(i int64, i2 int64) (Expr, error) {
		return NewBoolean(i > i2), nil
	}, s, e)
}

func NativeGeLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeGeLong", func(i int64, i2 int64) (Expr, error) {
		return NewBoolean(i >= i2), nil
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
	funcName := "NativeGetList"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
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
		return nil, errors.Errorf("%s: expected first argument Exprs, got %T", funcName, first)
	}

	lng, ok := second.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument *LongExpr, got %T", funcName, second)
	}

	if lng.Value < 0 || lng.Value >= int64(len(lst)) {
		return nil, errors.Errorf("%s: invalid index %d, len %d", funcName, lng.Value, len(lst))
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
	return mathLong("NativeSumLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i + i2), nil
	}, s, e)
}

func NativeSubLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeSubLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i - i2), nil
	}, s, e)
}

func NativeMulLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeMulLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i * i2), nil
	}, s, e)
}

func NativeDivLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeDivLong", func(i int64, i2 int64) (Expr, error) {
		if i2 == 0 {
			return nil, errors.New("zero division")
		}
		return NewLong(i / i2), nil
	}, s, e)
}

func NativeModLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeDivLong", func(i int64, i2 int64) (Expr, error) {
		if i2 == 0 {
			return nil, errors.New("zero division")
		}
		return NewLong(modDivision(i, i2)), nil
	}, s, e)
}

// Multiply and dividion with big integer intermediate representation
func NativeFractionLong(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeFractionLong"

	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	originalValue, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}

	multiplyer, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}

	divider, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s third argument expected to be *LongExpr, got %T", funcName, rs[2])
	}

	if divider.Value == 0 {
		return nil, errors.Errorf("%s division by zero", funcName)
	}

	a := big.NewInt(0)
	a.Mul(big.NewInt(originalValue.Value), big.NewInt(multiplyer.Value))
	a.Div(a, big.NewInt(divider.Value))

	if !a.IsInt64() {
		return nil, errors.Errorf("%s long overflow %s", funcName, a.String())
	}

	return NewLong(a.Int64()), nil
}

func mathLong(funcName string, f func(int64, int64) (Expr, error), s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s)
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

	return f(first.Value, second.Value)
}

// bytes
// signature
// public key
func NativeSigVerify(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 3 {
		return nil, errors.Errorf("NativeSigVerify: invalid params, expected 2, passed %d", l)
	}

	rs, err := e.EvaluateAll(s)
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

func NativeKeccak256(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 1 {
		return nil, errors.Errorf("NativeKeccak256: invalid params, expected 1, passed %d", l)
	}

	val, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "NativeKeccak256")
	}

	bts, ok := val.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeKeccak256: expected first argument to be *BytesExpr, found %T", val)
	}

	d := crypto.Keccak256(bts.bytes)
	return NewBytes(d.Bytes()), nil
}

func NativeBlake2b256(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 1 {
		return nil, errors.Errorf("NativeBlake2b256: invalid params, expected 1, passed %d", l)
	}

	val, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "NativeBlake2b256")
	}

	bts, ok := val.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeBlake2b256: expected first argument to be *BytesExpr, found %T", val)
	}

	d, err := crypto.FastHash(bts.bytes)
	if err != nil {
		return nil, errors.Wrap(err, "NativeBlake2b256")
	}
	return NewBytes(d.Bytes()), nil
}

func NativeSha256(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 1 {
		return nil, errors.Errorf("NativeSha256: invalid params, expected 1, passed %d", l)
	}

	val, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "NativeSha256")
	}

	bts, ok := val.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("NativeSha256: expected first argument to be *BytesExpr, found %T", val)
	}

	h := sha256.New()
	h.Write(bts.bytes)
	d := h.Sum(nil)

	return NewBytes(d), nil
}

func NativeTransactionHeightByID(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeTransactionHeightByID"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, got %T", funcName, rs)
	}

	sign, err := crypto.NewSignatureFromBytes(bts.bytes)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	height, err := s.State().TransactionHeightByID(&sign)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewLong(int64(height)), nil
}

func NativeTransactionByID(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeTransactionByID"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, got %T", funcName, rs)
	}

	sign, err := crypto.NewSignatureFromBytes(bts.bytes)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	tx, err := s.State().TransactionByID(&sign)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	vars, err := NewVariablesFromTransaction(s.Scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewObject(vars), nil
}

func NativeSizeBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeSizeBytes"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, rs)
	}

	return NewLong(int64(len(bts.bytes))), nil
}

func NativeTakeBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeTakeBytes"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, rs[0])
	}

	length, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *LongExpr, found %T", funcName, rs[1])
	}

	l := int(length.Value)

	if l >= len(bts.bytes) {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	out := make([]byte, l)
	copy(out, bts.bytes[:l])

	return NewBytes(out), nil
}

func NativeDropBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDropBytes"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, rs[0])
	}

	length, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *LongExpr, found %T", funcName, rs[1])
	}

	l := int(length.Value)

	if l >= len(bts.bytes) {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	out := make([]byte, len(bts.bytes)-l)
	copy(out, bts.bytes[l:])

	return NewBytes(out), nil
}

func NativeConcatBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDropBytes"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	prefix, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, rs[0])
	}

	suffix, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *BytesExpr, found %T", funcName, rs[1])
	}

	l := len(prefix.bytes) + len(suffix.bytes)

	if l > MaxBytesResult {
		return nil, errors.Errorf("%s byte length %d is greater than max %d", funcName, l, MaxBytesResult)
	}

	out := make([]byte, l)
	out = append(out[:0], prefix.bytes...)
	out = append(out[:len(prefix.bytes)], suffix.bytes...)

	return NewBytes(out), nil
}

func NativeConcatStrings(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeConcatStrings"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	prefix, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, rs[0])
	}

	suffix, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *StringExpr, found %T", funcName, rs[1])
	}

	l := len(prefix.Value) + len(suffix.Value)

	if l > MaxBytesResult {
		return nil, errors.Errorf("%s byte length %d is greater than max %d", funcName, l, MaxBytesResult)
	}

	out := prefix.Value + suffix.Value
	lengthInRunes := utf8.RuneCountInString(out)
	if lengthInRunes > MaxStringResult {
		return nil, errors.Errorf("%s string length %d is greater than max %d", funcName, lengthInRunes, MaxStringResult)
	}

	return NewString(out), nil
}

func NativeTakeStrings(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeTakeStrings"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, rs[0])
	}

	length, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *LongExpr, found %T", funcName, rs[1])
	}

	runeStr := []rune(str.Value)
	runeLen := len(runeStr)
	l := int(length.Value)

	if l >= runeLen {
		return nil, errors.Errorf("%s index %d out of range", funcName, l)
	}

	out := make([]rune, l)
	copy(out, runeStr[:l])

	return NewString(string(out)), nil
}

func NativeDropStrings(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDropStrings"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, rs[0])
	}

	length, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *LongExpr, found %T", funcName, rs[1])
	}

	runeStr := []rune(str.Value)
	runeLen := len(runeStr)
	l := int(length.Value)

	if l >= runeLen {
		return nil, errors.Errorf("%s index %d out of range", funcName, l)
	}

	out := make([]rune, runeLen-l)
	copy(out, runeStr[l:])

	return NewString(string(out)), nil
}

func NativeSizeString(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeSizeBytes"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := rs.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, rs)
	}

	return NewLong(int64(utf8.RuneCountInString(str.Value))), nil
}

func NativeSizeList(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeSizeList"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	// optimize not evaluate inner list
	if v, ok := e[0].(Exprs); ok {
		return NewLong(int64(len(v))), nil
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	lst, ok := rs.(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument Exprs, got %T", funcName, rs)
	}

	return NewLong(int64(len(lst))), nil
}

func NativeLongToBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeLongToBytes"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	long, ok := first.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument *LongExpr, got %T", funcName, first)
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, uint64(long.Value))

	return NewBytes(out), nil
}

func NativeStringToBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeStringToBytes"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument *StringExpr, got %T", funcName, first)
	}

	return NewBytes([]byte(str.Value)), nil
}

func NativeBooleanToBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeBooleanToBytes"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	b, ok := rs.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: exptected first argument to be *BooleanExpr, got %T", funcName, rs)
	}

	if b.Value {
		return NewBytes([]byte{1}), nil
	} else {
		return NewBytes([]byte{0}), nil
	}
}

func NativeThrow(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeThrow"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *StringExpr, found %T", funcName, first)
	}

	return nil, Throw{
		Message: str.Value,
	}
}

func NativeLongToString(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeLongToString"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	long, ok := first.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *LongExpr, found %T", funcName, first)
	}

	return NewString(fmt.Sprintf("%d", long.Value)), nil
}

func NativeBooleanToString(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeBooleanToString"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	b, ok := first.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BooleanExpr, found %T", funcName, first)
	}

	if b.Value {
		return NewString("true"), nil
	} else {
		return NewString("false"), nil
	}
}

func UserThrow(s Scope, e Exprs) (Expr, error) {
	return nil, Throw{Message: DefaultThrowMessage}
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
	case 2:
		fmt.Fprint(w, "throw(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 103:
		e[0].Write(w)
		fmt.Fprint(w, " >= ")
		e[1].Write(w)

	case 200:
		fmt.Fprint(w, "size(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 203:
		e[0].Write(w)
		fmt.Fprint(w, " + ")
		e[1].Write(w)
	case 300:
		e[0].Write(w)
		fmt.Fprint(w, " + ")
		e[1].Write(w)
	case 305:
		fmt.Fprint(w, "size(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 401:
		e[0].Write(w)
		fmt.Fprint(w, "[")
		e[1].Write(w)
		fmt.Fprint(w, "]")
	case 410, 411, 412:
		fmt.Fprint(w, "toBytes(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 420, 421:
		fmt.Fprint(w, "toString(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 500:
		fmt.Fprint(w, "sigVerify(")
		e[0].Write(w)
		fmt.Fprint(w, ", ")
		e[1].Write(w)
		fmt.Fprint(w, ", ")
		e[2].Write(w)
		fmt.Fprint(w, ")")

	case 1000:
		fmt.Fprint(w, "transactionById(")
		e[0].Write(w)
		fmt.Fprint(w, ")")
	case 1001:
		fmt.Fprint(w, "transactionHeightById(")
		e[0].Write(w)
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
