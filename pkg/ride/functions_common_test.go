package ride

import (
	stderrs "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

func TestGetType(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{rideInt(5)}, false, rideString("Int")},
		{[]rideType{rideString("xxx")}, false, rideString("String")},
		{[]rideType{rideBoolean(true)}, false, rideString("Boolean")},
		{[]rideType{tuple2{el1: rideString("xxx"), el2: rideInt(123)}}, false, rideString("(String, Int)")},
		{[]rideType{rideList{rideString("xxx"), rideInt(123)}}, false, rideString("List[Any]")},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := getType(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestSizeTuple(t *testing.T) {
	for _, test := range []struct {
		args []rideType
		fail bool
		r    rideType
	}{
		{[]rideType{tuple2{}}, false, rideInt(2)},
		{[]rideType{tuple3{}}, false, rideInt(3)},
		{[]rideType{tuple4{}}, false, rideInt(4)},
		{[]rideType{tuple5{}}, false, rideInt(5)},
		{[]rideType{tuple6{}}, false, rideInt(6)},
		{[]rideType{tuple7{}}, false, rideInt(7)},
		{[]rideType{tuple8{}}, false, rideInt(8)},
		{[]rideType{tuple9{}}, false, rideInt(9)},
		{[]rideType{tuple10{}}, false, rideInt(10)},
		{[]rideType{tuple11{}}, false, rideInt(11)},
		{[]rideType{tuple12{}}, false, rideInt(12)},
		{[]rideType{tuple13{}}, false, rideInt(13)},
		{[]rideType{tuple14{}}, false, rideInt(14)},
		{[]rideType{tuple15{}}, false, rideInt(15)},
		{[]rideType{tuple16{}}, false, rideInt(16)},
		{[]rideType{tuple17{}}, false, rideInt(17)},
		{[]rideType{tuple18{}}, false, rideInt(18)},
		{[]rideType{tuple19{}}, false, rideInt(19)},
		{[]rideType{tuple20{}}, false, rideInt(20)},
		{[]rideType{tuple21{}}, false, rideInt(21)},
		{[]rideType{tuple22{}}, false, rideInt(22)},
		{[]rideType{rideString("xxx")}, true, nil},
		{[]rideType{rideBoolean(true)}, true, nil},
		{[]rideType{rideList{rideString("xxx"), rideInt(123)}}, true, nil},
		{[]rideType{rideInt(1), rideUnit{}}, true, nil},
		{[]rideType{rideInt(1), rideInt(2), rideInt(3)}, true, nil},
		{[]rideType{}, true, nil},
	} {
		r, err := sizeTuple(nil, test.args...)
		if test.fail {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.r, r)
		}
	}
}

func TestThrowComplexities(t *testing.T) {
	makeEnv := func(v ast.LibraryVersion, rideV6Activated bool) *testEnv {
		const complexityLimit = 10
		te := newTestEnv(t).
			withProtobufTx().
			withLibVersion(v).
			withComplexityLimit(complexityLimit)
		if rideV6Activated {
			return te.withRideV6Activated()
		}
		return te
	}

	t.Run("without-message", func(t *testing.T) {
		const scriptTmpl = `
			{-# STDLIB_VERSION %d #-}
			{-# CONTENT_TYPE EXPRESSION #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			if (true) then throw() else true`
		tests := []struct {
			libV       ast.LibraryVersion
			rideV6     bool
			complexity int
		}{
			{ast.LibV1, false, 2},
			{ast.LibV2, false, 2},
			{ast.LibV3, false, 2},
			{ast.LibV4, false, 2},
			{ast.LibV5, false, 2},
			{ast.LibV6, false, 2},
			{ast.LibV7, false, 2},
			{ast.LibV8, false, 2},
			{ast.LibV9, false, 2},
			{ast.LibV1, true, 3},
			{ast.LibV2, true, 3},
			{ast.LibV3, true, 2},
			{ast.LibV4, true, 2},
			{ast.LibV5, true, 2},
			{ast.LibV6, true, 2},
			{ast.LibV7, true, 2},
			{ast.LibV8, true, 2},
			{ast.LibV9, true, 2},
		}
		for _, tc := range tests {
			tree, errs := ridec.CompileToTree(fmt.Sprintf(scriptTmpl, tc.libV))
			require.NoError(t, stderrs.Join(errs...))
			require.Equal(t, tc.libV, tree.LibVersion)
			t.Run(fmt.Sprintf("libV%d-rideV6=%t", tc.libV, tc.rideV6), func(t *testing.T) {
				env := makeEnv(tc.libV, tc.rideV6).toEnv()
				_, err := CallVerifier(env, tree)
				require.EqualError(t, err, defaultThrowMessage)
				assert.Equal(t, tc.complexity, env.complexityCalculator().complexity())
				assert.Equal(t, tc.complexity, EvaluationErrorSpentComplexity(err))
			})
		}
	})

	t.Run("with-message", func(t *testing.T) {
		const scriptTmpl = `
			{-# STDLIB_VERSION %d #-}
			{-# CONTENT_TYPE EXPRESSION #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			if (true) then throw("foo-bar-baz") else true`
		tests := []struct {
			libV       ast.LibraryVersion
			rideV6     bool
			complexity int
		}{
			{ast.LibV1, false, 2},
			{ast.LibV2, false, 2},
			{ast.LibV3, false, 2},
			{ast.LibV4, false, 2},
			{ast.LibV5, false, 2},
			{ast.LibV6, false, 2},
			{ast.LibV7, false, 2},
			{ast.LibV8, false, 2},
			{ast.LibV9, false, 2},
			{ast.LibV1, true, 1},
			{ast.LibV2, true, 1},
			{ast.LibV3, true, 1},
			{ast.LibV4, true, 1},
			{ast.LibV5, true, 1},
			{ast.LibV6, true, 1},
			{ast.LibV7, true, 1},
			{ast.LibV8, true, 1},
			{ast.LibV9, true, 1},
		}
		for _, tc := range tests {
			tree, errs := ridec.CompileToTree(fmt.Sprintf(scriptTmpl, tc.libV))
			require.NoError(t, stderrs.Join(errs...))
			require.Equal(t, tc.libV, tree.LibVersion)
			t.Run(fmt.Sprintf("libV%d-rideV6=%t", tc.libV, tc.rideV6), func(t *testing.T) {
				env := makeEnv(tc.libV, tc.rideV6).toEnv()
				_, err := CallVerifier(env, tree)
				require.EqualError(t, err, "foo-bar-baz")
				assert.Equal(t, tc.complexity, env.complexityCalculator().complexity())
				assert.Equal(t, tc.complexity, EvaluationErrorSpentComplexity(err))
			})
		}
	})
}

func TestCommonFunctionsArgumentErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		fn   rideFunction
		args []rideType
		err  string
	}{
		{"eq", eq, []rideType{rideInt(1)}, "eq: 1 is invalid number of arguments, expected 2"},
		{"neq", neq, []rideType{rideInt(1), nil}, "neq: argument 2 is empty"},
		{"instanceOf/count", instanceOf, []rideType{rideInt(1)},
			"instanceOf: 1 is invalid number of arguments, expected 2"},
		{"instanceOf/type", instanceOf, []rideType{rideInt(1), rideInt(2)},
			"instanceOf: second argument is not a String value but 'Int'"},
		{"extract", extract, nil, "extract: 0 is invalid number of arguments, expected 1"},
		{"isDefined", isDefined, []rideType{rideInt(1), rideInt(2)},
			"isDefined: 2 is invalid number of arguments, expected 1"},
		{"value", value, nil, "value: 0 is invalid number of arguments, expected 1"},
		{"valueOrErrorMessage/count", valueOrErrorMessage, []rideType{rideInt(1)},
			"valueOrErrorMessage: 1 is invalid number of arguments, expected 2"},
		{"valueOrErrorMessage/type", valueOrErrorMessage, []rideType{rideInt(1), rideInt(2)},
			"valueOrErrorMessage: unexpected argument type '2'"},
		{"valueOrElse", valueOrElse, []rideType{rideInt(1)},
			"valueOrErrorMessage: 1 is invalid number of arguments, expected 2"},
		{"throw", throw, []rideType{rideInt(1)}, "throw: argument 1 is not of type 'String' but 'Int'"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, err := test.fn(nil, test.args...)
			assert.EqualError(t, err, test.err)
			assert.Nil(t, r)
		})
	}
}

func TestCommonFunctionsUnitValues(t *testing.T) {
	_, err := extract(nil, rideUnit{})
	assert.EqualError(t, err, "extract() called on unit value")
	assert.Equal(t, UserError, GetEvaluationErrorType(err))
	_, err = value(nil, rideUnit{})
	assert.EqualError(t, err, defaultThrowMessage)
	_, err = valueOrErrorMessage(nil, rideUnit{}, rideString("custom"))
	assert.EqualError(t, err, "custom")
	r, err := valueOrElse(nil, rideUnit{}, rideInt(7))
	require.NoError(t, err)
	assert.Equal(t, rideInt(7), r)
	r, err = isDefined(nil, rideUnit{})
	require.NoError(t, err)
	assert.Equal(t, rideBoolean(false), r)
}
