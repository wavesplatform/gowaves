package parser

import (
	"github.com/pkg/errors"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

func BuildAst(r *BytesReader) (Expr, error) {
	b := r.ReadByte()
	// first byte always should be E_BYTES
	if b != E_BYTES {
		return nil, errors.Errorf("BuildAst: invalid format, expected 1, found %d", b)
	}

	return Walk(r)
}

func Walk(iter *BytesReader) (Expr, error) {
	if iter.Eof() {
		return nil, ErrUnexpectedEOF
	}

	next := iter.Next()

	switch next {
	case E_LONG:
		return &LongExpr{
			Value: iter.ReadLong(),
		}, nil
	case E_BYTES:
		return readBytes(iter)
	case E_STRING:
		return NewString(iter.ReadString()), nil
	case E_IF:
		return readIf(iter)
	case E_BLOCK:
		return readBlock(iter)
	case E_REF:
		return &RefExpr{
			Name: iter.ReadString(),
		}, nil
	case E_TRUE:
		return NewBoolean(true), nil
	case E_FALSE:
		return NewBoolean(false), nil
	case E_GETTER:
		return readGetter(iter)
	case E_FUNCALL:
		return readFuncCAll(iter)
	default:
		return nil, errors.Errorf("invalid byte %d", next)
	}
}

func readBlock(r *BytesReader) (*Block, error) {
	letName := r.ReadString()
	letValue, err := Walk(r)
	if err != nil {
		return nil, err
	}

	body, err := Walk(r)
	if err != nil {
		return nil, err
	}

	return &Block{
		Let:  NewLet(letName, letValue),
		Body: body,
	}, nil
}

func readFuncCAll(iter *BytesReader) (*FuncCall, error) {
	nativeOrUser := iter.ReadByte()
	switch nativeOrUser {
	case FH_NATIVE:
		f, err := readNativeFunction(iter)
		if err != nil {
			return nil, err
		}
		return NewFuncCall(f), nil
	case FH_USER:
		f, err := readUserFunction(iter)
		if err != nil {
			return nil, err
		}
		return NewFuncCall(f), nil
	default:
		return nil, errors.Errorf("invalid function type, expects 0 or 1, found %d", nativeOrUser)
	}

}

func readNativeFunction(iter *BytesReader) (*NativeFunction, error) {
	funcNumber := iter.ReadShort()
	argc := iter.ReadInt()
	argv := make([]Expr, argc)

	for i := int32(0); i < argc; i++ {
		v, err := Walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}

	return NewNativeFunction(funcNumber, int(argc), argv), nil
}

func readUserFunction(iter *BytesReader) (*UserFunction, error) {
	funcNumber := iter.ReadString()
	argc := iter.ReadInt()
	argv := make([]Expr, argc)
	for i := int32(0); i < argc; i++ {
		v, err := Walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}

	return NewUserFunction(funcNumber, int(argc), argv), nil
}

func readIf(r *BytesReader) (*IfExpr, error) {
	cond, err := Walk(r)
	if err != nil {
		return nil, err
	}
	True, err := Walk(r)
	if err != nil {
		return nil, err
	}
	False, err := Walk(r)
	if err != nil {
		return nil, err
	}
	return NewIf(cond, True, False), nil
}

func readBytes(r *BytesReader) (*BytesExpr, error) {
	return NewBytes(r.ReadBytes()), nil
}

func readGetter(r *BytesReader) (*GetterExpr, error) {
	a, err := Walk(r)
	if err != nil {
		return nil, err
	}

	s := r.ReadString()
	return NewGetterExpr(a, s), nil
}
