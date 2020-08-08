package messages

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	. "github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/ride/op"
	"github.com/wavesplatform/gowaves/pkg/ride/transpiler"
	"go.uber.org/zap"
)

type flags struct {
	blockV2 bool
	arrays  bool
}

func BuildScript(r *BytesReader) (*Script, error) {
	f := flags{}
	version, err := r.ReadByte()
	if err != nil {
		return nil, errors.Wrap(err, "parser: failed to read script version")
	}

	if version == 0 {
		dapp, err := f.parseDApp(r)
		if err != nil {
			return nil, err
		}
		return &Script{
			Version:    int(dapp.LibVersion),
			HasBlockV2: f.blockV2,
			HasArrays:  f.arrays,
			Verifier:   nil,
			DApp:       dapp,
			dApp:       true,
		}, nil
	}

	if version < 1 || version > 4 {
		return nil, errors.Errorf("parser: unsupported script version %d", version)
	}
	exp, err := f.walk(r)
	if err != nil {
		return nil, errors.Wrap(err, "parser")
	}

	b := op.NewOpCodeBuilder()
	r2 := r.Reset().StripCheckSum()
	//r2.Next()
	errer := transpiler.ErrImpl{}
	err = transpiler.BuildCode(r2, transpiler.NewInitial(b, &errer))
	zap.S().Debugf("BUILD CODE!!!!!")
	vmCode := b.Code()
	if err != nil {
		vmCode = nil
		zap.S().Debugf("transpiler.BuildCode: %s", err)
		return nil, err
	}
	if errer.Get() != nil {
		vmCode = nil
		zap.S().Debugf("transpiler.Fsm: %s", errer.Get())
		return nil, errer.Get()
	}
	//if err != nil {
	//	return nil, errors.Wrap(err, "transpiler")
	//}

	script := Script{
		Version:    int(version),
		HasBlockV2: f.blockV2,
		HasArrays:  f.arrays,
		Verifier:   exp,
		VmCode:     vmCode,
	}
	return &script, nil
}

type DApp struct {
	DAppVersion   byte
	LibVersion    byte
	Meta          DappMeta
	Declarations  ast.Exprs
	CallableFuncs map[string]*DappCallableFunc
	Verifier      *DappCallableFunc
}

type DappMeta struct {
	Version int32
	Bytes   []byte
}

func (f *flags) parseDApp(r *BytesReader) (DApp, error) {
	dApp := DApp{}
	dApp.DAppVersion = r.Next()
	dApp.LibVersion = r.Next()
	// meta
	meta := DappMeta{
		Version: r.ReadInt(),
		Bytes:   r.ReadBytes(),
	}
	dApp.Meta = meta

	declarations := ast.Exprs{}
	cnt := r.ReadInt()
	for i := int32(0); i < cnt; i++ {
		d, err := f.deserializeDeclaration(r)
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
		d, err := f.deserializeDeclaration(r)
		if err != nil {
			return dApp, err
		}
		f, ok := d.(*ast.FuncDeclaration)
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
		d, err := f.deserializeDeclaration(r)
		if err != nil {
			return dApp, err
		}
		f, ok := d.(*ast.FuncDeclaration)
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
	FuncDecl             *ast.FuncDeclaration
}

func (f *flags) walk(iter *BytesReader) (ast.Expr, error) {
	if iter.Eof() {
		return nil, ErrUnexpectedEOF
	}

	next := iter.Next()
	switch next {
	case E_LONG:
		return &ast.LongExpr{
			Value: iter.ReadLong(),
		}, nil
	case E_BYTES:
		return readBytes(iter)
	case E_STRING:
		return ast.NewString(iter.ReadString()), nil
	case E_IF:
		return f.readIf(iter)
	case E_BLOCK:
		return f.readBlock(iter)
	case E_REF:
		return &ast.RefExpr{
			Name: iter.ReadString(),
		}, nil
	case E_TRUE:
		return ast.NewBoolean(true), nil
	case E_FALSE:
		return ast.NewBoolean(false), nil
	case E_GETTER:
		return f.readGetter(iter)
	case E_FUNCALL:
		return f.readFuncCAll(iter)
	case E_BLOCK_V2:
		b, err := f.readBlockV2(iter)
		if err != nil {
			return nil, err
		}
		f.blockV2 = true
		return b, nil
	case E_ARR:
		a, err := f.readArray(iter)
		if err != nil {
			return nil, err
		}
		f.arrays = true
		return a, nil
	default:
		return nil, errors.Errorf("invalid byte %d", next)
	}
}

func (f *flags) readBlock(r *BytesReader) (*ast.Block, error) {
	letName := r.ReadString()
	letValue, err := f.walk(r)
	if err != nil {
		return nil, err
	}

	body, err := f.walk(r)
	if err != nil {
		return nil, err
	}

	return &ast.Block{
		Let:  ast.NewLet(letName, letValue),
		Body: body,
	}, nil
}

func (f *flags) deserializeDeclaration(r *BytesReader) (ast.Expr, error) {
	declType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch declType {
	case DEC_LET:
		name := r.ReadString()
		body, err := f.walk(r)
		if err != nil {
			return nil, err
		}
		return ast.NewLet(name, body), nil
	case DEC_FUNC:
		name := r.ReadString()
		argc := r.ReadInt()
		args := make([]string, argc)
		for i := int32(0); i < argc; i++ {
			args[i] = r.ReadString()
		}
		body, err := f.walk(r)
		if err != nil {
			return nil, err
		}
		return &ast.FuncDeclaration{
			Name: name,
			Args: args,
			Body: body,
		}, nil

	default:
		return nil, errors.Errorf("unknown declaration byte, expected %d or %d, found %d", DEC_LET, DEC_FUNC, declType)
	}
}

func (f *flags) readBlockV2(r *BytesReader) (*ast.BlockV2, error) {
	rs, err := f.deserializeDeclaration(r)
	if err != nil {
		return nil, err
	}

	body, err := f.walk(r)
	if err != nil {
		return nil, err
	}

	return &ast.BlockV2{
		Decl: rs,
		Body: body,
	}, nil
}

func (f *flags) readArray(r *BytesReader) (*ast.ArrayExpr, error) {
	cnt := r.ReadInt()
	items := make([]ast.Expr, cnt)
	for i := 0; i < int(cnt); i++ {
		item, err := f.walk(r)
		if err != nil {
			return nil, err
		}
		switch item.(type) {
		case *ast.LongExpr, *ast.BooleanExpr, *ast.StringExpr, *ast.BytesExpr:
			items[i] = item
		default:
			return nil, errors.New("unsupported type of array item")
		}
	}
	return ast.NewArray(items), nil
}

func (f *flags) readFuncCAll(iter *BytesReader) (*ast.FuncCallExpr, error) {
	nativeOrUser, err := iter.ReadByte()
	if err != nil {
		return nil, err
	}
	switch nativeOrUser {
	case FH_NATIVE:
		f, err := f.readNativeFunction(iter)
		if err != nil {
			return nil, err
		}
		return ast.NewFuncCall(f), nil
	case FH_USER:
		f, err := f.readUserFunction(iter)
		if err != nil {
			return nil, err
		}
		return ast.NewFuncCall(f), nil
	default:
		return nil, errors.Errorf("invalid function type, expects 0 or 1, found %d", nativeOrUser)
	}

}

func (f *flags) readNativeFunction(iter *BytesReader) (*ast.FunctionCall, error) {
	funcNumber := iter.ReadShort()
	name := strconv.Itoa(int(funcNumber))
	argc := iter.ReadInt()
	argv := make([]ast.Expr, argc)

	for i := int32(0); i < argc; i++ {
		v, err := f.walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}
	return ast.NewFunctionCall(name, argv), nil
}

func (f *flags) readUserFunction(iter *BytesReader) (*ast.FunctionCall, error) {
	name := iter.ReadString()
	argc := iter.ReadInt()
	argv := make([]ast.Expr, argc)
	for i := int32(0); i < argc; i++ {
		v, err := f.walk(iter)
		if err != nil {
			return nil, err
		}
		argv[i] = v
	}

	return ast.NewFunctionCall(name, argv), nil
}

func (f *flags) readIf(r *BytesReader) (*ast.IfExpr, error) {
	cond, err := f.walk(r)
	if err != nil {
		return nil, err
	}
	True, err := f.walk(r)
	if err != nil {
		return nil, err
	}
	False, err := f.walk(r)
	if err != nil {
		return nil, err
	}
	return ast.NewIf(cond, True, False), nil
}

func readBytes(r *BytesReader) (*ast.BytesExpr, error) {
	return ast.NewBytes(r.ReadBytes()), nil
}

func (f *flags) readGetter(r *BytesReader) (*ast.GetterExpr, error) {
	a, err := f.walk(r)
	if err != nil {
		return nil, err
	}

	s := r.ReadString()
	return ast.NewGetterExpr(a, s), nil
}
