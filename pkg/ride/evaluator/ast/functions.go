package ast

import (
	"bytes"
	. "crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"

	"github.com/ericlagergren/decimal"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
)

const (
	MaxBytesResult      = 65536
	MaxStringResult     = 32767
	MaxBytesToVerify    = 32 * 1024
	DefaultThrowMessage = "Explicit script termination"
)

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

func Params(params ...Expr) Exprs {
	return NewExprs(params...)
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

// Equality
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

// Get list element by position
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

// Internal function to check value type
func NativeIsInstanceOf(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeIsInstanceOf"

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

	str, ok := second.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be *StringExpr, got %T", funcName, second)
	}

	strVal := first.InstanceOf()
	return NewBoolean(strVal == str.Value), nil
}

// Integer sum
func NativeSumLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeSumLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i + i2), nil
	}, s, e)
}

// Integer substitution
func NativeSubLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeSubLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i - i2), nil
	}, s, e)
}

// Integer multiplication
func NativeMulLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeMulLong", func(i int64, i2 int64) (Expr, error) {
		return NewLong(i * i2), nil
	}, s, e)
}

// Integer division
func NativeDivLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeDivLong", func(i int64, i2 int64) (Expr, error) {
		if i2 == 0 {
			return nil, errors.New("zero division")
		}
		return NewLong(i / i2), nil
	}, s, e)
}

// Modulo
func NativeModLong(s Scope, e Exprs) (Expr, error) {
	return mathLong("NativeDivLong", func(i int64, i2 int64) (Expr, error) {
		if i2 == 0 {
			return nil, errors.New("zero division")
		}
		return NewLong(modDivision(i, i2)), nil
	}, s, e)
}

// Multiply and division with big integer intermediate representation
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

	multiplier, ok := rs[1].(*LongExpr)
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
	a.Mul(big.NewInt(originalValue.Value), big.NewInt(multiplier.Value))
	a.Div(a, big.NewInt(divider.Value))

	if !a.IsInt64() {
		return nil, errors.Errorf("%s long overflow %s", funcName, a.String())
	}

	return NewLong(a.Int64()), nil
}

//NativePowLong calculates power.
func NativePowLong(s Scope, e Exprs) (Expr, error) {
	funcName := "NativePowLong"
	if l := len(e); l != 6 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 6, received %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	base, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}

	bp, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}

	exponent, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s third argument expected to be *LongExpr, got %T", funcName, rs[2])
	}

	ep, ok := rs[3].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s 4th argument expected to be *LongExpr, got %T", funcName, rs[3])
	}

	rp, ok := rs[4].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s 5th argument expected to be *LongExpr, got %T", funcName, rs[4])
	}

	round, err := roundingMode(rs[5])
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	r, err := pow(base.Value, exponent.Value, int(bp.Value), int(ep.Value), int(rp.Value), round)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(r), nil
}

// NativeLogLong calculates logarithm.
func NativeLogLong(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeLogLong"
	if l := len(e); l != 6 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 6, received %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	base, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}

	bp, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}

	exponent, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s third argument expected to be *LongExpr, got %T", funcName, rs[2])
	}

	ep, ok := rs[3].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s 4th argument expected to be *LongExpr, got %T", funcName, rs[3])
	}

	rp, ok := rs[4].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s 5th argument expected to be *LongExpr, got %T", funcName, rs[4])
	}

	round, err := roundingMode(rs[5])
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	r, err := log(base.Value, exponent.Value, int(bp.Value), int(ep.Value), int(rp.Value), round)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(r), nil
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

func roundingMode(e Expr) (decimal.RoundingMode, error) {
	switch e.InstanceOf() {
	case "Ceiling":
		return decimal.ToPositiveInf, nil
	case "Floor":
		return decimal.ToNegativeInf, nil
	case "HalfEven":
		return decimal.ToNearestEven, nil
	case "Down":
		return decimal.ToZero, nil
	case "Up":
		return decimal.AwayFromZero, nil
	case "HalfUp":
		return decimal.ToNearestAway, nil
	case "HalfDown":
		// TODO: Enable this branch after PR https://github.com/ericlagergren/decimal/pull/136 is accepted. Before that this using this rounding mode will panic.
		// TODO: return decimal.ToNearestToZero, nil
		panic("not implemented rounding mode")
	default:
		return 0, errors.Errorf("unsupported rounding mode %s", e.InstanceOf())
	}
}

// Check signature
// accepts Value, signature and public key
func NativeSigVerify(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeSigVerify"

	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bytesExpr, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expects to be *BytesExpr, found %T", funcName, rs[0])
	}

	signatureExpr, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expects to be *BytesExpr, found %T", funcName, rs[1])
	}

	pkExpr, ok := rs[2].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expects to be *BytesExpr, found %T", funcName, rs[2])
	}

	pk, err := crypto.NewPublicKeyFromBytes(pkExpr.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	signature, err := crypto.NewSignatureFromBytes(signatureExpr.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	out := crypto.Verify(pk, signature, bytesExpr.Value)
	return NewBoolean(out), nil
}

// 256 bit Keccak256
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

	d, err := crypto.Keccak256(bts.Value)
	if err != nil {
		return nil, err
	}
	return NewBytes(d.Bytes()), nil
}

// 256 bit BLAKE
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

	d, err := crypto.FastHash(bts.Value)
	if err != nil {
		return nil, errors.Wrap(err, "NativeBlake2b256")
	}
	return NewBytes(d.Bytes()), nil
}

// 256 bit SHA-2
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
	if _, err = h.Write(bts.Value); err != nil {
		return nil, err
	}
	d := h.Sum(nil)

	return NewBytes(d), nil
}

// Рeight when transaction was stored to blockchain
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

	height, err := s.State().TransactionHeightByID(bts.Value)
	if err != nil {
		if err == mockstate.ErrNotFound {
			return Unit{}, nil
		}
		return nil, errors.Wrap(err, funcName)
	}

	return NewLong(int64(height)), nil
}

// Lookup transaction
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

	tx, err := s.State().TransactionByID(bts.Value)
	if err != nil {
		if err == mockstate.ErrNotFound {
			return Unit{}, nil
		}
		return nil, errors.Wrap(err, funcName)
	}

	vars, err := NewVariablesFromTransaction(s.Scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewObject(vars), nil
}

// Size of bytes vector
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

	return NewLong(int64(len(bts.Value))), nil
}

// Take firsts bytes
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

	if l >= len(bts.Value) {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	if l < 0 {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	out := make([]byte, l)
	copy(out, bts.Value[:l])

	return NewBytes(out), nil
}

// Skip firsts bytes
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

	if l >= len(bts.Value) {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	if l < 0 {
		return nil, errors.Errorf("%s index %d out of range", funcName, length.Value)
	}

	out := make([]byte, len(bts.Value)-l)
	copy(out, bts.Value[l:])

	return NewBytes(out), nil
}

// Limited bytes concatenation
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

	l := len(prefix.Value) + len(suffix.Value)

	if l > MaxBytesResult {
		return nil, errors.Errorf("%s byte length %d is greater than max %d", funcName, l, MaxBytesResult)
	}

	out := make([]byte, l)
	out = append(out[:0], prefix.Value...)
	out = append(out[:len(prefix.Value)], suffix.Value...)

	return NewBytes(out), nil
}

// Limited strings concatenation
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

// Take string prefix
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

	if l < 0 {
		return nil, errors.Errorf("%s index %d out of range", funcName, l)
	}

	out := make([]rune, l)
	copy(out, runeStr[:l])

	return NewString(string(out)), nil
}

// Remove string prefix
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

	if l < 0 {
		return nil, errors.Errorf("%s index %d out of range", funcName, l)
	}

	out := make([]rune, runeLen-l)
	copy(out, runeStr[l:])

	return NewString(string(out)), nil
}

// String size in characters
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

// Size of list
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

// Long to big endian bytes
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

// String to bytes representation
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

// Boolean to bytes representation (1 - true, 0 - false)
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

// Asset balance for account
func NativeAssetBalance(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeAssetBalance"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	addressOrAliasExpr, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	assetId, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	var r proto.Recipient

	switch a := addressOrAliasExpr.(type) {
	case AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(a))
	case AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(a))
	default:
		return nil, errors.Errorf("%s first argument expected to be AddressExpr or AliasExpr, found %T", funcName, addressOrAliasExpr)
	}

	if _, ok := assetId.(Unit); ok {
		return NewLong(int64(s.State().Account(r).AssetBalance(&proto.OptionalAsset{}))), nil
	}

	assetBts, ok := assetId.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *BytesExpr, found %T", funcName, assetId)
	}

	asset, err := proto.NewOptionalAssetFromBytes(assetBts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewLong(int64(s.State().Account(r).AssetBalance(asset))), nil
}

// Fail script
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

// String representation
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

// String representation
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

// Base58 encode
func NativeToBase58(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeToBase58"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	b, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", funcName, first)
	}

	return NewString(base58.Encode(b.Value)), nil
}

// Base58 decode
func NativeFromBase58(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeFromBase58"

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

	rs, err := base58.Decode(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewBytes(rs), nil
}

// Base64 decode
func NativeFromBase64(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeFromBase64"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, first)
	}

	decoded, err := base64.StdEncoding.DecodeString(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewBytes(decoded), nil
}

// Base64 encode
func NativeToBase64(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeToBase64"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bytes, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}

	encoded := base64.StdEncoding.EncodeToString(bytes.Value)
	return NewString(encoded), nil
}

// Base16 (Hex) decode
func NativeFromBase16(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeFromBase16"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, first)
	}

	decoded, err := hex.DecodeString(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewBytes(decoded), nil
}

// Base16 (Hex) encode
func NativeToBase16(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeToBase16"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bytes, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}

	encoded := hex.EncodeToString(bytes.Value)
	return NewString(encoded), nil
}

// Get integer from data of DataTransaction
func NativeDataLongFromArray(s Scope, e Exprs) (Expr, error) {
	return dataFromArray("NativeDataLongFromArray", s, e, proto.DataInteger)
}

// Get boolean from data of DataTransaction
func NativeDataBooleanFromArray(s Scope, e Exprs) (Expr, error) {
	return dataFromArray("NativeDataBooleanFromArray", s, e, proto.DataBoolean)
}

// Get string from data of DataTransaction
func NativeDataStringFromArray(s Scope, e Exprs) (Expr, error) {
	return dataFromArray("NativeDataBooleanFromArray", s, e, proto.DataString)
}

// Get bytes from data of DataTransaction
func NativeDataBinaryFromArray(s Scope, e Exprs) (Expr, error) {
	return dataFromArray("NativeDataBooleanFromArray", s, e, proto.DataBinary)
}

func dataFromArray(funcName string, s Scope, e Exprs, valueType proto.DataValueType) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	lstExpr, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	keyExpr, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	lst, ok := lstExpr.(*DataEntryListExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *DataEntryListExpr, found %T", funcName, lstExpr)
	}

	key, ok := keyExpr.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *StringExpr, found %T", funcName, keyExpr)
	}

	return lst.Get(key.Value, valueType), nil
}

// Get integer from account state
func NativeDataLongFromState(s Scope, e Exprs) (Expr, error) {
	return dataFromState("NativeDataLongFromState", s, e, proto.DataInteger)
}

// Get bool from account state
func NativeDataBooleanFromState(s Scope, e Exprs) (Expr, error) {
	return dataFromState("NativeDataBooleanFromState", s, e, proto.DataBoolean)
}

// Get bytes from account state
func NativeDataBytesFromState(s Scope, e Exprs) (Expr, error) {
	return dataFromState("NativeDataBytesFromState", s, e, proto.DataBinary)
}

// Get string from account state
func NativeDataStringFromState(s Scope, e Exprs) (Expr, error) {
	return dataFromState("NativeDataStringFromState", s, e, proto.DataString)
}

func dataFromState(funcName string, s Scope, e Exprs, valueType proto.DataValueType) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	addOrAliasExpr, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, err
	}

	if alias, ok := addOrAliasExpr.(AliasExpr); ok {
		r := proto.NewRecipientFromAlias(proto.Alias(alias))
		acc := s.State().Account(r)
		return dataFromArray(funcName, s.Clone(), Params(NewDataEntryList(acc.Data()), e[1]), valueType)
	}

	if addr, ok := addOrAliasExpr.(AddressExpr); ok {
		r := proto.NewRecipientFromAddress(proto.Address(addr))
		acc := s.State().Account(r)
		return dataFromArray(funcName, s.Clone(), Params(NewDataEntryList(acc.Data()), e[1]), valueType)
	}

	return nil, errors.Errorf("%s expected addOrAliasExpr argument to be AliasExpr or AddressExpr, found %T", funcName, addOrAliasExpr)
}

func NativeAddressFromRecipient(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeAddressFromRecipient"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	recipient, ok := first.(RecipientExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be RecipientExpr, found %T", funcName, recipient)
	}

	return NewAddressFromProtoAddress(s.State().Account(proto.Recipient(recipient)).Address()), nil
}

// Fail script without message (default will be used)
func UserThrow(_ Scope, _ Exprs) (Expr, error) {
	return nil, Throw{Message: DefaultThrowMessage}
}

// Decode account address
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

func NativeAddressToString(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAddressToString"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}

	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	addr, ok := rs.(AddressExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *AddressExpr, found %T", funcName, rs)
	}
	str := proto.Address(addr).String()
	return NewString(str), nil
}

// !=
func UserFunctionNeq(s Scope, e Exprs) (Expr, error) {
	funcName := "UserFunctionNeq"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	eq, err := rs[0].Eq(rs[1])
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewBoolean(!eq), nil
}

func UserIsDefined(s Scope, e Exprs) (Expr, error) {
	funcName := "UserIsDefined"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	val, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	if val.InstanceOf() == (Unit{}).InstanceOf() {
		return NewBoolean(false), nil
	}

	return NewBoolean(true), nil
}

func UserExtract(s Scope, e Exprs) (Expr, error) {
	funcName := "UserIsDefined"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	val, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	if val.InstanceOf() == (Unit{}).InstanceOf() {
		return NativeThrow(s.Clone(), Params(NewString("extract() called on unit value")))
	}

	return val, nil
}

func UserDropRightBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDropRightBytes"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	length, err := NativeSizeBytes(s.Clone(), Params(e[0]))
	if err != nil {
		return nil, err
	}

	takeLeft, err := NativeSubLong(s.Clone(), Params(length, e[1]))
	if err != nil {
		return nil, err
	}

	return NativeTakeBytes(s.Clone(), Params(e[0], takeLeft))
}

func UserTakeRightBytes(s Scope, e Exprs) (Expr, error) {
	funcName := "UserTakeRightBytes"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	length, err := NativeSizeBytes(s.Clone(), Params(e[0]))
	if err != nil {
		return nil, err
	}

	takeLeft, err := NativeSubLong(s.Clone(), Params(length, e[1]))
	if err != nil {
		return nil, err
	}

	return NativeDropBytes(s.Clone(), Params(e[0], takeLeft))
}

func UserTakeRightString(s Scope, e Exprs) (Expr, error) {
	funcName := "UserTakeRightString"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	length, err := NativeSizeString(s.Clone(), Params(e[0]))
	if err != nil {
		return nil, err
	}

	takeLeft, err := NativeSubLong(s.Clone(), Params(length, e[1]))
	if err != nil {
		return nil, err
	}

	return NativeDropStrings(s.Clone(), Params(e[0], takeLeft))
}

func UserDropRightString(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDropRightString"

	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	length, err := NativeSizeString(s.Clone(), Params(e[0]))
	if err != nil {
		return nil, err
	}

	takeLeft, err := NativeSubLong(s.Clone(), Params(length, e[1]))
	if err != nil {
		return nil, err
	}

	return NativeTakeStrings(s.Clone(), Params(e[0], takeLeft))
}

func UserUnaryMinus(s Scope, e Exprs) (Expr, error) {
	return NativeSubLong(s, append(Exprs{NewLong(0)}, e...))
}

func UserUnaryNot(s Scope, e Exprs) (Expr, error) {
	funcName := "UserUnaryNot"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	boolExpr, ok := first.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BooleanExpr, found %T", funcName, first)
	}

	return NewBoolean(!boolExpr.Value), nil
}

func dataFromArrayByIndex(funcName string, s Scope, e Exprs, valueType proto.DataValueType) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	lstExpr, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	indexExpr, err := e[1].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	lst, ok := lstExpr.(*DataEntryListExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *DataEntryListExpr, found %T", funcName, lstExpr)
	}

	key, ok := indexExpr.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *LongExpr, found %T", funcName, indexExpr)
	}

	return lst.GetByIndex(int(key.Value), valueType), nil
}

func UserDataIntegerFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDataIntegerFromArrayByIndex"
	return dataFromArrayByIndex(funcName, s, e, proto.DataInteger)
}

func UserDataBooleanFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDataBooleanFromArrayByIndex"
	return dataFromArrayByIndex(funcName, s, e, proto.DataBoolean)
}

func UserDataBinaryFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDataBinaryFromArrayByIndex"
	return dataFromArrayByIndex(funcName, s, e, proto.DataBinary)
}

func UserDataStringFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	funcName := "UserDataStringFromArrayByIndex"
	return dataFromArrayByIndex(funcName, s, e, proto.DataString)
}

func UserAddressFromPublicKey(s Scope, e Exprs) (Expr, error) {
	funcName := "UserAddressFromPublicKey"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}

	publicKeyExpr, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := publicKeyExpr.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, publicKeyExpr)
	}

	public, err := crypto.NewPublicKeyFromBytes(bts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	addr, err := proto.NewAddressFromPublicKey(s.Scheme(), public)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewAddressFromProtoAddress(addr), nil
}

// type constructor
func UserAddress(s Scope, e Exprs) (Expr, error) {
	funcName := "UserAddress"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	bts, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, first)
	}

	addr, err := proto.NewAddressFromBytes(bts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewAddressFromProtoAddress(addr), nil
}

func UserAlias(s Scope, e Exprs) (Expr, error) {
	funcName := "UserAlias"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, first)
	}

	alias, err := proto.NewAliasFromString(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewAliasFromProtoAlias(*alias), nil
}

func SimpleTypeConstructorFactory(name string, expr Expr) Callable {
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 0 {
			return nil, errors.Errorf("%s: no params expected, passed %d", name, l)
		}
		return expr, nil
	}
}

func UserWavesBalance(s Scope, e Exprs) (Expr, error) {
	return NativeAssetBalance(s, append(e, NewUnit()))
}

func NativeRSAVerify(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeRSAVerify"
	if l := len(e); l != 4 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 4, received %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	digest, err := digest(rs[0])
	if err != nil {
		return nil, errors.Wrapf(err, "%s: failed to get digest algorithm from first argument", funcName)
	}

	message, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *BytesExpr, found %T", funcName, rs[1])
	}

	sig, ok := rs[2].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expected to be *BytesExpr, found %T", funcName, rs[2])
	}

	pk, ok := rs[3].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: 4th argument expected to be *BytesExpr, found %T", funcName, rs[3])
	}

	if len(message.Value) > MaxBytesToVerify {
		return nil, errors.Errorf("%s: message is too long, must be no longer than %d bytes", funcName, MaxBytesToVerify)
	}

	key, err := x509.ParsePKIXPublicKey(pk.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: invalid public key", funcName)
	}
	k, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.Errorf("%s: not an RSA key", funcName)
	}

	d := message.Value
	if digest != 0 {
		h := digest.New()
		_, _ = h.Write(message.Value)
		d = h.Sum(nil)
	}

	ok, err = VerifyPKCS1v15(k, digest, d, sig.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: failed to check RSA signature", funcName)
	}

	return NewBoolean(ok), nil
}

func digest(e Expr) (Hash, error) {
	switch e.InstanceOf() {
	case "NoAlg":
		return 0, nil
	case "Md5":
		return MD5, nil
	case "Sha1":
		return SHA1, nil
	case "Sha224":
		return SHA224, nil
	case "Sha256":
		return SHA256, nil
	case "Sha384":
		return SHA384, nil
	case "Sha512":
		return SHA512, nil
	case "Sha3224":
		return SHA3_224, nil
	case "Sha3256":
		return SHA3_256, nil
	case "Sha3384":
		return SHA3_384, nil
	case "Sha3512":
		return SHA3_512, nil
	default:
		return 0, errors.Errorf("unsupported digest %s", e.InstanceOf())
	}
}

func NativeCheckMerkleProof(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeMerkleVerify"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	root, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Wrapf(err, "%s: first argument expected to be *BytesExpr, found %T", funcName, rs[0])
	}

	proof, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *BytesExpr, found %T", funcName, rs[1])
	}

	leaf, ok := rs[2].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expected to be *BytesExpr, found %T", funcName, rs[2])
	}

	r, err := merkleRootHash(leaf.Value, proof.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewBoolean(bytes.Equal(root.Value, r)), nil
}

func prefix(w io.Writer, name string, e Exprs) {
	_, _ = fmt.Fprintf(w, "%s(", name)
	last := len(e) - 1
	for i := 0; i < len(e); i++ {
		e[i].Write(w)
		if last != i {
			_, _ = fmt.Fprint(w, ", ")
		}
	}
	_, _ = fmt.Fprint(w, ")")
}

func infix(w io.Writer, name string, e Exprs) {
	e[0].Write(w)
	_, _ = fmt.Fprintf(w, " %s ", name)
	e[1].Write(w)
}

func writeNativeFunction(w io.Writer, id int16, e Exprs) {

	switch id {
	case 0:
		infix(w, "==", e)
	case 1:
		prefix(w, "_isInstanceOf", e)
	case 2:
		prefix(w, "throw", e)
	case 103:
		infix(w, ">=", e)
	case 108:
		prefix(w, "pow", e)
	case 109:
		prefix(w, "log", e)
	case 200:
		prefix(w, "size", e)
	case 203, 300:
		infix(w, "+", e)
	case 305:
		prefix(w, "size", e)
	case 401:
		e[0].Write(w)
		_, _ = fmt.Fprint(w, "[")
		e[1].Write(w)
		_, _ = fmt.Fprint(w, "]")
	case 410, 411, 412:
		prefix(w, "toBytes", e)
	case 420, 421:
		prefix(w, "toString", e)
	case 500:
		prefix(w, "sigVerify", e)
	case 501:
		prefix(w, "keccak256", e)
	case 502:
		prefix(w, "blake2b256", e)
	case 503:
		prefix(w, "sha256", e)
	case 600:
		prefix(w, "toBase58String", e)
	case 601:
		prefix(w, "fromBase58String", e)
	case 1000:
		prefix(w, "transactionById", e)
	case 1001:
		prefix(w, "transactionHeightById", e)
	case 1003:
		prefix(w, "assetBalance", e)
	case 1060:
		prefix(w, "addressFromRecipient", e)
	default:
		prefix(w, fmt.Sprintf("FUNCTION_%d(", id), e)
	}
}
