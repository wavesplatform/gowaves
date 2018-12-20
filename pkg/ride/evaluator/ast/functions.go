package ast

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
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
func NativeIsInstanceof(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeIsInstanceof"

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

// Integer devision
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

	d := crypto.Keccak256(bts.Value)
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
	h.Write(bts.Value)
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
		if err == state.ErrNotFound {
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
		if err == state.ErrNotFound {
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

	r := proto.Recipient{}

	switch a := addressOrAliasExpr.(type) {
	case AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(a))
	case AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(a))
	default:
		return nil, errors.Errorf("%s first argument expected to be AddressExpr or AliasExpr, found %T", funcName, addressOrAliasExpr)
	}

	if _, ok := assetId.(Unit); ok {
		return NewLong(int64(s.State().AssetBalance(r, &proto.OptionalAsset{}))), nil
	}

	assetBts, ok := assetId.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected second argument to be *BytesExpr, found %T", funcName, assetId)
	}

	asset, err := proto.NewOptionalAssetFromBytes(assetBts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return NewLong(int64(s.State().AssetBalance(r, asset))), nil
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
func NativeFromBase64String(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeFromBase64String"

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
func NativeToBse64String(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeToBse64String"

	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}

	first, err := e[0].Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	str, ok := first.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s expected first argument to be *BytesExpr, found %T", funcName, first)
	}

	encoded := base64.StdEncoding.EncodeToString(str.Value)
	return NewString(encoded), nil
}

// Get integer from data of DataTransaction
func NativeDataLongFromArray(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDataLongFromArray"
	return dataFromArray(funcName, s, e, proto.Integer)
}

// Get boolean from data of DataTransaction
func NativeDataBooleanFromArray(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDataBooleanFromArray"
	return dataFromArray(funcName, s, e, proto.Boolean)
}

// Get string from data of DataTransaction
func NativeDataStringFromArray(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDataBooleanFromArray"
	return dataFromArray(funcName, s, e, proto.String)
}

// Get bytes from data of DataTransaction
func NativeDataBinaryFromArray(s Scope, e Exprs) (Expr, error) {
	funcName := "NativeDataBooleanFromArray"
	return dataFromArray(funcName, s, e, proto.Binary)
}

func dataFromArray(funcName string, s Scope, e Exprs, valueType proto.ValueType) (Expr, error) {
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

func classic(w io.Writer, name string, e Exprs) {
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
		classic(w, "_isInstanceOf", e)
	case 2:
		classic(w, "throw", e)
	case 103:
		infix(w, ">=", e)
	case 200:
		classic(w, "size", e)
	case 203, 300:
		infix(w, "+", e)
	case 305:
		classic(w, "size", e)
	case 401:
		e[0].Write(w)
		_, _ = fmt.Fprint(w, "[")
		e[1].Write(w)
		_, _ = fmt.Fprint(w, "]")
	case 410, 411, 412:
		classic(w, "toBytes", e)
	case 420, 421:
		classic(w, "toString", e)
	case 500:
		classic(w, "sigVerify", e)
	case 501:
		classic(w, "keccak256", e)
	case 502:
		classic(w, "blake2b256", e)
	case 503:
		classic(w, "sha256", e)
	case 600:
		classic(w, "toBase58String", e)
	case 601:
		classic(w, "fromBase58String", e)
	case 1000:
		classic(w, "transactionById", e)
	case 1001:
		classic(w, "transactionHeightById", e)
	case 1003:
		classic(w, "assetBalance", e)
	default:
		classic(w, fmt.Sprintf("FUNCTION_%d(", id), e)
	}
}
