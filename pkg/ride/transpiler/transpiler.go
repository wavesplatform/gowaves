package transpiler

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

func BuildCode(r *reader.BytesReader, a Fsm) error {
	r.Next()

	f := flags{
		a: a,
	}
	return f.walk(r)

	//for !r.Eof() {
	//	n := r.Next()
	//	switch n {
	//	case reader.E_LONG:
	//		a = a.Long(r.ReadLong())
	//	case reader.E_BYTES:
	//		a = a.Bytes(r.ReadBytes())
	//	case reader.E_STRING:
	//		a = a.String(r.ReadByteString())
	//	case reader.E_IF:
	//		a = a.If()
	//	case reader.E_BLOCK:
	//		s := r.ReadByteString()
	//		a = a.BlockV1(s)
	//	case reader.E_REF: // 5
	//		a = a.Ref(r.ReadByteString())
	//	case reader.E_TRUE:
	//		a = a.Bool(true)
	//	case reader.E_FALSE:
	//		a = a.Bool(false)
	//	case reader.E_GETTER:
	//		next := r.Peek()
	//		if next == reader.E_REF {
	//			r.Next()
	//			ref := r.ReadByteString()
	//			attr := r.ReadByteString()
	//			a = a.Call([]byte("$getter"), 2).Ref(ref).String(attr)
	//			continue
	//		}
	//		if next == reader.E_FUNCALL {
	//			return errors.Errorf("E_GETTER: unsupported operation with function on first argument")
	//		}
	//		return errors.Errorf("expected reader.E_REF %d, found %d", reader.E_REF, next)
	//	case reader.E_FUNCALL:
	//		nativeOrUser, err := r.ReadByte()
	//		if err != nil {
	//			return errors.Wrap(err, "reader.E_FUNCALL: reading native or user")
	//		}
	//		switch nativeOrUser {
	//		case reader.FH_NATIVE:
	//			a = readNativeFunction(a, r)
	//		case reader.FH_USER:
	//			a = readUserFunction(a, r)
	//		default:
	//			return errors.Errorf("invalid function type, expects 0 or 1, found %d", nativeOrUser)
	//		}
	//	default:
	//		fmt.Printf("unknown code %+v, pos: %d %v", n, r.Pos(), r.Content())
	//		//return nil, errors.Errorf("unknown code %d", n)
	//		return errors.Errorf("unknown code %d, at pos %d", n, r.Pos())
	//	}
	//}
	//return nil
}

type flags struct {
	blockV2 bool
	arrays  bool
	a       Fsm
}

func (f *flags) walk(r *reader.BytesReader) error {
	if r.Eof() {
		return reader.ErrUnexpectedEOF
	}

	next := r.Next()
	switch next {
	case reader.E_LONG:
		return f.readLong(r)
	case reader.E_BYTES:
		return f.readBytes(r)
	case reader.E_STRING:
		return f.readString(r)
		//return ast.NewString(iter.ReadString()), nil
	case reader.E_IF:
		return f.readIf(r)
	case reader.E_BLOCK:
		return f.readBlock(r)
	case reader.E_REF:
		return f.readRef(r)
		//return &ast.RefExpr{
		//	Name: iter.ReadString(),
		//}, nil
	case reader.E_TRUE:
		return f.readBool(r, true)
	case reader.E_FALSE:
		return f.readBool(r, false)
	case reader.E_GETTER:
		return f.readGetter(r)
	case reader.E_FUNCALL:
		return f.readFuncCAll(r)
	//case E_BLOCK_V2:
	//	b, err := f.readBlockV2(iter)
	//	if err != nil {
	//		return nil, err
	//	}
	//	f.blockV2 = true
	//	return b, nil
	//case E_ARR:
	//	a, err := f.readArray(iter)
	//	if err != nil {
	//		return nil, err
	//	}
	//	f.arrays = true
	//	return a, nil
	default:
		return errors.Errorf("invalid byte %d", next)
	}
}

func readNativeFunction(fsm Fsm, iter *reader.BytesReader) Fsm {
	funcNumber := iter.ReadShort()
	name := strconv.Itoa(int(funcNumber))
	argc := iter.ReadInt()
	return fsm.Call([]byte(name), argc)
}

func readUserFunction(fsm Fsm, iter *reader.BytesReader) Fsm {
	name := iter.ReadByteString()
	argc := iter.ReadInt()
	return fsm.Call(name, argc)
}

func (f *flags) readBlock(r *reader.BytesReader) error {
	letName := r.ReadByteString()
	f.a = f.a.BlockV1(letName)
	err := f.walk(r)
	if err != nil {
		return err
	}

	err = f.walk(r)
	if err != nil {
		return err
	}
	return nil

	//return &ast.Block{
	//	Let:  ast.NewLet(letName, letValue),
	//	Body: body,
	//}, nil
}

/*
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
*/
func (f *flags) readFuncCAll(iter *reader.BytesReader) error {
	//f.a = f.a.Call()
	nativeOrUser, err := iter.ReadByte()
	if err != nil {
		return err
	}
	switch nativeOrUser {
	case reader.FH_NATIVE:
		err := f.readNativeFunction(iter)
		if err != nil {
			return err
		}
		return nil
	case reader.FH_USER:
		err := f.readUserFunction(iter)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.Errorf("invalid function type, expects 0 or 1, found %d", nativeOrUser)
	}

}

func (f *flags) readNativeFunction(iter *reader.BytesReader) error {
	funcNumber := iter.ReadShort()
	name := strconv.Itoa(int(funcNumber))
	argc := iter.ReadInt()
	//argv := make([]ast.Expr, argc)
	f.a = f.a.Call([]byte(name), argc)

	for i := int32(0); i < argc; i++ {
		err := f.walk(iter)
		if err != nil {
			return err
		}
		//argv[i] = v
	}
	return nil
}

func (f *flags) readUserFunction(iter *reader.BytesReader) error {
	name := iter.ReadByteString()
	argc := iter.ReadInt()
	f.a = f.a.Call(name, argc)
	for i := int32(0); i < argc; i++ {
		err := f.walk(iter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *flags) readIf(r *reader.BytesReader) error {
	f.a = f.a.If()
	err := f.walk(r)
	if err != nil {
		return err
	}
	err = f.walk(r)
	if err != nil {
		return err
	}
	err = f.walk(r)
	if err != nil {
		return err
	}
	return nil
}

func (f *flags) readGetter(r *reader.BytesReader) error {
	f.a = f.a.Call([]byte("$getter"), 2)
	err := f.walk(r)
	if err != nil {
		return err
	}

	s := r.ReadByteString()
	f.a = f.a.String(s)
	return nil
}

func (f *flags) readBytes(r *reader.BytesReader) error {
	f.a = f.a.Bytes(r.ReadBytes())
	return nil
}

func (f *flags) readString(r *reader.BytesReader) error {
	f.a = f.a.String(r.ReadByteString())
	return nil
}

func (f *flags) readRef(r *reader.BytesReader) error {
	f.a = f.a.Ref(r.ReadByteString())
	return nil
}

func (f *flags) readLong(r *reader.BytesReader) error {
	f.a = f.a.Long(r.ReadLong())
	return nil
}

func (f *flags) readBool(r *reader.BytesReader, b bool) error {
	f.a = f.a.Bool(b)
	return nil
}
