package vm

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/jvm"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

//func withLong(expr ast.Expr, f func(l int64) error) error {

//}

//type With struct {
//}
//
//func (a *With) Long(f func(long int64, w *With) error) error {
//
//}

// expects f looks like `func(x int64) error`
func with(s Context, f interface{}) error {
	v := reflect.ValueOf(f)
	x := v.Type()

	if x.NumOut() != 1 {
		return errors.Errorf("expected passed function returns exactly 1 arguument, passed %d, %s", x.NumOut(), x.Name())
	}
	args := make([]reflect.Value, x.NumIn())
	for i := x.NumIn() - 1; i >= 0; i-- {
		inV := x.In(i)
		value := s.Pop()
		if value == nil {
			return errors.Errorf("with: empty stack")
		}
		// TODO check outher types
		switch inV.Kind() {
		case reflect.Int64:
			args[i] = reflect.ValueOf(value.(*ast.LongExpr).Value)
		case reflect.Bool:
			args[i] = reflect.ValueOf(value.(*ast.BooleanExpr).Value)
		case reflect.String:
			v, ok := value.(*ast.StringExpr)
			if !ok {
				return errors.Errorf("with: expected '%d' argument to be *ast.StringExpr, found %T", i, value)
			}
			args[i] = reflect.ValueOf(v.Value)
		case reflect.Interface:
			args[i] = reflect.ValueOf(value)
		case reflect.Slice: // []byte
			args[i] = reflect.ValueOf(value.(*ast.BytesExpr).Value)
		}
	}
	resp := v.Call(args)[0]
	if err, ok := resp.Interface().(error); ok {
		return err
	}
	return nil
}

//func GteLong(s Context) error {
//	second := s.Pop().(*ast.LongExpr).Value
//	first := s.Pop().(*ast.LongExpr).Value
//	s.Push(ast.NewBoolean(first >= second))
//	return nil
//}

func GteLong(s Context) error {
	//return with(s).Long(func(second int64, w *With) error {
	//	return w.Long(func(first int64, w *With) error {
	//		s.Push(ast.NewBoolean(first >= second))
	//		return nil
	//	})
	//})
	//
	//w := with(s)
	//w.Second().AsLong()

	//first := s.Pop().L
	//s.Push(ast.NewBoolean(first >= second))
	//return nil
	return with(s, func(first int64, second int64) error {
		s.Push(ast.NewBoolean(first >= second))
		return nil
	})
}

func Eq(s Context) error {
	second := s.Pop()
	first := s.Pop()

	//if first.IsObject() {
	//	s.Push(B(first.O.Eq(second.O)))
	//	return nil
	//}
	s.Push(ast.NewBoolean(first.Eq(second)))
	return nil
}

func GetterFn(s Context) error {
	second := s.Pop().(*ast.StringExpr)
	first := s.Pop()
	g, ok := first.(ast.Getable)
	if !ok {
		return errors.Errorf("GetterFn: expected first argument to be ast.Getable, found %T", first)
	}
	expr, err := g.Get(second.Value)
	if err != nil {
		return err
	}
	s.Push(expr)
	return nil
}

func UserFunctionNeq(s Context) error {
	err := Eq(s)
	if err != nil {
		return err
	}
	return with(s, func(v bool) error {
		return s.Push(ast.NewBoolean(!v))
	})
}

func IsInstanceOf(s Context) error {
	return with(s, func(first ast.Expr, second string) error {
		s.Push(ast.NewBoolean(first.InstanceOf() == second))
		return nil
	})
}

// Size of list
func NativeSizeList(s Context) error {
	const funcName = "NativeSizeList"
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	e := s.Pop()
	// optimize not evaluate inner list
	if v, ok := e.(ast.Exprs); ok {
		s.Push(ast.NewLong(int64(len(v))))
		return nil
	}
	return errors.Errorf("%s: expected first argument to be ast.Expr, got %T", funcName, e)
	//rs, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//lst, ok := rs.(Exprs)
	//if !ok {
	//	return nil, errors.Errorf("%s: expected first argument Exprs, got %T", funcName, rs)
	//}
	//return NewLong(int64(len(lst))), nil
}

// type constructor
func UserAddress(s Context) error {
	//const funcName = "UserAddress"
	return with(s, func(value []byte) error {
		addr, err := proto.NewAddressFromBytes(value)
		if err != nil {
			s.Push(&ast.InvalidAddressExpr{Value: value})
			return nil
		}
		s.Push(ast.NewAddressFromProtoAddress(addr))
		return nil
	})
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//first = s.Pop()
	////if err != nil {
	////	return nil, errors.Wrap(err, funcName)
	////}
	//bts, ok := first.(*BytesExpr)
	//if !ok {
	//	return nil, errors.Errorf("%s: first argument expected to be *BytesExpr, found %T", funcName, first)
	//}
	//addr, err := proto.NewAddressFromBytes(bts.Value)
	//if err != nil {
	//	return &ast.InvalidAddressExpr{Value: bts.Value}, nil
	//}
	//return NewAddressFromProtoAddress(addr), nil
}

//// Decode account address
//func UserAddressFromString(s Context) error {
//	str := s.Pop().(*ast.StringExpr).Value
//	//proto.NewAddressFromString(str)
//
//	//rs, err := e[0].Evaluate(s)
//	//if err != nil {
//	//	return nil, errors.Wrap(err, "UserAddressFromString")
//	//}
//	//str, ok := rs.(*StringExpr)
//	//if !ok {
//	//	return nil, errors.Errorf("UserAddressFromString: expected first argument to be *StringExpr, found %T", rs)
//	//}
//	addr, err := ast.NewAddressFromString(str)
//	if err != nil {
//		s.Push(ast.NewUnit())
//		return nil
//	}
//	// TODO return this back
//	//if addr[1] != s.Scheme() {
//	//	return NewUnit(), nil
//	//}
//	s.Push(addr)
//	return nil
//}

func NativeCreateList(s Context) error {
	const funcName = "NativeCreateList"
	second := s.Pop()
	head := s.Pop()
	//if l := len(e); l != 2 {
	//	return nil, errors.Errorf("%s: invalid parameters, expected 2, received %d", funcName, l)
	//}
	//head, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//t, err := e[1].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	tail, ok := second.(ast.Exprs)
	if !ok {
		return errors.Errorf("%s: invalid second parameter, expected Exprs, received %T", funcName, second)
	}
	if len(tail) == 0 {
		s.Push(ast.NewExprs(head))
		//return NewExprs(head), nil
		return nil
	}
	//return append(ast.NewExprs(head), tail...), nil
	s.Push(append(ast.NewExprs(head), tail...))
	return nil
}

// Get list element by position
func NativeGetList(s Context) error {
	const funcName = "NativeGetList"
	//if l := len(e); l != 2 {
	//	return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	//}
	second := s.Pop()
	first := s.Pop()

	lst, ok := first.(ast.Exprs)
	if !ok {
		return errors.Errorf("%s: expected first argument Exprs, got %T", funcName, first)
	}
	lng, ok := second.(*ast.LongExpr)
	if !ok {
		return errors.Errorf("%s: expected second argument *LongExpr, got %T", funcName, second)
	}
	if lng.Value < 0 || lng.Value >= int64(len(lst)) {
		return errors.Errorf("%s: invalid index %d, len %d", funcName, lng.Value, len(lst))
	}
	s.Push(lst[lng.Value])
	return nil
}

// Fail script
func NativeThrow(s Context) error {
	const funcName = "NativeThrow"
	first := s.Pop()
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//first, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	str, ok := first.(*ast.StringExpr)
	if !ok {
		return errors.Errorf("%s: expected first argument to be *StringExpr, found %T", funcName, first)
	}
	return &ast.Throw{
		Message: str.Value,
	}
}

func Throw(message string) error {
	return &ast.Throw{
		Message: message,
	}
}

func SigVerifyV2(s Context) error {
	return with(s, func(data []byte, sigBytes []byte, publicKey []byte) error {
		//if l := len(data); !s.validMessageLength(l) || limit > 0 && l > limit*1024 {
		//	return errors.Errorf("%s: invalid message size %d", fn, l)
		//}

		signature, err := crypto.NewSignatureFromBytes(sigBytes)
		if err != nil {
			return s.Push(ast.NewBoolean(false)) //errs.Extendf(err, "bytes len: %d", len(sigBytes))
		}
		pk, err := crypto.NewPublicKeyFromBytes(publicKey)
		if err != nil {
			return s.Push(ast.NewBoolean(false))
		}
		out := crypto.Verify(pk, signature, data)
		return s.Push(ast.NewBoolean(out))
	})
}

//func limitedSigVerify(limit int) Func {
//	fn := "SigVerify"
//	if limit > 0 {
//		fn = fmt.Sprintf("%s_%dKb", fn, limit)
//	}
//	return func(s Context) error {
//		return with(s, func(data []byte, sigBytes []byte, publicKey []byte) error {
//			if l := len(data); !s.validMessageLength(l) || limit > 0 && l > limit*1024 {
//				return errors.Errorf("%s: invalid message size %d", fn, l)
//			}
//
//			signature, err := crypto.NewSignatureFromBytes(sigBytes)
//			if err != nil {
//				return err
//			}
//			pk, err := crypto.NewPublicKeyFromBytes(publicKey)
//			if err != nil {
//				return err
//			}
//			out := crypto.Verify(pk, signature, data)
//			s.Push(ast.NewBoolean(out))
//			return nil
//		})
//		//if l := len(e); l != 3 {
//		//	return nil, errors.Errorf("%s: invalid number of parameters %d, expected 3", fn, l)
//		//}
//		//rs, err := e.EvaluateAll(s)
//		//if err != nil {
//		//	return nil, errors.Wrap(err, fn)
//		//}
//		/*
//			messageExpr, ok := rs[0].(*BytesExpr)
//			if !ok {
//				return nil, errors.Errorf("%s: first argument expects to be *BytesExpr, found %T", fn, rs[0])
//			}
//			if l := len(messageExpr.Value); !s.validMessageLength(l) || limit > 0 && l > limit*1024 {
//				return nil, errors.Errorf("%s: invalid message size %d", fn, l)
//			}
//			signatureExpr, ok := rs[1].(*BytesExpr)
//			if !ok {
//				return nil, errors.Errorf("%s: second argument expects to be *BytesExpr, found %T", fn, rs[1])
//			}
//			pkExpr, ok := rs[2].(*BytesExpr)
//			if !ok {
//				return nil, errors.Errorf("%s: third argument expects to be *BytesExpr, found %T", fn, rs[2])
//			}
//			pk, err := crypto.NewPublicKeyFromBytes(pkExpr.Value)
//			if err != nil {
//				return NewBoolean(false), nil
//			}
//			signature, err := crypto.NewSignatureFromBytes(signatureExpr.Value)
//			if err != nil {
//				return NewBoolean(false), nil
//			}
//			out := crypto.Verify(pk, signature, messageExpr.Value)
//			return NewBoolean(out), nil
//		*/
//	}
//}

func extractRecipient(e ast.Expr) (proto.Recipient, error) {
	var r proto.Recipient
	switch a := e.(type) {
	case *ast.AddressExpr:
		r = proto.NewRecipientFromAddress(proto.Address(*a))
	case *ast.AliasExpr:
		r = proto.NewRecipientFromAlias(proto.Alias(*a))
	case *ast.RecipientExpr:
		r = proto.Recipient(*a)
	default:
		return proto.Recipient{}, errors.Errorf("expected to be AddressExpr or AliasExpr, found %T", e)
	}
	return r, nil
}

func extractRecipientAndKey(s Context) (proto.Recipient, string, error) {
	//if l := len(e); l != 2 {
	//	return proto.Recipient{}, "", errors.Errorf("invalid params, expected 2, passed %d", l)
	//}

	second := s.Pop()
	first := s.Pop()
	//if err != nil {
	//	return proto.Recipient{}, "", err
	//}
	r, err := extractRecipient(first)
	if err != nil {
		return proto.Recipient{}, "", errors.Errorf("first argument %v", err)
	}

	//if err != nil {
	//	return proto.Recipient{}, "", err
	//}
	key, ok := second.(*ast.StringExpr)
	if !ok {
		return proto.Recipient{}, "", errors.Errorf("second argument expected to be *StringExpr, found %T", second)
	}
	return r, key.Value, nil
}

// Get integer from account state
func NativeDataIntegerFromState(s Context) error {
	r, k, err := extractRecipientAndKey(s)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	entry, err := s.State().RetrieveNewestIntegerEntry(r, k)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	return s.Push(ast.NewLong(entry.Value))
}

// Decode account address
func UserAddressFromString(s Context) error {
	return with(s, func(value string) error {
		addr, err := ast.NewAddressFromString(value)
		if err != nil {
			return s.Push(ast.NewUnit())
		}
		if addr[1] != s.Scheme() {
			return s.Push(ast.NewUnit())
		}
		return s.Push(addr)
	})
}

// Integer sum
func NativeSumLong(s Context) error {
	return with(s, func(i int64, i2 int64) error {
		return s.Push(ast.NewLong(i + i2))
	})
}

func NativeGtLong(s Context) error {
	return with(s, func(i int64, i2 int64) error {
		return s.Push(ast.NewBoolean(i > i2))
	})
}

func UserExtract(s Context) error {
	return with(s, func(val ast.Expr) error {
		if val.InstanceOf() == (&ast.Unit{}).InstanceOf() {
			return Throw("extract() called on unit value")
		}
		return s.Push(val)
	})
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//val, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//if val.InstanceOf() == (&Unit{}).InstanceOf() {
	//	return NativeThrow(s, Params(NewString("extract() called on unit value")))
	//}
	//return val, nil
}

// Modulo
func NativeModLong(s Context) error {
	return with(s, func(i int64, i2 int64) error {
		if i2 == 0 {
			return errors.New("zero division")
		}
		return s.Push(ast.NewLong(jvm.ModDivision(i, i2)))
	})
}

// Integer substitution
func NativeSubLong(s Context) error {
	return with(s, func(i int64, i2 int64) error {
		return s.Push(ast.NewLong(i - i2))
	})
}

// Integer multiplication
func NativeMulLong(s Context) error {
	return with(s, func(i int64, i2 int64) error {
		return s.Push(ast.NewLong(i * i2))
	})
}

// Integer division
func NativeDivLong(s Context) error {
	return with(s, func(x int64, y int64) error {
		if y == 0 {
			return errors.New("zero division")
		}
		return s.Push(ast.NewLong(jvm.FloorDiv(x, y)))
	})
}

// Get string from account state
func NativeDataStringFromState(s Context) error {
	r, k, err := extractRecipientAndKey(s)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	entry, err := s.State().RetrieveNewestStringEntry(r, k)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	return s.Push(ast.NewString(entry.Value))
}

func UserIsDefined(s Context) error {
	return with(s, func(val ast.Expr) error {
		return s.Push(ast.NewBoolean(val.InstanceOf() != (&ast.Unit{}).InstanceOf()))
	})
	//const funcName = "UserIsDefined"
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//val, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//if val.InstanceOf() == (&Unit{}).InstanceOf() {
	//	return NewBoolean(false), nil
	//}
	//return NewBoolean(true), nil
}

func UserUnaryNot(s Context) error {
	return with(s, func(val bool) error {
		return s.Push(ast.NewBoolean(!val))
	})
}

func UserWavesBalanceV3(s Context) error {
	_ = s.Push(ast.NewUnit())
	return NativeAssetBalanceV3(s) //, append(e, NewUnit()))
}

// Asset balance for account
func NativeAssetBalanceV3(s Context) error {
	const funcName = "NativeAssetBalanceV3"
	return with(s, func(addressOrAliasExpr ast.Expr, assetId ast.Expr) error {
		//if l := len(e); l != 2 {
		//	return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
		//}
		//addressOrAliasExpr, err := e[0].Evaluate(s)
		//if err != nil {
		//	return nil, errors.Wrap(err, funcName)
		//}
		r, err := extractRecipient(addressOrAliasExpr)
		if err != nil {
			return errors.Errorf("%s: first argument %v", funcName, err)
		}
		//assetId, err := e[1].Evaluate(s)
		//if err != nil {
		//	return nil, errors.Wrap(err, funcName)
		//}
		if _, ok := assetId.(*ast.Unit); ok {
			balance, err := s.State().NewestAccountBalance(r, nil)
			if err != nil {
				return errors.Wrap(err, funcName)
			}
			return s.Push(ast.NewLong(int64(balance)))
		}
		assetBts, ok := assetId.(*ast.BytesExpr)
		if !ok {
			return errors.Errorf("%s: expected second argument to be *BytesExpr, found %T", funcName, assetId)
		}
		balance, err := s.State().NewestAccountBalance(r, assetBts.Value)
		if err != nil {
			return errors.Wrap(err, funcName)
		}
		return s.Push(ast.NewLong(int64(balance)))
	})
}

func UserAddressFromPublicKey(s Context) error {
	const funcName = "UserAddressFromPublicKey"
	return with(s, func(val []byte) error {
		addr, err := proto.NewAddressLikeFromAnyBytes(s.Scheme(), val)
		if err != nil {
			return s.Push(ast.NewUnit())
		}
		return s.Push(ast.NewAddressFromProtoAddress(addr))
	})
}

func dataFromArray(s Context) error {
	return with(s, func(first ast.Expr, second ast.Expr) error {
		lst, ok := first.(ast.Exprs)
		if !ok {
			return errors.Errorf("expected first argument to be *Exprs, found %T", first)
		}
		key, ok := second.(*ast.StringExpr)
		if !ok {
			return errors.Errorf("expected second argument to be *StringExpr, found %T", second)
		}
		for i, e := range lst {
			item, ok := e.(ast.Getable)
			if !ok {
				return errors.Errorf("unexpected list element of type %T", e)
			}
			k, err := item.Get("key")
			if err != nil {
				return errors.Wrapf(err, "%dth element doesn't have 'key' field", i)
			}
			if key.Eq(k) {
				v, err := item.Get("value")
				if err != nil {
					return errors.Wrapf(err, "%dth element doesn't have 'value' field", i)
				}
				return s.Push(v)
			}
		}
		return s.Push(ast.NewUnit())
	})
}

// Get integer from data of DataTransaction
func NativeDataIntegerFromArray(s Context) error {
	err := dataFromArray(s)
	if err != nil {
		return errors.Wrap(err, "NativeDataIntegerFromArray")
	}
	d := s.Pop()
	_, ok := d.(*ast.LongExpr)
	if !ok {
		return s.Push(ast.NewUnit())
	}
	return s.Push(d)
}

// Base58 decode
func NativeFromBase58(s Context) error {
	const funcName = "NativeFromBase58"
	return with(s, func(val string) error {
		if val == "" {
			return s.Push(ast.NewBytes(nil))
		}
		rs, err := base58.Decode(val)
		if err != nil {
			return errors.Wrap(err, funcName)
		}
		return s.Push(ast.NewBytes(rs))
	})

	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//first, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//str, ok := first.(*StringExpr)
	//if !ok {
	//	return nil, errors.Errorf("%s: expected first argument to be *StringExpr, found %T", funcName, first)
	//}
	//if str.Value == "" {
	//	return NewBytes(nil), nil
	//}
	//rs, err := base58.Decode(str.Value)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//return NewBytes(rs), nil
}

// String representation
func NativeLongToString(s Context) error {
	return with(s, func(i int64) error {
		return s.Push(ast.NewString(strconv.FormatInt(i, 10)))
	})
	//if l := len(e); l != 1 {
	//	return nil, errors.Errorf("%s: invalid params, expected 1, passed %d", funcName, l)
	//}
	//first, err := e[0].Evaluate(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//long, ok := first.(*LongExpr)
	//if !ok {
	//	return nil, errors.Errorf("%s: expected first argument to be *LongExpr, found %T", funcName, first)
	//}
	//return NewString(fmt.Sprintf("%d", long.Value)), nil
}

// Base58 encode
func NativeToBase58(s Context) error {
	const funcName = "NativeToBase58"
	return with(s, func(first ast.Expr) error {
		switch arg := first.(type) {
		case *ast.BytesExpr:
			return s.Push(ast.NewString(base58.Encode(arg.Value)))
		case *ast.Unit:
			return s.Push(ast.NewString(base58.Encode(nil)))
		default:
			return errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", funcName, first)
		}
	})
}

// Get string from data of DataTransaction
func NativeDataStringFromArray(s Context) error {
	err := dataFromArray(s)
	if err != nil {
		return errors.Wrap(err, "NativeDataStringFromArray")
	}
	d := s.Pop()
	_, ok := d.(*ast.StringExpr)
	if !ok {
		return s.Push(ast.NewUnit())
	}
	return s.Push(d)
}

// Lookup transaction
func NativeTransactionByID(s Context) error {
	const funcName = "NativeTransactionByID"
	return with(s, func(val []byte) error {
		tx, err := s.State().NewestTransactionByID(val)
		if err != nil {
			if s.State().IsNotFound(err) {
				return s.Push(ast.NewUnit())
			}
			return errors.Wrap(err, funcName)
		}
		vars, err := ast.NewVariablesFromTransaction(s.Scheme(), tx)
		if err != nil {
			return errors.Wrap(err, funcName)
		}
		return s.Push(ast.NewObject(vars))
	})
}

// Limited strings concatenation
func NativeConcatStrings(s Context) error {
	const funcName = "NativeConcatStrings"
	return with(s, func(prefix string, suffix string) error {
		l := len(prefix) + len(suffix)
		if l > ast.MaxBytesResult {
			return errors.Errorf("%s byte length %d is greater than max %d", funcName, l, ast.MaxBytesResult)
		}
		out := prefix + suffix
		lengthInRunes := utf8.RuneCountInString(out)
		if lengthInRunes > ast.MaxStringResult {
			return errors.Errorf("%s string length %d is greater than max %d", funcName, lengthInRunes, ast.MaxStringResult)
		}
		return s.Push(ast.NewString(out))
	})
	//
	//if l := len(e); l != 2 {
	//	return nil, errors.Errorf("%s: invalid params, expected 2, passed %d", funcName, l)
	//}
	//rs, err := e.EvaluateAll(s)
	//if err != nil {
	//	return nil, errors.Wrap(err, funcName)
	//}
	//prefix, ok := rs[0].(*StringExpr)
	//if !ok {
	//	return nil, errors.Errorf("%s expected first argument to be *StringExpr, found %T", funcName, rs[0])
	//}
	//suffix, ok := rs[1].(*StringExpr)
	//if !ok {
	//	return nil, errors.Errorf("%s expected second argument to be *StringExpr, found %T", funcName, rs[1])
	//}
	//l := len(prefix.Value) + len(suffix.Value)
	//if l > MaxBytesResult {
	//	return nil, errors.Errorf("%s byte length %d is greater than max %d", funcName, l, MaxBytesResult)
	//}
	//out := prefix.Value + suffix.Value
	//lengthInRunes := utf8.RuneCountInString(out)
	//if lengthInRunes > MaxStringResult {
	//	return nil, errors.Errorf("%s string length %d is greater than max %d", funcName, lengthInRunes, MaxStringResult)
	//}
	//return NewString(out), nil
}

//// Decode account address
//func UserAddressFromString(s Context) error {
//	return with(s, func(val string) error {
//		addr, err := NewAddressFromString(str.Value)
//		if err != nil {
//			return NewUnit(), nil
//		}
//		if addr[1] != s.Scheme() {
//			return NewUnit(), nil
//		}
//		return addr, nil
//	})
//	if l := len(e); l != 1 {
//		return nil, errors.Errorf("UserAddressFromString: invalid params, expected 1, passed %d", l)
//	}
//	rs, err := e[0].Evaluate(s)
//	if err != nil {
//		return nil, errors.Wrap(err, "UserAddressFromString")
//	}
//	str, ok := rs.(*StringExpr)
//	if !ok {
//		return nil, errors.Errorf("UserAddressFromString: expected first argument to be *StringExpr, found %T", rs)
//	}
//	addr, err := NewAddressFromString(str.Value)
//	if err != nil {
//		return NewUnit(), nil
//	}
//	if addr[1] != s.Scheme() {
//		return NewUnit(), nil
//	}
//	return addr, nil
//}

// String to bytes representation
func NativeStringToBytes(s Context) error {
	return with(s, func(val string) error {
		return s.Push(ast.NewBytes([]byte(val)))
	})
}

func limitedKeccak256(limit int) func(ctx Context) error {
	fn := "Keccak256"
	if limit > 0 {
		fn = fmt.Sprintf("%s_%dKb", fn, limit)
	}
	return func(s Context) error {
		return with(s, func(val []byte) error {
			if l := len(val); limit > 0 && l > limit {
				return errors.Errorf("%s: invalid size of data %d bytes", fn, l)
			}
			d, err := crypto.Keccak256(val)
			if err != nil {
				return errors.Wrap(err, fn)
			}
			return s.Push(ast.NewBytes(d.Bytes()))
		})
		//if l := len(e); l != 1 {
		//	return nil, errors.Errorf("%s: invalid number of parameters %d, expected 1", fn, l)
		//}
		//val, err := e[0].Evaluate(s)
		//if err != nil {
		//	return nil, errors.Wrapf(err, fn)
		//}
		//dataExpr, ok := val.(*BytesExpr)
		//if !ok {
		//	return nil, errors.Errorf("%s: expected first argument to be *BytesExpr, found %T", fn, val)
		//}
		//if l := len(dataExpr.Value); limit > 0 && l > limit {
		//	return nil, errors.Errorf("%s: invalid size of data %d bytes", fn, l)
		//}
		//d, err := crypto.Keccak256(dataExpr.Value)
		//if err != nil {
		//	return nil, errors.Wrap(err, fn)
		//}
		//return NewBytes(d.Bytes()), nil
	}
}

// Get bool from account state
func NativeDataBooleanFromState(s Context) error {
	r, k, err := extractRecipientAndKey(s)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	entry, err := s.State().RetrieveNewestBooleanEntry(r, k)
	if err != nil {
		return s.Push(ast.NewUnit())
	}
	return s.Push(ast.NewBoolean(entry.Value))
}

func NativeAddressFromRecipient(s Context) error {
	const funcName = "NativeAddressFromRecipient"
	return with(s, func(val ast.Expr) error {
		recipient, ok := val.(*ast.RecipientExpr)
		if !ok {
			return errors.Errorf("%s expected first argument to be RecipientExpr, found %T", funcName, val)
		}
		if recipient.Address != nil {
			return s.Push(ast.NewAddressFromProtoAddress(*recipient.Address))
		}
		if recipient.Alias != nil {
			addr, err := s.State().NewestAddrByAlias(*recipient.Alias)
			if err != nil {
				return errors.Wrap(err, funcName)
			}
			return s.Push(ast.NewAddressFromProtoAddress(addr))
		}
		return errors.Errorf("can't get address from recipient, recipient %v", recipient)
	})
}
