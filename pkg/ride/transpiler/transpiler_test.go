package transpiler

import (
	"testing"

	"github.com/stretchr/testify/require"
	//"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	op "github.com/wavesplatform/gowaves/pkg/ride/op"
	op2 "github.com/wavesplatform/gowaves/pkg/ride/op"
)

type testBuilder struct {
	base64 string
	//s      *vm.Scope
}

//func (a *testBuilder) WithScope(s *vm.Scope) *testBuilder {
//	a.s = s
//	return a
//}

func (a *testBuilder) WithBase64(base64 string) *testBuilder {
	a.base64 = base64
	return a
}

//func (a *testBuilder) Expect(t *testing.T, value interface{}) {
//	r, err := reader.NewReaderFromBase64(a.base64)
//	require.NoError(t, err)
//
//	code := op2.NewOpCodeBuilder()
//	err = BuildCode(r, NewInitial(code))
//	require.NoError(t, err)
//
//	ok, err := vm.EvaluateExpressionAsBoolean(code.Code(), a.s)
//	require.NoError(t, err)
//	require.Equal(t, value, ok)
//}

func From64(base64 string) *testBuilder {
	return &testBuilder{
		base64: base64,
	}
}

//func WithScope(scope *vm.Scope) *testBuilder {
//	return &testBuilder{
//		s: scope,
//	}
//}

//type scope map[string]vm.Func
//type F map[string]vm.Func

//type C map[string]ast.Expr

//func defaultScope() *vm.Scope {
//	fncs := F{
//		"0":                 vm.Eq,
//		"1":                 vm.IsInstanceOf,
//		"!=":                vm.Neq,
//		"$getter":           vm.GetterFn,
//		"addressFromString": vm.UserAddressFromString,
//	}
//	return vm.NewScope(fncs, C{"tx": ast.NewObject(map[string]ast.Expr{})}, 'I')
//}

func TestSingleValue(t *testing.T) {
	b := op2.NewOpCodeBuilder()
	NewInitial(b).
		Bool(true)

	require.Equal(t, []byte{op.StackPushTrue, op.Ret}, b.Code())
}

/*
let x = true
false
*/
func TestBlockV1(t *testing.T) {
	b := op2.NewOpCodeBuilder()
	NewInitial(b).
		BlockV1([]byte("x")).
		Bool(true).
		Bool(false)

	require.Equal(t, []byte{
		op2.Label, 0, 1, 'x', 0, 0, 0, 0xA /*shift*/, op2.StackPushTrue, op2.Ret,
		op2.StackPushFalse,
		op.Ret}, b.Code())
}

func TestCall(t *testing.T) {
	op := op2.NewOpCodeBuilder()
	NewInitial(op).
		Call([]byte{'$'}, 2).
		Bool(true).
		Bool(true)

	require.Equal(t, []byte{
		op2.StackPushTrue, op2.StackPushTrue, op2.Call, 0, 1, '$',
		op2.Ret,
	}, op.Code())
}

/*
let x = $(true, true)
true
*/
func TestBlockWithFunCall(t *testing.T) {
	op := op2.NewOpCodeBuilder()
	NewInitial(op).
		BlockV1([]byte{'x'}).
		Call([]byte{'$'}, 2).
		Bool(true).
		Bool(true).
		Bool(true)

	require.Equal(t, []byte{
		op2.Label, 0, 1, 'x',
		0, 0, 0, 0xA + 5,
		op2.StackPushTrue, op2.StackPushTrue, op2.Call, 0, 1, '$', op2.Ret,
		op2.StackPushTrue,
		op2.Ret,
	}, op.Code())
}

/*
match tx {
  case _: TransferTransaction => true
  case _ => false
}
*/
//func TestMatchSimpleCase(t *testing.T) {
//	base64 := "AwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24GBwmqVVU="
//	r, err := reader.NewReaderFromBase64(base64)
//	require.NoError(t, err)
//
//	code := op2.NewOpCodeBuilder()
//	err = BuildCode(r, NewInitial(code))
//	require.NoError(t, err)
//	ok, err := vm.EvaluateExpressionAsBoolean(code.Code(), defaultScope())
//	require.NoError(t, err)
//	require.Equal(t, false, ok)
//}

/*
	if true then true else false
*/
func Test_If(t *testing.T) {
	b := op.NewOpCodeBuilder()
	v := NewInitial(b)
	v = v.If()
	v = v.Bool(true)
	v = v.Bool(true)
	v = v.Bool(false)

	require.Equal(t,
		[]byte{op2.StackPushTrue, op2.JumpIfNot, 0, 0, 0, 0xC,
			/*true branch*/ op2.StackPushTrue, op2.Jmp, 0, 0, 0, 0xD,
			/*false branch*/ op2.StackPushFalse,
			op.Ret},
		b.Code())
}

// AwQAAAABeAQAAAABeQAAAAAAAAAABAUAAAABeQkAAAAAAAACBQAAAAF4AAAAAAAAAAAEwTszeQ==
/**
let x = {
    let y = true
    y
}
true
*/

func TestLetInLet(t *testing.T) {
	b := op.NewOpCodeBuilder()
	v := NewInitial(b)
	v = v.BlockV1([]byte("x"))
	v = v.BlockV1([]byte("y"))
	v = v.Bool(true)
	v = v.Ref([]byte("y"))
	v = v.Bool(true)

	require.Equal(t, []byte{
		/*label*/ op.Label, 0, 1, 'x', 0, 0, 0, 23,
		/*value*/ op.Label, 0, 1, 'y', 0, 0, 0, 18 /*value2*/, op.StackPushTrue, op.Ret, op.JmpRef, 0, 1, 'y', op.Ret,
		op.StackPushTrue, op.Ret,
	},
		b.Code())

}

func TestEmptyFunctions(t *testing.T) {
	// let x = false
	//if x then throw() else !true
	//AQQAAAABeAcDBQAAAAF4CQEAAAAFdGhyb3cAAAAACQEAAAABIQAAAAEGLvqQKg==
	b := op.NewOpCodeBuilder()
	v := NewInitial(b)
	v = v.Call([]byte("throw"), 0)

	require.Equal(t, []byte{
		/*call*/ op.Call, 0, 5, 't', 'h', 'r', 'o', 'w',
		/*ret*/ op.Ret,
		///*value*/ op.Label, 0, 1, 'y', 0, 0, 0, 18 /*value2*/, op.StackPushTrue, op.Ret, op.JmpRef, 0, 1, 'y', op.Ret,
		//op.StackPushTrue, op.Ret,
	},
		b.Code())

}
