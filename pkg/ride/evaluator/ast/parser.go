package ast

import (
	"strconv"

	"github.com/pkg/errors"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

func BuildScript(r *BytesReader) (*Script, error) {
	version, err := r.ReadByte()
	if err != nil {
		return nil, errors.Wrap(err, "parser: failed to read script version")
	}

	if version == 0 {
		dapp, err := parseDApp(r)
		if err != nil {
			return nil, err
		}
		return &Script{
			Version:    0,
			HasBlockV2: false,
			Verifier:   nil,
			DApp:       dapp,
			dApp:       true,
		}, nil
	}

	if version < 1 || version > 3 {
		return nil, errors.Errorf("parser: unsupported script version %d", version)
	}
	exp, err := Walk(r)
	if err != nil {
		return nil, errors.Wrap(err, "parser")
	}
	script := Script{
		Version:    int(version),
		HasBlockV2: false,
		Verifier:   exp,
	}
	return &script, nil
}

type DApp struct {
	DAppVersion   byte
	LibVersion    byte
	Meta          DappMeta
	Declarations  Exprs
	CallableFuncs map[string]*DappCallableFunc
	Verifier      *DappCallableFunc
}

type DappMeta struct {
	Version int32
	Bytes   []byte
}

func parseDApp(r *BytesReader) (DApp, error) {
	dApp := DApp{}
	dApp.DAppVersion = r.Next()
	dApp.LibVersion = r.Next()
	// meta
	meta := DappMeta{
		Version: r.ReadInt(),
		Bytes:   r.ReadBytes(),
	}
	dApp.Meta = meta

	declarations := Exprs{}
	cnt := r.ReadInt()
	for i := int32(0); i < cnt; i++ {
		d, err := deserializeDeclaration(r)
		if err != nil {
			return dApp, err
		}
		declarations = append(declarations, d)
	}
	dApp.Declarations = declarations

	// callable func declarations
	var callableFuncs = make(map[string]*DappCallableFunc)
	cnt = r.ReadInt()
	for i := int32(0); i < cnt; i++ {
		rest := r.Rest()
		_ = rest
		annotationInvokeName := r.ReadString()
		d, err := deserializeDeclaration(r)
		if err != nil {
			return dApp, err
		}
		f, ok := d.(*FuncDeclaration)
		if !ok {
			return dApp, errors.Errorf("expected to be *FuncDeclaration, found %T", f)
		}
		callableFuncs[f.Name] = &DappCallableFunc{
			AnnotationInvokeName: annotationInvokeName,
			FuncDecl:             f,
		}
	}
	dApp.CallableFuncs = callableFuncs

	// parse verifier
	cnt = r.ReadInt()
	_ = cnt
	if cnt != 0 {
		annotationInvokeName := r.ReadString()
		d, err := deserializeDeclaration(r)
		if err != nil {
			return dApp, err
		}
		f, ok := d.(*FuncDeclaration)
		if !ok {
			return dApp, errors.Errorf("expected to be *FuncDeclaration, found %T", f)
		}
		dApp.Verifier = &DappCallableFunc{
			AnnotationInvokeName: annotationInvokeName,
			FuncDecl:             f,
		}
	}

	return dApp, nil
}

type DappCallableFunc struct {
	AnnotationInvokeName string
	FuncDecl             *FuncDeclaration
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
	case E_BLOCK_V2:
		return readBlockV2(iter)
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

func deserializeDeclaration(r *BytesReader) (Expr, error) {
	declType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch declType {
	case DEC_LET:
		name := r.ReadString()
		body, err := Walk(r)
		if err != nil {
			return nil, err
		}
		return NewLet(name, body), nil
	case DEC_FUNC:
		name := r.ReadString()
		argc := r.ReadInt()
		args := make([]string, argc)
		for i := int32(0); i < argc; i++ {
			args[i] = r.ReadString()
		}
		body, err := Walk(r)
		if err != nil {
			return nil, err
		}
		return &FuncDeclaration{
			Name: name,
			Args: args,
			Body: body,
		}, nil

	default:
		return nil, errors.Errorf("unknown declaration byte, expected %d or %d, found %d", DEC_LET, DEC_FUNC, declType)
	}
}

func readBlockV2(r *BytesReader) (*BlockV2, error) {
	rs, err := deserializeDeclaration(r)
	if err != nil {
		return nil, err
	}

	body, err := Walk(r)
	if err != nil {
		return nil, err
	}

	return &BlockV2{
		Decl: rs,
		Body: body,
	}, nil
}

func readFuncCAll(iter *BytesReader) (*FuncCallExpr, error) {
	nativeOrUser, err := iter.ReadByte()
	if err != nil {
		return nil, err
	}
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

func readNativeFunction(iter *BytesReader) (*FunctionCall, error) {
	funcNumber := iter.ReadShort()
	name := strconv.Itoa(int(funcNumber))
	argc := iter.ReadInt()
	argv := make([]Expr, argc)

	for i := int32(0); i < argc; i++ {
		v, err := Walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}
	return NewFunctionCall(name, argv), nil
}

func readUserFunction(iter *BytesReader) (*FunctionCall, error) {
	name := iter.ReadString()
	argc := iter.ReadInt()
	argv := make([]Expr, argc)
	for i := int32(0); i < argc; i++ {
		v, err := Walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}

	return NewFunctionCall(name, argv), nil
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
