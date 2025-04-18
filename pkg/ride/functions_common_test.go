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
			{ast.LibV1, true, 3},
			{ast.LibV2, true, 3},
			{ast.LibV3, true, 2},
			{ast.LibV4, true, 2},
			{ast.LibV5, true, 2},
			{ast.LibV6, true, 2},
			{ast.LibV7, true, 2},
			{ast.LibV8, true, 2},
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
			{ast.LibV1, true, 1},
			{ast.LibV2, true, 1},
			{ast.LibV3, true, 1},
			{ast.LibV4, true, 1},
			{ast.LibV5, true, 1},
			{ast.LibV6, true, 1},
			{ast.LibV7, true, 1},
			{ast.LibV8, true, 1},
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
