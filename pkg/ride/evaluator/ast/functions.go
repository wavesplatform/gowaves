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
	"sort"
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
	MaxListSize         = 1000
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

func LimitedCreateList(s Scope, e Exprs) (Expr, error) {
	const funcName = "LimitedCreateList"
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
	if len(tail) == MaxListSize {
		return nil, errors.Errorf("%s: list size can not exceed %d elements", funcName, MaxListSize)
	}
	if len(tail) == 0 {
		return NewExprs(head), nil
	}
	return append(NewExprs(head), tail...), nil
}

func AppendToList(s Scope, e Exprs) (Expr, error) {
	const funcName = "AppendToList"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid parameters, expected 2, received %d", funcName, l)
	}
	l, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	list, ok := l.(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: invalid first parameter, expected Exprs, received %T", funcName, e[0])
	}
	if len(list) == MaxListSize {
		return nil, errors.Errorf("%s: list size can not exceed %d elements", funcName, MaxListSize)
	}
	element, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	if len(list) == 0 {
		return NewExprs(element), nil
	}
	return append(list, element), nil
}

func Concat(s Scope, e Exprs) (Expr, error) {
	const funcName = "Concat"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	list1, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: invalid first parameter, expected Exprs, received %T", funcName, rs[0])
	}
	list2, ok := rs[1].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: invalid second parameter, expected Exprs, received %T", funcName, rs[1])
	}
	if len(list1)+len(list2) > MaxListSize {
		return nil, errors.Errorf("%s: list size can not exceed %d elements", funcName, MaxListSize)
	}
	if len(list1) == 0 {
		list1 = NewExprs()
	}
	return append(list1, list2...), nil
}

func Median(s Scope, e Exprs) (Expr, error) {
	const funcName = "Median"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid parameters, expected 1, received %d", funcName, l)
	}
	l, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	list, ok := l.(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: invalid first parameter, expected Exprs, received %T", funcName, e[0])
	}
	size := len(list)
	if size > MaxListSize || size < 2 {
		return nil, errors.Errorf("%s: invalid list size %d", funcName, size)
	}
	items := make([]int, size)
	for i, el := range list {
		item, ok := el.(*LongExpr)
		if !ok {
			return nil, errors.Errorf("%s: list must contain only LongExpr elements", funcName)
		}
		items[i] = int(item.Value)
	}
	sort.Ints(items)
	half := size / 2
	if size%2 == 1 {
		return NewLong(int64(items[half])), nil
	} else {
		return NewLong(floorDiv(int64(items[half-1])+int64(items[half]), 2)), nil
	}
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
	value, ok := rs[0].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s first argument expected to be *LongExpr, got %T", funcName, rs[0])
	}
	numerator, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s second argument expected to be *LongExpr, got %T", funcName, rs[1])
	}
	denominator, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s third argument expected to be *LongExpr, got %T", funcName, rs[2])
	}
	res, err := fraction(value.Value, numerator.Value, denominator.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(res), nil
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

func limitedSigVerify(limit int) Callable {
	fn := "SigVerify"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 3 {
			return nil, errors.Errorf("%s: invalid number of parameters %d, expected 3", fn, l)
		}
		rs, err := e.EvaluateAll(s)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		messageExpr, ok := rs[0].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: first argument expects to be *BytesExpr, found %T", fn, rs[0])
		}
		if l := len(messageExpr.Value); !s.validMessageLength(l) || limit > 0 && l > limit*1024 {
			return nil, errors.Errorf("%s: invalid message size %d", fn, l)
		}
		signatureExpr, ok := rs[1].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: second argument expects to be *BytesExpr, found %T", fn, rs[1])
		}
		pkExpr, ok := rs[2].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: third argument expects to be *BytesExpr, found %T", fn, rs[2])
		}
		pk, err := crypto.NewPublicKeyFromBytes(pkExpr.Value)
		if err != nil {
			return NewBoolean(false), nil
		}
		signature, err := crypto.NewSignatureFromBytes(signatureExpr.Value)
		if err != nil {
			return NewBoolean(false), nil
		}
		out := crypto.Verify(pk, signature, messageExpr.Value)
		return NewBoolean(out), nil
	}
}

func limitedKeccak256(limit int) Callable {
	fn := "Keccak256"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 1 {
			return nil, errors.Errorf("%s: invalid number of parameters %d, expected 1", fn, l)
		}
		val, err := e[0].Evaluate(s)
		if err != nil {
			return nil, errors.Wrapf(err, fn)
		}
		dataExpr, ok := val.(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", fn, val)
		}
		if l := len(dataExpr.Value); limit > 0 && l > limit {
			return nil, errors.Errorf("%s: invalid size of data %d bytes", fn, l)
		}
		d, err := crypto.Keccak256(dataExpr.Value)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		return NewBytes(d.Bytes()), nil
	}
}

func limitedBlake2b256(limit int) Callable {
	fn := "Blake2b256"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 1 {
			return nil, errors.Errorf("%s: invalid number of parameters %d, expected 1", fn, l)
		}
		val, err := e[0].Evaluate(s)
		if err != nil {
			return nil, errors.Wrapf(err, fn)
		}
		dataExpr, ok := val.(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", fn, val)
		}
		if l := len(dataExpr.Value); limit > 0 && l > limit*1024 {
			return nil, errors.Errorf("%s: invalid data size %d bytes", fn, l)
		}
		d, err := crypto.FastHash(dataExpr.Value)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		return NewBytes(d.Bytes()), nil
	}
}

// 256 bit SHA-2
func limitedSha256(limit int) Callable {
	fn := "Sha256"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 1 {
			return nil, errors.Errorf("%s: invalid number of parameters %d, expected 1", fn, l)
		}
		val, err := e[0].Evaluate(s)
		if err != nil {
			return nil, errors.Wrapf(err, fn)
		}
		var bytes []byte
		switch s := val.(type) {
		case *BytesExpr:
			bytes = s.Value
		case *StringExpr:
			bytes = []byte(s.Value)
		default:
			return nil, errors.Errorf("%s: expected first argument to be *BytesExpr or *StringExpr, found %T", fn, val)
		}
		if l := len(bytes); limit > 0 && l > limit*1024 {
			return nil, errors.Errorf("%s: invalid data size %d bytes", fn, l)
		}
		h := sha256.New()
		if _, err = h.Write(bytes); err != nil {
			return nil, errors.Wrap(err, fn)
		}
		d := h.Sum(nil)
		return NewBytes(d), nil
	}
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
	case *proto.TransferWithProofs:
		rs, err := newVariablesFromTransferWithProofs(s.Scheme(), t)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewObject(rs), nil
	case *proto.TransferWithSig:
		rs, err := newVariablesFromTransferWithSig(s.Scheme(), t)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		return NewObject(rs), nil
	default:
		return NewUnit(), nil
	}
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
func NativeAssetBalanceV3(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetBalanceV3"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	addressOrAliasExpr, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	r, err := extractRecipient(addressOrAliasExpr)
	if err != nil {
		return nil, errors.Errorf("%s: first argument %v", funcName, err)
	}
	assetId, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
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
		return nil, errors.Errorf("%s: expected second argument to be *BytesExpr, found %T", funcName, assetId)
	}
	balance, err := s.State().NewestAccountBalance(r, assetBts.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewLong(int64(balance)), nil
}

func NativeAssetBalanceV4(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetBalanceV4"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	}
	addressOrAliasExpr, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	r, err := extractRecipient(addressOrAliasExpr)
	if err != nil {
		return nil, errors.Errorf("%s: first argument %v", funcName, err)
	}
	assetId, err := e[1].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	assetBts, ok := assetId.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be *BytesExpr, found %T", funcName, assetId)
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
		return NewUnit(), nil
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
		return NewUnit(), nil
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
		return NewUnit(), nil
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
		return NewUnit(), nil
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
func NativeAssetInfoV3(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetInfoV3"
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
	return NewAssetInfo(newMapAssetInfoV3(*info)), nil
}

func NativeAssetInfoV4(s Scope, e Exprs) (Expr, error) {
	const funcName = "NativeAssetInfoV4"
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
	info, err := s.State().NewestFullAssetInfo(assetId)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewAssetInfo(newMapAssetInfoV4(*info)), nil
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

	bi, err := NewBlockInfo(s.Scheme(), h, proto.Height(height.Value))
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}

	return bi, nil
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

func checkedDataEntry(entryType proto.DataValueType) Callable {
	return func(s Scope, e Exprs) (Expr, error) {
		const funcName = "CheckedDataEntry"
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
		var typedEntry Expr
		var entryName string
		switch entryType {
		case proto.DataInteger:
			typedEntry, ok = rs[1].(*LongExpr)
			entryName = "IntegerEntry"
		case proto.DataString:
			typedEntry, ok = rs[1].(*StringExpr)
			entryName = "StringEntry"
		case proto.DataBinary:
			typedEntry, ok = rs[1].(*BytesExpr)
			entryName = "BinaryEntry"
		case proto.DataBoolean:
			typedEntry, ok = rs[1].(*BooleanExpr)
			entryName = "BooleanEntry"
		default:
			return nil, errors.Errorf("%s: unsupported data type %T", funcName, entryType)
		}
		if !ok {
			return nil, errors.Errorf("%s: invalid value type for %s", funcName, entryName)
		}
		return NewDataEntry(key.Value, typedEntry), nil
	}
}

func DeleteEntry(s Scope, e Exprs) (Expr, error) {
	const funcName = "DeleteEntry"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	key, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: first argument expected to be *StringExpr, found %T", funcName, rs[0])
	}
	return NewDataEntryDeleteExpr(key.Value), nil
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

func UserWavesBalanceV3(s Scope, e Exprs) (Expr, error) {
	return NativeAssetBalanceV3(s, append(e, NewUnit()))
}

func UserWavesBalanceV4(s Scope, e Exprs) (Expr, error) {
	const funcName = "UserWavesBalanceV4"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	}
	addressOrAliasExpr, err := e[0].Evaluate(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	r, err := extractRecipient(addressOrAliasExpr)
	if err != nil {
		return nil, errors.Errorf("%s: first argument %v", funcName, err)
	}
	balance, err := s.State().NewestFullWavesBalance(r)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewBalanceDetailsExpr(balance), nil
}

func limitedRSAVerify(limit int) Callable {
	fn := "RSAVerify"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 4 {
			return nil, errors.Errorf("%s: invalid number of parameters, expected 4, received %d", fn, l)
		}
		rs, err := e.EvaluateAll(s)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		digest, err := digest(rs[0])
		if err != nil {
			return nil, errors.Wrapf(err, "%s: failed to get digest algorithm from first argument", fn)
		}
		message, ok := rs[1].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: second argument expected to be *BytesExpr, found %T", fn, rs[1])
		}
		sig, ok := rs[2].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: third argument expected to be *BytesExpr, found %T", fn, rs[2])
		}
		pk, ok := rs[3].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: 4th argument expected to be *BytesExpr, found %T", fn, rs[3])
		}
		if l := len(message.Value); l > MaxBytesToVerify || limit > 0 && l > limit*1024 {
			return nil, errors.Errorf("%s: invalid message size %d bytes", fn, l)
		}
		key, err := x509.ParsePKIXPublicKey(pk.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "%s: invalid public key", fn)
		}
		k, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.Errorf("%s: not an RSA key", fn)
		}
		d := message.Value
		if digest != 0 {
			h := digest.New()
			_, _ = h.Write(message.Value)
			d = h.Sum(nil)
		}
		ok, err = verifyPKCS1v15(k, digest, d, sig.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "%s: failed to check RSA signature", fn)
		}
		return NewBoolean(ok), nil
	}
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

// Issue is a constructor of IssueExpr type
func Issue(s Scope, e Exprs) (Expr, error) {
	const funcName = "Issue"
	if l := len(e); l != 7 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 7, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	name, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be '*StringExpr', got '%T'", funcName, rs[0])
	}
	description, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be '*StringExpr', got '%T'", funcName, rs[1])
	}
	quantity, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected third argument to be '*LongExpr', got '%T'", funcName, rs[2])
	}
	decimals, ok := rs[3].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected forth argument to be '*LongExpr', got '%T'", funcName, rs[3])
	}
	reissuable, ok := rs[4].(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected 5th argument to be '*BooleanExpr', got '%T'", funcName, rs[4])
	}
	//TODO: in V4 parameter #5 "script" is always Unit, reserved for future use, here we just check the type
	_, ok = rs[5].(*Unit)
	if !ok {
		return nil, errors.Errorf("%s: expected 6th argument to be 'Unit', got '%T'", funcName, rs[5])
	}
	nonce, ok := rs[6].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected 7th argument to be '*LongExpr', got '%T'", funcName, rs[6])
	}
	return NewIssueExpr(name.Value, description.Value, quantity.Value, decimals.Value, reissuable.Value, nonce.Value), nil
}

// SimplifiedIssue is a constructor of IssueExpr with some parameters set to default values
func SimplifiedIssue(s Scope, e Exprs) (Expr, error) {
	const funcName = "SimplifiedIssue"
	if l := len(e); l != 5 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 7, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	name, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be '*StringExpr', got '%T'", funcName, rs[0])
	}
	description, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be '*StringExpr', got '%T'", funcName, rs[1])
	}
	quantity, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected third argument to be '*LongExpr', got '%T'", funcName, rs[2])
	}
	decimals, ok := rs[3].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected forth argument to be '*LongExpr', got '%T'", funcName, rs[3])
	}
	reissuable, ok := rs[4].(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected 5th argument to be '*BooleanExpr', got '%T'", funcName, rs[4])
	}
	return NewIssueExpr(name.Value, description.Value, quantity.Value, decimals.Value, reissuable.Value, 0), nil
}

// Reissue is a constructor of ReissueExpr type
func Reissue(s Scope, e Exprs) (Expr, error) {
	const funcName = "Reissue"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 3, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	assetID, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be '*BytesExpr', got '%T'", funcName, rs[0])
	}
	reissuable, ok := rs[1].(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be '*BooleanExpr', got '%T'", funcName, rs[1])
	}
	quantity, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected third argument to be '*LongExpr', got '%T'", funcName, rs[2])
	}
	r, err := NewReissueExpr(assetID.Value, quantity.Value, reissuable.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return r, nil
}

func Burn(s Scope, e Exprs) (Expr, error) {
	const funcName = "Burn"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	assetID, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be '*BytesExpr', got '%T'", funcName, rs[0])
	}
	quantity, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be '*LongExpr', got '%T'", funcName, rs[1])
	}
	r, err := NewBurnExpr(assetID.Value, quantity.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return r, nil
}

func Sponsorship(s Scope, e Exprs) (Expr, error) {
	const funcName = "Sponsorship"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	assetID, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument to be '*BytesExpr', got '%T'", funcName, rs[0])
	}
	minFee, ok := rs[1].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument to be '*LongExpr', got '%T'", funcName, rs[1])
	}
	r, err := NewSponsorshipExpr(assetID.Value, minFee.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return r, nil
}

func Contains(s Scope, e Exprs) (Expr, error) {
	const funcName = "Contains"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	str, ok := rs[0].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument of type '*StringExpr', got '%T'", funcName, rs[0])
	}
	substr, ok := rs[1].(*StringExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument of type '*StringExpr', got '%T'", funcName, rs[1])
	}
	return NewBoolean(strings.Contains(str.Value, substr.Value)), nil
}

func ValueOrElse(s Scope, e Exprs) (Expr, error) {
	const funcName = "ValueOrElse"
	if l := len(e); l != 2 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 2, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	if _, ok := rs[0].(*Unit); ok {
		return rs[1], nil
	}
	return rs[0], nil
}

func CalculateAssetID(s Scope, e Exprs) (Expr, error) {
	const funcName = "CalculateAssetID"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	issue, ok := rs[0].(*IssueExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected argument of type '*IssueExpr', got '%T'", funcName, rs[0])
	}
	txID, ok := s.Value("txId")
	if !ok {
		return nil, errors.Errorf("%s: no txId in scope", funcName)
	}
	idb, ok := txID.(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: invalid type of txId: '%T'", funcName, txID)
	}
	d, err := crypto.NewDigestFromBytes(idb.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	id := proto.GenerateIssueScriptActionID(issue.Name, issue.Description, issue.Decimals, issue.Quantity, issue.Reissuable, issue.Nonce, d)
	return NewBytes(id.Bytes()), nil
}

func TransferFromProtobuf(s Scope, e Exprs) (Expr, error) {
	const funcName = "TransferFromProtobuf"
	if l := len(e); l != 1 {
		return nil, errors.Errorf("%s: invalid number of parameters, expected 1, received %d", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	bytesExpr, ok := rs[0].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected argument of type *BytesExpr, got '%T'", funcName, rs[0])
	}
	var tx proto.TransferWithProofs
	err = tx.UnmarshalSignedFromProtobuf(bytesExpr.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	err = tx.GenerateID(s.Scheme())
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	//TODO: using scope's scheme is not quite correct here, because it should be possible to validate transfers from other networks
	obj, err := newVariablesFromTransferWithProofs(s.Scheme(), &tx)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	return NewObject(obj), nil
}

func RebuildMerkleRoot(s Scope, e Exprs) (Expr, error) {
	const funcName = "RebuildMerkleRoot"
	if l := len(e); l != 3 {
		return nil, errors.Errorf("%s: invalid number of parameters %d, expected 3", funcName, l)
	}
	rs, err := e.EvaluateAll(s)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	proofsExpr, ok := rs[0].(Exprs)
	if !ok {
		return nil, errors.Errorf("%s: expected first argument of type Exprs, got '%T'", funcName, rs[0])
	}
	if l := len(proofsExpr); l > 16 {
		return nil, errors.Errorf("%s: too many proofs %d, expected no more than 16", funcName, l)
	}
	proofs := make([]crypto.Digest, len(proofsExpr))
	for i, x := range proofsExpr {
		b, ok := x.(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: unexpected element of type '%T' of proofs array at position %d", funcName, x, i)
		}
		d, err := crypto.NewDigestFromBytes(b.Value)
		if err != nil {
			return nil, errors.Wrap(err, funcName)
		}
		proofs[i] = d
	}
	leafExpr, ok := rs[1].(*BytesExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected second argument of type *BytesExpr, got '%T'", funcName, rs[1])
	}
	leaf, err := crypto.NewDigestFromBytes(leafExpr.Value)
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	indexExpr, ok := rs[2].(*LongExpr)
	if !ok {
		return nil, errors.Errorf("%s: expected third argument of type *LongExpr, got '%T'", funcName, rs[2])
	}
	index := uint64(indexExpr.Value)
	tree, err := crypto.NewMerkleTree()
	if err != nil {
		return nil, errors.Wrap(err, funcName)
	}
	root := tree.RebuildRoot(leaf, proofs, index)
	return NewBytes(root[:]), nil
}

func limitedGroth16Verify(limit int) Callable {
	fn := "Groth16Verify"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dinputs", fn, limit)
	}
	return func(s Scope, e Exprs) (Expr, error) {
		if l := len(e); l != 3 {
			return nil, errors.Errorf("%s: invalid number of parameters %d, expected %d", fn, l, 3)
		}
		rs, err := e.EvaluateAll(s)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		key, ok := rs[0].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: expected first argument of type *BytesExpr, received %T", fn, rs[0])
		}
		proof, ok := rs[1].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: expected second argument of type *BytesExpr, received %T", fn, rs[1])
		}
		inputs, ok := rs[2].(*BytesExpr)
		if !ok {
			return nil, errors.Errorf("%s: expected third argument of type *BytesExpr, received %T", fn, rs[1])
		}
		if l := len(inputs.Value); l > 32*limit {
			return nil, errors.Errorf("%s: invalid size of inputs %d bytes, must not exceed %d bytes", fn, l, limit*32)
		}
		ok, err = crypto.Groth16Verify(key.Value, proof.Value, inputs.Value)
		if err != nil {
			return nil, errors.Wrap(err, fn)
		}
		return NewBoolean(ok), nil
	}
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

func extractRecipient(e Expr) (proto.Recipient, error) {
	var r proto.Recipient
	switch a := e.(type) {
	case *AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(*a))
	case *AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(*a))
	case *RecipientExpr:
		r = proto.Recipient(*a)
	default:
		return proto.Recipient{}, errors.Errorf("expected to be AddressExpr or AliasExpr, found %T", e)
	}
	return r, nil
}

func extractRecipientAndKey(s Scope, e Exprs) (proto.Recipient, string, error) {
	if l := len(e); l != 2 {
		return proto.Recipient{}, "", errors.Errorf("invalid params, expected 2, passed %d", l)
	}
	first, err := e[0].Evaluate(s)
	if err != nil {
		return proto.Recipient{}, "", err
	}
	r, err := extractRecipient(first)
	if err != nil {
		return proto.Recipient{}, "", errors.Errorf("first argument %v", err)
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
