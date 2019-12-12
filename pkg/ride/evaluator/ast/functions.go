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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

func Params(params ...Expr) Exprs {
	return NewExprs(params...)
}

func DataEntries(params ...*DataEntryExpr) []*DataEntryExpr {
	return params
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
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, "NativeEq evaluate first param")
	}
	second, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, "NativeEq evaluate second param")
	}
	b := first.Eq(second)
	return NewBoolean(b), err
}

// Get list element by position
func NativeGetList(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeGetList"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, err
	}
	second, err := e[1].Evaluate(s)
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

func NativeCreateList(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeCreateList"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid parameters, expected 2, received %d", funcName, l)
	}
	head, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	t, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	tail, ok := t.(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: invalid second parameter, expected Exprs, received %T", funcName, e[1])
	}
	if len(tail) == 0 {
		return NewExprs(head), nil
	}
	return append(NewExprs(head), tail...), nil
}

// Internal function to check value type
func NativeIsInstanceOf(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeIsInstanceOf"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, err
	}
	second, err := e[1].Evaluate(s)
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
	return mathLong("NativeDivLong", func(x int64, y int64) (Expr, error) {
		if y == 0 {
			return nil, errors.New("zero division")
		}
		return NewLong(floorDiv(x, y)), nil
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
	const funcName = "NativeFractionLong"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	const funcName = "NativePowLong"
	if l := len(e); l != 6 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 6, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	const funcName = "NativeLogLong"
	if l := len(e); l != 6 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 6, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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

// Check signature
// accepts Value, signature and public key
func NativeSigVerify(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeSigVerify"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid params, expected 3, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bytesExpr, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expects to be *BytesExpr, found %T", funcName, rs[0])
	}
	if l := len(bytesExpr.Value); !s.validMessageLength(l) {
		return nil, errors.Errorf("%s: invalid message size %d", funcName, l)
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
		return NewBoolean(false), nil
	}
	signature, err := crypto.NewSignatureFromBytes(signatureExpr.Value)
	if err != nil {
		return NewBoolean(false), nil
	}
	out := crypto.Verify(pk, signature, bytesExpr.Value)
	return NewBoolean(out), nil
}

// 256 bit Keccak256
func NativeKeccak256(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 1 {
		return nil, errors.Errorf("NativeKeccak256: invalid params, expected 1, passed %d", l)
	}
	val, err := e[0].Evaluate(s)
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
	val, err := e[0].Evaluate(s)
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
	val, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrapf(err, "NativeSha256")
	}
	var bytes []byte
	switch s := val.(type) {
	case *BytesExpr:
		bytes = s.Value
	case *StringExpr:
		bytes = []byte(s.Value)
	default:
		return nil, errors.Errorf("NativeSha256: expected first argument to be *BytesExpr or *StringExpr, found %T", val)
	}
	h := sha256.New()
	if _, err = h.Write(bytes); err != nil {
		return nil, err
	}
	d := h.Sum(nil)
	return NewBytes(d), nil
}

// Height when transaction was stored to blockchain
func NativeTransactionHeightByID(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeTransactionHeightByID"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, got %T", funcName, rs)
	}
	height, err := s.State().NewestTransactionHeightByID(bts.Value)
	if err != nil {
		if s.State().IsNotFound(err) {
			return &Unit{}, nil
		}
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(int64(height)), nil
}

// Lookup transaction
func NativeTransactionByID(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeTransactionByID"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, got %T", funcName, rs)
	}
	tx, err := s.State().NewestTransactionByID(bts.Value)
	if err != nil {
		if s.State().IsNotFound(err) {
			return NewUnit(), nil
		}
		return nil, errors.Wrap(err, funcName)
	}
	vars, err := NewVariablesFromTransaction(s.Scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewObject(vars), nil
}

//1006: returns Union[TransferTransaction, Unit]
func NativeTransferTransactionByID(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeTransferTransactionByID"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, got %T", funcName, rs)
	}
	tx, err := s.State().NewestTransactionByID(bts.Value)
	if err != nil {
		if s.State().IsNotFound(err) {
			return NewUnit(), nil
		}
		return nil, errors.Wrap(err, funcName)
	}

	switch t := tx.(type) {
	case *proto.TransferV2:
		rs, err := newVariablesFromTransferV2(s.Scheme(), t)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewObject(rs), nil
	case *proto.TransferV1:
		rs, err := newVariablesFromTransferV1(s.Scheme(), t)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewObject(rs), nil
	default:
		return NewUnit(), nil
	}
}

func NativeParseBlockHeader(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeParseBlockHeader"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := rs.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, rs)
	}

	h := proto.BlockHeader{}
	err = h.UnmarshalHeaderFromBinary(bts.Value)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	obj, err := newMapFromBlockHeader(s.Scheme(), &h)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	return NewBlockHeader(obj), nil
}

// Size of bytes vector
func NativeSizeBytes(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeSizeBytes"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
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
	const funcName = "NativeTakeBytes"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	if l > len(bts.Value) {
		l = len(bts.Value)
	}
	if l < 0 {
		l = 0
	}
	out := make([]byte, l)
	copy(out, bts.Value[:l])
	return NewBytes(out), nil
}

// Skip firsts bytes
func NativeDropBytes(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeDropBytes"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	if l > len(bts.Value) {
		l = len(bts.Value)
	}
	if l < 0 {
		l = 0
	}
	out := make([]byte, len(bts.Value)-l)
	copy(out, bts.Value[l:])
	return NewBytes(out), nil
}

// Limited bytes concatenation
func NativeConcatBytes(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeConcatBytes"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	const funcName = "NativeConcatStrings"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	const funcName = "NativeTakeStrings"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	if l > runeLen {
		l = runeLen
	}
	if l < 0 {
		l = 0
	}
	out := make([]rune, l)
	copy(out, runeStr[:l])
	return NewString(string(out)), nil
}

// Remove string prefix
func NativeDropStrings(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeDropStrings"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	if l > runeLen {
		l = runeLen
	}
	if l < 0 {
		l = 0
	}
	out := make([]rune, runeLen-l)
	copy(out, runeStr[l:])
	return NewString(string(out)), nil
}

// String size in characters
func NativeSizeString(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeSizeBytes"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
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
	const funcName = "NativeSizeList"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	// optimize not evaluate inner list
	if v, ok := e[0].(Exprs); ok {
		return NewLong(int64(len(v))), nil
	}
	rs, err := e[0].Evaluate(s)
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
	const funcName = "NativeLongToBytes"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeStringToBytes"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeBooleanToBytes"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := rs.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *BooleanExpr, got %T", funcName, rs)
	}
	if b.Value {
		return NewBytes([]byte{1}), nil
	} else {
		return NewBytes([]byte{0}), nil
	}
}

// Asset balance for account
func NativeAssetBalance(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetBalance"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	addressOrAliasExpr, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	assetId, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	var r proto.Recipient
	switch a := addressOrAliasExpr.(type) {
	case *AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(*a))
	case *AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(*a))
	case *RecipientExpr:
		r = proto.Recipient(*a)
	default:
		return nil, errors.Errorf("%s first argument expected to be AddressExpr or AliasExpr, found %T", funcName, addressOrAliasExpr)
	}
	if _, ok := assetId.(*Unit); ok {
		balance, err := s.State().NewestAccountBalance(r, nil)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewLong(int64(balance)), nil
	}
	assetBts, ok := assetId.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *BytesExpr, found %T", funcName, assetId)
	}
	balance, err := s.State().NewestAccountBalance(r, assetBts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(int64(balance)), nil
}

// Fail script
func NativeThrow(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeThrow"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeLongToString"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeBooleanToString"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeToBase58"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	switch arg := first.(type) {
	case *BytesExpr:
		return NewString(base58.Encode(arg.Value)), nil
	case *Unit:
		return NewString(base58.Encode(nil)), nil
	default:
		return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", funcName, first)
	}
}

// Base58 decode
func NativeFromBase58(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeFromBase58"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be *StringExpr, found %T", funcName, first)
	}
	if str.Value == "" {
		return NewBytes(nil), nil
	}
	rs, err := base58.Decode(str.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewBytes(rs), nil
}

// Base64 decode
func NativeFromBase64(s Scope, e Exprs) (Expr, error) {
	const prefix = "base64:"
	const funcName = "NativeFromBase64"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, first)
	}
	ev := strings.TrimPrefix(str.Value, prefix)
	decoded, err := base64.StdEncoding.DecodeString(ev)
	if err != nil {
		// Try no padding.
		decoded, err = base64.RawStdEncoding.DecodeString(ev)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewBytes(decoded), nil
	}
	return NewBytes(decoded), nil
}

// Base64 encode
func NativeToBase64(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeToBase64"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}
	encoded := base64.StdEncoding.EncodeToString(b.Value)
	return NewString(encoded), nil
}

// Base16 (Hex) decode
func NativeFromBase16(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeFromBase16"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
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
	const funcName = "NativeToBase16"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}
	encoded := hex.EncodeToString(b.Value)
	return NewString(encoded), nil
}

// Get integer from data of DataTransaction
func NativeDataIntegerFromArray(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArray(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataIntegerFromArray")
	}
	_, ok := d.(*LongExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

// Get boolean from data of DataTransaction
func NativeDataBooleanFromArray(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArray(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataBooleanFromArray")
	}
	_, ok := d.(*BooleanExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

// Get bytes from data of DataTransaction
func NativeDataBinaryFromArray(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArray(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataBinaryFromArray")
	}
	_, ok := d.(*BytesExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

// Get string from data of DataTransaction
func NativeDataStringFromArray(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArray(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataStringFromArray")
	}
	_, ok := d.(*StringExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

// Get integer from account state
func NativeDataIntegerFromState(s Scope, e Exprs) (Expr, error) {
	r, k, err := extractRecipientAndKey(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataIntegerFromState")
	}
	entry, err := s.State().RetrieveNewestIntegerEntry(r, k)
	if err != nil {
		return NewUnit(), nil
	}
	return NewLong(entry.Value), nil
}

// Get bool from account state
func NativeDataBooleanFromState(s Scope, e Exprs) (Expr, error) {
	r, k, err := extractRecipientAndKey(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataBooleanFromState")
	}
	entry, err := s.State().RetrieveNewestBooleanEntry(r, k)
	if err != nil {
		return NewUnit(), nil
	}
	return NewBoolean(entry.Value), nil
}

// Get bytes from account state
func NativeDataBinaryFromState(s Scope, e Exprs) (Expr, error) {
	r, k, err := extractRecipientAndKey(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataBinaryFromState")
	}
	entry, err := s.State().RetrieveNewestBinaryEntry(r, k)
	if err != nil {
		return NewUnit(), nil
	}
	return NewBytes(entry.Value), nil
}

// Get string from account state
func NativeDataStringFromState(s Scope, e Exprs) (Expr, error) {
	r, k, err := extractRecipientAndKey(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "NativeDataStringFromState")
	}
	entry, err := s.State().RetrieveNewestStringEntry(r, k)
	if err != nil {
		return NewUnit(), nil
	}
	return NewString(entry.Value), nil
}

func NativeAddressFromRecipient(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAddressFromRecipient"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	recipient, ok := first.(*RecipientExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be RecipientExpr, found %T", funcName, first)
	}

	if recipient.Address != nil {
		return NewAddressFromProtoAddress(*recipient.Address), nil
	}

	if recipient.Alias != nil {
		addr, err := s.State().NewestAddrByAlias(*recipient.Alias)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewAddressFromProtoAddress(addr), nil
	}

	return nil, errors.Errorf("can't get address from recipient, recipient %v", recipient)
}

// 1004: accepts id: []byte
func NativeAssetInfo(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetInfo"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	id, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}
	assetId, err := crypto.NewDigestFromBytes(id.Value)
	if err != nil {
		return NewUnit(), nil // Return Unit not an error on invalid Asset IDs
	}
	info, err := s.State().NewestAssetInfo(assetId)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewAssetInfo(newMapAssetInfo(*info)), nil
}

//1005:
func NativeBlockInfoByHeight(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeBlockInfoByHeight"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	height, ok := first.(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *LongExpr, found %T", funcName, first)
	}

	h, err := s.State().NewestHeaderByHeight(proto.Height(height.Value))
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	obj, err := newMapFromBlockHeader(s.Scheme(), h)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewBlockInfo(obj, proto.Height(height.Value)), nil
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
		return NewUnit(), nil
	}
	if addr[1] != s.Scheme() {
		return NewUnit(), nil
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
	addr, ok := rs.(*AddressExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *AddressExpr, found %T", funcName, rs)
	}
	str := proto.Address(*addr).String()
	return NewString(str), nil
}

// !=
func UserFunctionNeq(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserFunctionNeq"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	eq := rs[0].Eq(rs[1])
	return NewBoolean(!eq), nil
}

func UserIsDefined(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserIsDefined"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	val, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	if val.InstanceOf() == (&Unit{}).InstanceOf() {
		return NewBoolean(false), nil
	}
	return NewBoolean(true), nil
}

func UserExtract(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserExtract"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	val, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	if val.InstanceOf() == (&Unit{}).InstanceOf() {
		return NativeThrow(s, Params(NewString("extract() called on unit value")))
	}
	return val, nil
}

func UserDropRightBytes(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserDropRightBytes"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	length, err := NativeSizeBytes(s, Params(e[0]))
	if err != nil {
		return nil, err
	}
	takeLeft, err := NativeSubLong(s, Params(length, e[1]))
	if err != nil {
		return nil, err
	}
	return NativeTakeBytes(s, Params(e[0], takeLeft))
}

func UserTakeRightBytes(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserTakeRightBytes"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	length, err := NativeSizeBytes(s, Params(e[0]))
	if err != nil {
		return nil, err
	}
	takeLeft, err := NativeSubLong(s, Params(length, e[1]))
	if err != nil {
		return nil, err
	}
	return NativeDropBytes(s, Params(e[0], takeLeft))
}

func UserTakeRightString(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserTakeRightString"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	length, err := NativeSizeString(s, Params(e[0]))
	if err != nil {
		return nil, err
	}
	takeLeft, err := NativeSubLong(s, Params(length, e[1]))
	if err != nil {
		return nil, err
	}
	return NativeDropStrings(s, Params(e[0], takeLeft))
}

func UserDropRightString(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserDropRightString"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	length, err := NativeSizeString(s, Params(e[0]))
	if err != nil {
		return nil, err
	}
	takeLeft, err := NativeSubLong(s, Params(length, e[1]))
	if err != nil {
		return nil, err
	}
	return NativeTakeStrings(s, Params(e[0], takeLeft))
}

func UserUnaryMinus(s Scope, e Exprs) (Expr, error) {
	return NativeSubLong(s, append(Exprs{NewLong(0)}, e...))
}

func UserUnaryNot(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserUnaryNot"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	boolExpr, ok := first.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BooleanExpr, found %T", funcName, first)
	}
	return NewBoolean(!boolExpr.Value), nil
}

func UserDataIntegerFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArrayByIndex(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "UserDataIntegerFromArrayByIndex")
	}
	_, ok := d.(*LongExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

func UserDataBooleanFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArrayByIndex(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "UserDataBooleanFromArrayByIndex")
	}
	_, ok := d.(*BooleanExpr)
	if !ok {
		return NewUnit(), nil
	}
	if d.InstanceOf() == "DataEntry" {
		val, err := d.(*DataEntryExpr).Get("value")
		if err != nil {
			return nil, errors.Wrap(err, "UserDataBooleanFromArrayByIndex")
		}
		_, ok := val.(*BooleanExpr)
		if !ok {
			return NewUnit(), nil
		}
	}
	return d, nil
}

func UserDataBinaryFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArrayByIndex(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "UserDataBinaryFromArrayByIndex")
	}
	_, ok := d.(*BytesExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

func UserDataStringFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	d, err := dataFromArrayByIndex(s, e)
	if err != nil {
		return nil, errors.Wrap(err, "UserDataStringFromArrayByIndex")
	}
	_, ok := d.(*StringExpr)
	if !ok {
		return NewUnit(), nil
	}
	return d, nil
}

func UserAddressFromPublicKey(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserAddressFromPublicKey"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	publicKeyExpr, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := publicKeyExpr.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, publicKeyExpr)
	}
	addr, err := proto.NewAddressLikeFromAnyBytes(s.Scheme(), bts.Value)
	if err != nil {
		return NewUnit(), nil
	}
	return NewAddressFromProtoAddress(addr), nil
}

// type constructor
func UserAddress(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserAddress"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bts, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, first)
	}
	addr, err := proto.NewAddressFromBytes(bts.Value)
	if err != nil {
		return &InvalidAddressExpr{Value: bts.Value}, nil
	}
	return NewAddressFromProtoAddress(addr), nil
}

func UserAlias(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserAlias"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := first.(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, first)
	}
	alias := proto.NewAlias(s.Scheme(), str.Value)
	return NewAliasFromProtoAlias(*alias), nil
}

func DataEntry(s Scope, e Exprs) (Expr, error) {
	const funcName = "DataEntry"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	key, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	switch t := rs[1].(type) {
	case *LongExpr, *BooleanExpr, *BytesExpr, *StringExpr:
		return NewDataEntry(key.Value, t), nil
	default:
		return nil, errors.Errorf("%s: unsupported value type %T", funcName, t)
	}
}

func DataTransaction(s Scope, e Exprs) (Expr, error) {
	const funcName = "DataTransaction"
	if l := len(e); l != 9 {
		return nil, errors.Errorf("%s: invalid params, expected 9, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	out := make(map[string]Expr)

	entries, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be List, found %T", funcName, rs[0])
	}
	out["data"] = entries

	id, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *BytesExpr, found %T", funcName, rs[1])
	}
	out["id"] = id

	fee, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expected to be *LongExpr, found %T", funcName, rs[2])
	}
	out["fee"] = fee

	timestamp, ok := rs[3].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: 4th argument expected to be *LongExpr, found %T", funcName, rs[3])
	}
	out["timestamp"] = timestamp

	version, ok := rs[4].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: 5th argument expected to be *LongExpr, found %T", funcName, rs[4])
	}
	out["version"] = version

	addr, ok := rs[5].(*AddressExpr)
	if !ok {
		return nil, errors.Errorf("%s: 6th argument expected to be *AddressExpr, found %T", funcName, rs[5])
	}
	out["sender"] = addr

	pk, ok := rs[6].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: 7th argument expected to be *BytesExpr, found %T", funcName, rs[6])
	}
	out["senderPublicKey"] = pk

	body, ok := rs[7].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: 8th argument expected to be *BytesExpr, found %T", funcName, rs[7])
	}
	out["bodyBytes"] = body

	proofs, ok := rs[8].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: 9th argument expected to be List, found %T", funcName, rs[8])
	}
	out["proofs"] = proofs
	out[InstanceFieldName] = NewString("DataTransaction")

	return NewObject(out), nil
}

func AssetPair(s Scope, e Exprs) (Expr, error) {
	const funcName = "AssetPair"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewAssetPair(rs[0], rs[1]), nil
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
	const funcName = "NativeRSAVerify"
	if l := len(e); l != 4 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 4, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
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
	ok, err = verifyPKCS1v15(k, digest, d, sig.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: failed to check RSA signature", funcName)
	}
	return NewBoolean(ok), nil
}

func NativeCheckMerkleProof(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeMerkleVerify"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	root, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, rs[0])
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

func NativeBytesToUTF8String(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeBytesToUTF8String"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, rs[0])
	}
	return NewString(string(b.Value)), nil
}

func NativeBytesToLong(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeBytesToLong"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, rs[0])
	}
	if l := len(b.Value); l < 8 {
		return nil, errors.Errorf("%s: %d is not enough bytes to make Long value, required 8 bytes", funcName, l)
	}
	return NewLong(int64(binary.BigEndian.Uint64(b.Value))), nil
}

func NativeBytesToLongWithOffset(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeBytesToLongWithOffset"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	b, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, rs[0])
	}
	off, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *LongExpr, found %T", funcName, rs[1])
	}
	offset := int(off.Value)
	if offset < 0 || offset > len(b.Value)-8 {
		return nil, errors.Errorf("%s: offset %d is out of bytes array bounds", funcName, offset)
	}
	return NewLong(int64(binary.BigEndian.Uint64(b.Value[offset:]))), nil
}

func NativeIndexOfSubstring(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeIndexOfSubstring"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	sub, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	i := strings.Index(str.Value, sub.Value)
	if i == -1 {
		return NewUnit(), nil
	}
	return NewLong(int64(i)), nil
}

func NativeIndexOfSubstringWithOffset(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeIndexOfSubstringWithOffset"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	sub, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	off, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expected to be *LongExpr, found %T", funcName, rs[2])
	}
	offset := int(off.Value)
	if offset < 0 || offset > len(str.Value) {
		return NewUnit(), nil
	}
	i := strings.Index(str.Value[offset:], sub.Value)
	if i == -1 {
		return NewUnit(), nil
	}
	return NewLong(int64(i + offset)), nil
}

func NativeSplitString(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeSplitString"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	sep, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	r := NewExprs()
	for _, p := range strings.Split(str.Value, sep.Value) {
		r = append(r, NewString(p))
	}
	return r, nil
}

func NativeParseInt(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeParseInt"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}

	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	i, err := strconv.ParseInt(str.Value, 10, 64)
	if err != nil {
		return NewUnit(), nil
	}
	return NewLong(i), nil
}

func NativeLastIndexOfSubstring(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeLastIndexOfSubstring"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	sub, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	i := strings.LastIndex(str.Value, sub.Value)
	if i == -1 {
		return NewUnit(), nil
	}
	return NewLong(int64(i)), nil
}

func NativeLastIndexOfSubstringWithOffset(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeLastIndexOfSubstringWithOffset"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	sub, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	off, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: third argument expected to be *LongExpr, found %T", funcName, rs[2])
	}
	offset := int(off.Value)
	if offset < 0 {
		return NewUnit(), nil
	}
	i := strings.LastIndex(str.Value, sub.Value)
	for i > offset {
		i = strings.LastIndex(str.Value[:i], sub.Value)
	}
	if i == -1 {
		return NewUnit(), nil
	}
	return NewLong(int64(i)), nil
}

func UserValue(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserValue"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	if _, ok := rs[0].(*Unit); ok {
		return nil, Throw{Message: DefaultThrowMessage}
	}
	return rs[0], nil
}

func UserValueOrErrorMessage(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserValueOrErrorMessage"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	msg, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: second argument expected to be *StringExpr, found %T", funcName, rs[1])
	}
	if _, ok := rs[0].(*Unit); ok {
		return nil, Throw{Message: msg.Value}
	}
	return rs[0], nil
}

func UserWriteSet(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserWriteSet"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	listOfDataEntries, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be Exprs, found %T", funcName, rs[0])
	}

	var dataEntries []*DataEntryExpr
	for _, expr := range listOfDataEntries {
		if expr.InstanceOf() != "DataEntry" {
			return nil, errors.Errorf("Expected instance of DataEntry, found %s, %T", expr.InstanceOf(), expr)
		}
		dataEntries = append(dataEntries, expr.(*DataEntryExpr))
	}
	return NewWriteSet(dataEntries...), nil
}

func UserTransferSet(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserTransferSet"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	listOfScriptTransfer, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be Exprs, found %T", funcName, rs[0])
	}
	var transfers []*ScriptTransferExpr
	for _, expr := range listOfScriptTransfer {
		if expr.InstanceOf() != "ScriptTransfer" {
			return nil, errors.Errorf("Expected instance of ScriptTransfer, found %s, %T", expr.InstanceOf(), expr)
		}
		transfers = append(transfers, expr.(*ScriptTransferExpr))
	}
	return NewTransferSet(transfers...), nil
}

func ScriptTransfer(s Scope, e Exprs) (Expr, error) {
	const funcName = "ScriptTransfer"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	recipient, ok := rs[0].(Recipient)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be 'RecipientExpr', got '%T'", funcName, rs[0])
	}
	amount, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected secnd argument to be '*LongExpr', got '%T'", funcName, rs[1])
	}
	return NewScriptTransfer(recipient, amount, rs[2])
}

func ScriptResult(s Scope, e Exprs) (Expr, error) {
	const funcName = "ScriptResult"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrapf(err, funcName)
	}
	writeSet, ok := rs[0].(*WriteSetExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be 'Exprs', got '%T'", funcName, rs[0])
	}
	transferSet, ok := rs[1].(*TransferSetExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected secnd argument to be 'Exprs', got '%T'", funcName, rs[1])
	}
	return NewScriptResult(writeSet, transferSet), nil
}

func wrapWithExtract(c Callable, name string) Callable {
	return func(s Scope, e Exprs) (Expr, error) {
		rs, err := c(s, e)
		if err != nil {
			return nil, errors.Wrap(err, name)
		}
		if _, ok := rs.(*Unit); ok {
			return nil, Throw{Message: "failed to extract from Unit value"}
		}
		return rs, err
	}
}

func dataFromArray(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("invalid params, expected 2, passed %d", l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, err
	}
	lst, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("expected first argument to be *Exprs, found %T", rs[0])
	}
	key, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("expected second argument to be *StringExpr, found %T", rs[1])
	}
	for i, e := range lst {
		item, ok := e.(Getable)
		if !ok {
			return nil, errors.Errorf("unexpected list element of type %T", e)
		}
		k, err := item.Get("key")
		if err != nil {
			return nil, errors.Wrapf(err, "%dth element doesn't have 'key' field", i)
		}
		b := key.Eq(k)
		if b {
			v, err := item.Get("value")
			if err != nil {
				return nil, errors.Wrapf(err, "%dth element doesn't have 'value' field", i)
			}
			return v, nil
		}
	}
	return NewUnit(), nil
}

func extractRecipientAndKey(s Scope, e Exprs) (proto.Recipient, string, error) {
	if l := len(e); l != 2 {
		return proto.Recipient{}, "", errors.Errorf("invalid params, expected 2, passed %d", l)
	}
	addOrAliasExpr, err := e[0].Evaluate(s)
	if err != nil {
		return proto.Recipient{}, "", err
	}
	var r proto.Recipient
	switch a := addOrAliasExpr.(type) {
	case *AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(*a))
	case *AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(*a))
	case *RecipientExpr:
		r = proto.Recipient(*a)
	default:
		return proto.Recipient{}, "", errors.Errorf("expected first argument of types AliasExpr of AddressExpr, found %T", addOrAliasExpr)
	}
	second, err := e[1].Evaluate(s)
	if err != nil {
		return proto.Recipient{}, "", err
	}
	key, ok := second.(*StringExpr)
	if !ok {
		return proto.Recipient{}, "", errors.Errorf("second argument expected to be *StringExpr, found %T", second)
	}
	return r, key.Value, nil
}

func dataFromArrayByIndex(s Scope, e Exprs) (Expr, error) {
	if l := len(e); l != 2 {
		return nil, errors.Errorf("invalid params, expected 2, passed %d", l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, err
	}
	lst, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("expected first argument to be *Exprs, found %T", rs[0])
	}
	index, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("expected second argument to be *LongExpr, found %T", rs[1])
	}
	i := int(index.Value)
	if i < 0 || i >= len(lst) {
		return nil, errors.Errorf("invalid index %d", i)
	}
	item, ok := lst[i].(Getable)
	if !ok {
		return nil, errors.Errorf("unexpected list element of type %T", e)
	}
	v, err := item.Get("value")
	if err != nil {
		return nil, errors.Wrapf(err, "%dth element doesn't have 'value' field", i)
	}
	return v, nil
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

func writeFunction(w io.Writer, id string, e Exprs) {
	switch id {
	case "0":
		infix(w, "==", e)
	case "1":
		prefix(w, "_isInstanceOf", e)
	case "2":
		prefix(w, "throw", e)
	case "103":
		infix(w, ">=", e)
	case "108":
		prefix(w, "pow", e)
	case "109":
		prefix(w, "log", e)
	case "200":
		prefix(w, "size", e)
	case "203", "300":
		infix(w, "+", e)
	case "305":
		prefix(w, "size", e)
	case "401":
		e[0].Write(w)
		_, _ = fmt.Fprint(w, "[")
		e[1].Write(w)
		_, _ = fmt.Fprint(w, "]")
	case "410", "411", "412":
		prefix(w, "toBytes", e)
	case "420", "421":
		prefix(w, "toString", e)
	case "500":
		prefix(w, "sigVerify", e)
	case "501":
		prefix(w, "keccak256", e)
	case "502":
		prefix(w, "blake2b256", e)
	case "503":
		prefix(w, "sha256", e)
	case "600":
		prefix(w, "toBase58String", e)
	case "601":
		prefix(w, "fromBase58String", e)
	case "1000":
		prefix(w, "transactionById", e)
	case "1001":
		prefix(w, "transactionHeightById", e)
	case "1003":
		prefix(w, "assetBalance", e)
	case "1060":
		prefix(w, "addressFromRecipient", e)
	default:
		prefix(w, fmt.Sprintf("FUNCTION_%s(", id), e)
	}
}
