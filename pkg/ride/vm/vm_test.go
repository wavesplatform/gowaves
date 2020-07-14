package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/ride/op"
	"github.com/wavesplatform/gowaves/pkg/ride/transpiler"
)

//type State struct {
//}
//
//func p(err error) {
//	if err != nil {
//		panic(err)
//	}
//}
//
//const (
//	LONG = iota + 1
//	STRING
//	BOOL
//	REFERENCE
//	LIST
//	OBJECT
//
//	CALL = iota + 1
//	SET
//	SETR // set result
//	GET
//	POPINT
//	PUSH
//	PUSHR // push reference on stack
//	PUSHL // push long on stack
//	MOV   // MOVE R1 S("bla") - move to register R1 string value
//	CMP   // R1 < R2 = -1, R1 == R2 = 0, R1 > R2 = 1
//
//	RETR // set result value from reference
//
//	R1 = iota + 1
//	R2
//	R3
//	R4
//)

//type value struct {
//	t int
//	s string
//	l int64
//	b bool
//	//r  string // reference
//	vs []value
//}

//type Stack interface {
//	Pop()
//}

//type Stack struct {
//	index int
//	stack []value
//}
//
//func (a *Stack) PushInt(i int64) {
//
//}
//func (a *Stack) Push(i interface{}) {
//
//}
//
//func (a *Stack) PopInt() int64 {
//	return 0
//}
//
//// func of 2 args
//func eqInt(stack *Stack) {
//	i1 := stack.PopInt()
//	i2 := stack.PopInt()
//	stack.Push(i1 == i2)
//}

//var eqF = []cmd{
//	{POPINT, nil, nil},
//	{},
//}
//
//type cmd struct {
//	c int
//	v value
//	//i1 value
//	//i2 value
//}

//func S(s string, vs ...value) value {
//	return value{
//		t:  STRING,
//		s:  s,
//		vs: vs,
//	}
//}

//
//func L(l int64) value {
//	return value{
//		t: LONG,
//		l: l,
//	}
//}
//
//func R(r string) value { //reference
//	return value{
//		t: REFERENCE,
//		s: r,
//	}
//}
//
//func B(b bool) value { //reference
//	return value{
//		t: BOOL,
//		b: b,
//	}
//}

//func VS(vs []value) value { //reference
//	return value{
//		t:  LIST,
//		vs: vs,
//	}
//}

//type Function func(scope *Scope) error

//func TestMainain(t *testing.T) {
//	rs, err := reader.NewReaderFromBase64("AwQAAAACeDACAAAABHFxcXEEAAAAAngxCQAETAAAAAIFAAAAAngwCQAETAAAAAIFAAAAAngwCQAETAAAAAIFAAAAAngwBQAAAANuaWwEAAAAAngyCQAETAAAAAIFAAAAAngxCQAETAAAAAIFAAAAAngxCQAETAAAAAIFAAAAAngxBQAAAANuaWwEAAAAAngzCQAETAAAAAIFAAAAAngyCQAETAAAAAIFAAAAAngyCQAETAAAAAIFAAAAAngyBQAAAANuaWwEAAAAAng0CQAETAAAAAIFAAAAAngzCQAETAAAAAIFAAAAAngzCQAETAAAAAIFAAAAAngzBQAAAANuaWwEAAAAAng1CQAETAAAAAIFAAAAAng0CQAETAAAAAIFAAAAAng0CQAETAAAAAIFAAAAAng0BQAAAANuaWwEAAAAAng2CQAETAAAAAIFAAAAAng1CQAETAAAAAIFAAAAAng1CQAETAAAAAIFAAAAAng1BQAAAANuaWwEAAAAAng3CQAETAAAAAIFAAAAAng2CQAETAAAAAIFAAAAAng2CQAETAAAAAIFAAAAAng2BQAAAANuaWwEAAAAAng4CQAETAAAAAIFAAAAAng3CQAETAAAAAIFAAAAAng3CQAETAAAAAIFAAAAAng3BQAAAANuaWwEAAAAAng5CQAETAAAAAIFAAAAAng4CQAETAAAAAIFAAAAAng4CQAETAAAAAIFAAAAAng4BQAAAANuaWwEAAAAA3gxMAkABEwAAAACBQAAAAJ4OQkABEwAAAACBQAAAAJ4OQkABEwAAAACBQAAAAJ4OQUAAAADbmlsBAAAAAN4MTEJAARMAAAAAgUAAAADeDEwCQAETAAAAAIFAAAAA3gxMAkABEwAAAACBQAAAAN4MTAFAAAAA25pbAQAAAADeDEyCQAETAAAAAIFAAAAA3gxMQkABEwAAAACBQAAAAN4MTEJAARMAAAAAgUAAAADeDExBQAAAANuaWwEAAAAA3gxMwkABEwAAAACBQAAAAN4MTIJAARMAAAAAgUAAAADeDEyCQAETAAAAAIFAAAAA3gxMgUAAAADbmlsBAAAAAN4MTQJAARMAAAAAgUAAAADeDEzCQAETAAAAAIFAAAAA3gxMwkABEwAAAACBQAAAAN4MTMFAAAAA25pbAQAAAADeDE1CQAETAAAAAIFAAAAA3gxNAkABEwAAAACBQAAAAN4MTQJAARMAAAAAgUAAAADeDE0BQAAAANuaWwEAAAAA3gxNgkABEwAAAACBQAAAAN4MTUJAARMAAAAAgUAAAADeDE1CQAETAAAAAIFAAAAA3gxNQUAAAADbmlsBAAAAAN4MTcJAARMAAAAAgUAAAADeDE2CQAETAAAAAIFAAAAA3gxNgkABEwAAAACBQAAAAN4MTYFAAAAA25pbAQAAAADeDE4CQAETAAAAAIFAAAAA3gxNwkABEwAAAACBQAAAAN4MTcJAARMAAAAAgUAAAADeDE3BQAAAANuaWwEAAAAA3gxOQkABEwAAAACBQAAAAN4MTgJAARMAAAAAgUAAAADeDE4CQAETAAAAAIFAAAAA3gxOAUAAAADbmlsBAAAAAN4MjAJAARMAAAAAgUAAAADeDE5CQAETAAAAAIFAAAAA3gxOQkABEwAAAACBQAAAAN4MTkFAAAAA25pbAQAAAADeDIxCQAETAAAAAIFAAAAA3gyMAkABEwAAAACBQAAAAN4MjAJAARMAAAAAgUAAAADeDIwBQAAAANuaWwEAAAAA3gyMgkABEwAAAACBQAAAAN4MjEJAARMAAAAAgUAAAADeDIxCQAETAAAAAIFAAAAA3gyMQUAAAADbmlsBAAAAAJ5MAIAAAAEcXFxcQQAAAACeTEJAARMAAAAAgUAAAACeTAJAARMAAAAAgUAAAACeTAJAARMAAAAAgUAAAACeTAFAAAAA25pbAQAAAACeTIJAARMAAAAAgUAAAACeTEJAARMAAAAAgUAAAACeTEJAARMAAAAAgUAAAACeTEFAAAAA25pbAQAAAACeTMJAARMAAAAAgUAAAACeTIJAARMAAAAAgUAAAACeTIJAARMAAAAAgUAAAACeTIFAAAAA25pbAQAAAACeTQJAARMAAAAAgUAAAACeTMJAARMAAAAAgUAAAACeTMJAARMAAAAAgUAAAACeTMFAAAAA25pbAQAAAACeTUJAARMAAAAAgUAAAACeTQJAARMAAAAAgUAAAACeTQJAARMAAAAAgUAAAACeTQFAAAAA25pbAQAAAACeTYJAARMAAAAAgUAAAACeTUJAARMAAAAAgUAAAACeTUJAARMAAAAAgUAAAACeTUFAAAAA25pbAQAAAACeTcJAARMAAAAAgUAAAACeTYJAARMAAAAAgUAAAACeTYJAARMAAAAAgUAAAACeTYFAAAAA25pbAQAAAACeTgJAARMAAAAAgUAAAACeTcJAARMAAAAAgUAAAACeTcJAARMAAAAAgUAAAACeTcFAAAAA25pbAQAAAACeTkJAARMAAAAAgUAAAACeTgJAARMAAAAAgUAAAACeTgJAARMAAAAAgUAAAACeTgFAAAAA25pbAQAAAADeTEwCQAETAAAAAIFAAAAAnk5CQAETAAAAAIFAAAAAnk5CQAETAAAAAIFAAAAAnk5BQAAAANuaWwEAAAAA3kxMQkABEwAAAACBQAAAAN5MTAJAARMAAAAAgUAAAADeTEwCQAETAAAAAIFAAAAA3kxMAUAAAADbmlsBAAAAAN5MTIJAARMAAAAAgUAAAADeTExCQAETAAAAAIFAAAAA3kxMQkABEwAAAACBQAAAAN5MTEFAAAAA25pbAQAAAADeTEzCQAETAAAAAIFAAAAA3kxMgkABEwAAAACBQAAAAN5MTIJAARMAAAAAgUAAAADeTEyBQAAAANuaWwEAAAAA3kxNAkABEwAAAACBQAAAAN5MTMJAARMAAAAAgUAAAADeTEzCQAETAAAAAIFAAAAA3kxMwUAAAADbmlsBAAAAAN5MTUJAARMAAAAAgUAAAADeTE0CQAETAAAAAIFAAAAA3kxNAkABEwAAAACBQAAAAN5MTQFAAAAA25pbAQAAAADeTE2CQAETAAAAAIFAAAAA3kxNQkABEwAAAACBQAAAAN5MTUJAARMAAAAAgUAAAADeTE1BQAAAANuaWwEAAAAA3kxNwkABEwAAAACBQAAAAN5MTYJAARMAAAAAgUAAAADeTE2CQAETAAAAAIFAAAAA3kxNgUAAAADbmlsBAAAAAN5MTgJAARMAAAAAgUAAAADeTE3CQAETAAAAAIFAAAAA3kxNwkABEwAAAACBQAAAAN5MTcFAAAAA25pbAQAAAADeTE5CQAETAAAAAIFAAAAA3kxOAkABEwAAAACBQAAAAN5MTgJAARMAAAAAgUAAAADeTE4BQAAAANuaWwEAAAAA3kyMAkABEwAAAACBQAAAAN5MTkJAARMAAAAAgUAAAADeTE5CQAETAAAAAIFAAAAA3kxOQUAAAADbmlsBAAAAAN5MjEJAARMAAAAAgUAAAADeTIwCQAETAAAAAIFAAAAA3kyMAkABEwAAAACBQAAAAN5MjAFAAAAA25pbAQAAAADeTIyCQAETAAAAAIFAAAAA3kyMQkABEwAAAACBQAAAAN5MjEJAARMAAAAAgUAAAADeTIxBQAAAANuaWwJAAAAAAAAAgUAAAADeDE3BQAAAAN5MTfU8+3M")
//	//rs, err := reader.NewReaderFromBase64("AgQAAAABeAAAAAAAAAAABQQAAAABZQAAAAAAAAAABgkAAAAAAAACBQAAAAF4BQAAAAFlVE38Hw==")
//	p(err)
//	scr, err := ast.BuildScript(rs)
//	p(err)
//
//	scope := ast.NewScope(3, 'E', nil)
//	ev, err := scr.Eval(scope)
//	p(err)
//
//	fmt.Println(ev)
//
//}
//
//type Scope struct {
//	values map[string]value
//	stack  []value
//	fns    map[string]Function
//	Result value
//}
//
//func NewScope() *Scope {
//	return &Scope{
//		values: make(map[string]value),
//		fns:    make(map[string]Function),
//	}
//}

//
////go:inline
//func (a *Scope) Set(name string, v value) {
//	a.values[name] = v
//}
//
//func (a *Scope) SetF(name string, f Function) {
//	a.fns[name] = f
//}
//
//func (a *Scope) Call(name string) error {
//	f, ok := a.fns[name]
//	if !ok {
//		return errors.Errorf("no function named %s", name)
//	}
//	err := f(a)
//	if err != nil {
//		return errors.Wrap(err, "Scope.Call failed func call "+name)
//	}
//	return nil
//}

////go:inline
//func (a *Scope) Push(v value) {
//	a.stack = append(a.stack, v)
//}
//
//func (a *Scope) Pop() (value, error) {
//	out := a.stack[len(a.stack)-1]
//	a.stack = a.stack[:len(a.stack)-1]
//	return out, nil
//}

//func eval(s *Scope, cmds []cmd) error {
//	for _, cmd := range cmds {
//		switch cmd.c {
//		case SET:
//			s.Set(cmd.v.s, cmd.v.vs[0])
//		case PUSHR:
//			s.Push(R(cmd.v.s))
//		case PUSHL: // push long
//			s.Push(cmd.v)
//		case CALL:
//			err := s.Call(cmd.v.s)
//			if err != nil {
//				return err
//			}
//		case SETR:
//			s.Set(cmd.v.s, s.Result)
//		default:
//			return errors.Errorf("unknown cmd %d", cmd.c)
//		}
//	}
//	return nil
//}

/*
let x = "string"
func RemoveUnderscoreIfPresent(remaining: String) = if ((size(remaining) > 0)) then drop(remaining, 1) else remaining;

func ParseNextAttribute (remaining: String) = {
	let s = size(remaining)
	if (s > 0)
		then {
			let nn = parseIntValue(take(remaining, 2))
						let v = take(drop(remaining, 2), nn)
			let tmpRemaining = drop(remaining, (nn + 2))
			let remainingState = RemoveUnderscoreIfPresent(tmpRemaining)
			[v, remainingState]

			}
		else throw("Empty string was passed into parseNextAttribute func")
}

*/

//{REG, 0, },

//{PUSH, S()},

//{DEC, L(100500)},
//{POP, L(0)},
//{CMPL, },

//{PUSH},

//{CMP_LONG,

// let x = 5; 6 >= x
//func TestVmRun(t *testing.T) {
//	a := NewOpCodeBuilder()
//	a.LabelS("x")
//	a.RememberShift()
//	a.StackPushL(5)
//	a.Ret()
//	a.ApplyShift()
//	a.StackPushL(6)
//	a.JmpRefS("x")
//	a.CallS("103") // gte
//
//	cd := a.Code()
//
//	fncs := map[string]Func{
//		"103": GteLong,
//	}
//
//	scope := NewScope(fncs, nil, 'I')
//	ok, err := EvaluateExpressionAsBoolean(cd, scope)
//	require.NoError(t, err)
//	require.Equal(t, true, ok)
//}

func defaultScope() *Scope {
	fns := map[string]Func{
		"1100": NativeCreateList,
		"401":  NativeGetList,
		"103":  GteLong,
		"0":    Eq,
	}
	return NewScope(fns, map[string]ast.Expr{"nil": ast.NewExprs()}, 'I')
}

func TestVm_PutStringOnStack(t *testing.T) {
	a := op.NewOpCodeBuilder()
	a.StackPushS([]byte("x"))

	ok, err := EvaluateExpression(a.Code(), nil)
	require.NoError(t, err)
	require.Equal(t, ast.NewString("x"), ok)
}

/**
`let x = 5; 6 >= x`
*/
func BenchmarkVmRun(b *testing.B) {
	b.ReportAllocs()
	a := op.NewOpCodeBuilder()
	//a.LabelS("x")
	//a.StackPushL(5)
	//a.Ret()
	//a.StackPushL(6)
	//a.JmpRefS("x")
	//a.CallS("103") // gte

	m := transpiler.NewInitial(a)
	m.BlockV1([]byte("x")).
		Long(5).
		Call([]byte("103"), 2).
		Long(5).
		Ref([]byte{'x'})

	fncs := map[string]Func{
		"103": GteLong,
	}
	scope := NewScope(fncs, nil, 'I')

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ok, err := EvaluateExpressionAsBoolean(a.Code(), scope)
		require.NoError(b, err)
		require.Equal(b, true, ok)
	}
}

/**
`let x = 5; 6 >= x`
*/
//func BenchmarkTreeRun(b *testing.B) {
//	b.ReportAllocs()
//	r, err := reader.NewReaderFromBase64("AQQAAAABeAAAAAAAAAAABQkAAGcAAAACAAAAAAAAAAAGBQAAAAF4jlxXHA==")
//	fail(err)
//
//	script, err := messages.BuildScript(r)
//	fail(err)
//
//	scope := ast.NewScope(1, 'I', nil)
//
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		rs, err := script.Verifier.Evaluate(scope)
//		require.NoError(b, err)
//		require.Equal(b, ast.NewBoolean(true), rs)
//	}
//
//}

/**
if true then true else false
*/
func TestMatchSimpleCase(t *testing.T) {
	b := op.NewOpCodeBuilder()
	v := transpiler.NewInitial(b)
	v = v.If()
	v = v.Bool(true)
	v = v.Bool(true)
	v = v.Bool(false)

	ok, err := EvaluateExpressionAsBoolean(b.Code(), NewScope(nil, nil, 'I'))
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

/*
[1][0] == 1
*/
func TestCons(t *testing.T) {
	r, _ := reader.NewReaderFromBase64(`AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQUAAAADbmlsAAAAAAAAAAAAAAAAAAAAAAAB5EKjUA==`)
	code := op.NewOpCodeBuilder()
	err := transpiler.BuildCode(r, transpiler.NewInitial(code))
	require.NoError(t, err)

	ok, err := EvaluateExpressionAsBoolean(code.Code(), defaultScope())
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

/**
[1, 2, 3, 4, 5][4] == 5
*/

func TestMultipleCons(t *testing.T) {
	b64 := `AwkAAAAAAAACCQABkQAAAAIJAARMAAAAAgAAAAAAAAAAAQkABEwAAAACAAAAAAAAAAACCQAETAAAAAIAAAAAAAAAAAMJAARMAAAAAgAAAAAAAAAABAkABEwAAAACAAAAAAAAAAAFBQAAAANuaWwAAAAAAAAAAAQAAAAAAAAAAAVrPjYC`
	r, _ := reader.NewReaderFromBase64(b64)
	code := op.NewOpCodeBuilder()
	err := transpiler.BuildCode(r, transpiler.NewInitial(code))
	require.NoError(t, err)

	ok, err := EvaluateExpressionAsBoolean(code.Code(), defaultScope())
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

/**

 */
//func Test2222(t *testing.T) {
//	b64 := `AQQAAAAOZm91bmRlcjFQdWJLZXkBAAAAIMh2i+XT/MY1+/hrJrtIus2QMa38/Df+fQWT+JwvK4Q5BAAAAA5mb3VuZGVyMlB1YktleQEAAAAgg6NsQVNVutIFVA3Q6BRpc+bVCixXnfxkjf4ooj/eZUkEAAAADmZvdW5kZXIzUHViS2V5AQAAACA4JJ1ewAlCaiDno1cNUoIn8BzVQ/fh4t0WLAsIYHePAgQAAAAOZm91bmRlcjRQdWJLZXkBAAAAIIXi/L19m8YAAL7Ugn+xSrpwdokGb3oU8nYLuW/yHkEDBAAAAA5mb3VuZGVyNVB1YktleQEAAAAg0nVdPtSbtsN2OmK/1IX+yN0J8Ff5tr+50TEcMoZci38EAAAADmZvdW5kZXIxU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAAFAAAADmZvdW5kZXIxUHViS2V5AAAAAAAAAAABAAAAAAAAAAAABAAAAA5mb3VuZGVyMlNpZ25lZAMJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAABBQAAAA5mb3VuZGVyMlB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAOZm91bmRlcjNTaWduZWQDCQAB9AAAAAMIBQAAAAJ0eAAAAAlib2R5Qnl0ZXMJAAGRAAAAAggFAAAAAnR4AAAABnByb29mcwAAAAAAAAAAAgUAAAAOZm91bmRlcjNQdWJLZXkAAAAAAAAAAAEAAAAAAAAAAAAEAAAADmZvdW5kZXI0U2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAMFAAAADmZvdW5kZXI0UHViS2V5AAAAAAAAAAABAAAAAAAAAAAABAAAAA5mb3VuZGVyNVNpZ25lZAMJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAEBQAAAA5mb3VuZGVyNVB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAkAAGcAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAADmZvdW5kZXIxU2lnbmVkBQAAAA5mb3VuZGVyMlNpZ25lZAUAAAAOZm91bmRlcjNTaWduZWQFAAAADmZvdW5kZXI0U2lnbmVkBQAAAA5mb3VuZGVyNVNpZ25lZAAAAAAAAAAAAh52YSo=`
//	r, _ := reader.NewReaderFromBase64(b64)
//	code := op.NewOpCodeBuilder()
//	err := transpiler.BuildCode(r, transpiler.NewInitial(code))
//	require.NoError(t, err)
//
//	ok, err := EvaluateExpressionAsBoolean(code.Code(), defaultScope())
//	require.NoError(t, err)
//	require.Equal(t, true, ok)
//}

//func Test222(t *testing.T) {
//	x := []byte{1, 4, 0, 0, 0, 5, 112, 75, 101, 121, 48, 1, 0, 0, 0, 32, 139, 156, 188, 105, 36, 49, 157, 232, 35, 93, 74, 58, 2, 2, 20, 180, 220, 172, 216, 239, 216, 49, 24, 235, 209, 248, 21, 50, 30, 113, 60, 69, 4, 0, 0, 0, 5, 112, 75, 101, 121, 49, 1, 0, 0, 0, 32, 136, 194, 176, 221, 33, 193, 126, 39, 31, 18, 42, 194, 241, 210, 179, 65, 245, 146, 6, 241, 229, 173, 11, 254, 121, 119, 248, 63, 231, 108, 128, 69, 4, 0, 0, 0, 11, 112, 75, 101, 121, 48, 83, 105, 103, 110, 101, 100, 3, 9, 0, 1, 244, 0, 0, 0, 3, 8, 5, 0, 0, 0, 2, 116, 120, 0, 0, 0, 9, 98, 111, 100, 121, 66, 121, 116, 101, 115, 9, 0, 1, 145, 0, 0, 0, 2, 8, 5, 0, 0, 0, 2, 116, 120, 0, 0, 0, 6, 112, 114, 111, 111, 102, 115, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 5, 112, 75, 101, 121, 48, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 11, 112, 75, 101, 121, 49, 83, 105, 103, 110, 101, 100, 3, 9, 0, 1, 244, 0, 0, 0, 3, 8, 5, 0, 0, 0, 2, 116, 120, 0, 0, 0, 9, 98, 111, 100, 121, 66, 121, 116, 101, 115, 9, 0, 1, 145, 0, 0, 0, 2, 8, 5, 0, 0, 0, 2, 116, 120, 0, 0, 0, 6, 112, 114, 111, 111, 102, 115, 0, 0, 0, 0, 0, 0, 0, 0, 1, 5, 0, 0, 0, 5, 112, 75, 101, 121, 49, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 103, 0, 0, 0, 2, 9, 0, 0, 100, 0, 0, 0, 2, 5, 0, 0, 0, 11, 112, 75, 101, 121, 48, 83, 105, 103, 110, 101, 100, 5, 0, 0, 0, 11, 112, 75, 101, 121, 49, 83, 105, 103, 110, 101, 100, 0, 0, 0, 0, 0, 0, 0, 0, 1}
//
//	//_ = x
//
//	b := op.NewOpCodeBuilder()
//	err := transpiler.BuildCode(reader.NewBytesReader(x), transpiler.NewInitial(b))
//	fmt.Println(err)
//}
