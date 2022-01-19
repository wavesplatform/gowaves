package ride

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	nativeFoldTestState = &MockSmartState{}
	nativeFoldTestEnv   = &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		stateFunc: func() types.SmartState {
			return nativeFoldTestState
		},
		libVersionFunc: func() int {
			return 5
		},
		rideV6ActivatedFunc: func() bool {
			return true
		},
	}
)

func evaluateFold(t *testing.T, code string) {
	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	res, err := CallVerifier(nativeFoldTestEnv, tree)
	require.NoError(t, err)
	r, ok := res.(ScriptResult)
	require.True(t, ok)
	require.True(t, r.res)
}

func TestNativeFoldEvaluationCorrectLeftFold(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func sum(a: String, b: Int) = "(" + a + "+" + toString(b) + ")"

		fold_20([1,2,3,4,5,6,7,8,9,10,11,12,13], "0", sum) == "(((((((((((((0+1)+2)+3)+4)+5)+6)+7)+8)+9)+10)+11)+12)+13)"
	*/
	code := "BgEKAQNzdW0CAWEBYgkArAICCQCsAgIJAKwCAgkArAICAgEoBQFhAgErCQCkAwEFAWICASkJAAACCQDCAwMJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHCQDMCAIACAkAzAgCAAkJAMwIAgAKCQDMCAIACwkAzAgCAAwJAMwIAgANBQNuaWwCATACA3N1bQI5KCgoKCgoKCgoKCgoKDArMSkrMikrMykrNCkrNSkrNikrNykrOCkrOSkrMTApKzExKSsxMikrMTMpW4xQtQ=="
	evaluateFold(t, code)
}

func TestNativeFoldSum(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func sum(a: Int, b: Int) = a + b

		fold_20([1,2,3,4,5,6,7,8,9,10,11,12,13], 0, sum) == 91
	*/
	code := "BgEKAQNzdW0CAWEBYgkAZAIFAWEFAWIJAAACCQDCAwMJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHCQDMCAIACAkAzAgCAAkJAMwIAgAKCQDMCAIACwkAzAgCAAwJAMwIAgANBQNuaWwAAAIDc3VtAFtN86UP"
	evaluateFold(t, code)
}

func TestNativeFoldFilter(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func filter(a: List[Int], b: Int) = if b % 2 == 0 then a ++ [b] else a
		fold_20([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13], [], filter) == [2, 4, 6, 8, 10, 12]
	*/
	code := "BgEKAQZmaWx0ZXICAWEBYgMJAAACCQBqAgUBYgACAAAJAM4IAgUBYQkAzAgCBQFiBQNuaWwFAWEJAAACCQDCAwMJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHCQDMCAIACAkAzAgCAAkJAMwIAgAKCQDMCAIACwkAzAgCAAwJAMwIAgANBQNuaWwFA25pbAIGZmlsdGVyCQDMCAIAAgkAzAgCAAQJAMwIAgAGCQDMCAIACAkAzAgCAAoJAMwIAgAMBQNuaWyxw0Yw"
	evaluateFold(t, code)
}

func TestNativeFoldExampleSum(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func sum(accum: Int, next: Int) = accum + next
		let arr = [1,2,3,4,5]
		fold_20(arr, 0, sum) == 15
	*/
	code := "BgEKAQNzdW0CBWFjY3VtBG5leHQJAGQCBQVhY2N1bQUEbmV4dAQDYXJyCQDMCAIAAQkAzAgCAAIJAMwIAgADCQDMCAIABAkAzAgCAAUFA25pbAkAAAIJAMIDAwUDYXJyAAACA3N1bQAPYO5AYA=="
	evaluateFold(t, code)
}

func TestNativeFoldExampleProduct(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func mult(accum: Int, next: Int) = accum * next
		let arr = [1,2,3,4,5]
		fold_20(arr, 1, mult) == 120
	*/
	code := "BgEKAQRtdWx0AgVhY2N1bQRuZXh0CQBoAgUFYWNjdW0FBG5leHQEA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgABAgRtdWx0AHhsgJ2A"
	evaluateFold(t, code)
}

func TestNativeFoldExampleFilter(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func filterEven(accum: List[Int], next: Int) = if (next % 2 == 0) then accum :+ next else accum
		let arr = [1,2,3,4,5]
		fold_20(arr, [], filterEven) == [2, 4]
	*/
	code := "BgEKAQpmaWx0ZXJFdmVuAgVhY2N1bQRuZXh0AwkAAAIJAGoCBQRuZXh0AAIAAAkAzQgCBQVhY2N1bQUEbmV4dAUFYWNjdW0EA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgUDbmlsAgpmaWx0ZXJFdmVuCQDMCAIAAgkAzAgCAAQFA25pbDcomcQ="
	evaluateFold(t, code)
}

func TestNativeFoldExampleMap(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func map(accum: List[Int], next: Int) = (next - 1) :: accum
		let arr = [1, 2, 3, 4, 5]
		fold_20(arr, [], map) == [4, 3, 2, 1, 0]
	*/
	code := "BgEKAQNtYXACBWFjY3VtBG5leHQJAMwIAgkAZQIFBG5leHQAAQUFYWNjdW0EA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgUDbmlsAgNtYXAJAMwIAgAECQDMCAIAAwkAzAgCAAIJAMwIAgABCQDMCAIAAAUDbmlsg4VsgA=="
	evaluateFold(t, code)
}

func TestFoldFunctionOverlap(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		let a = 4
		func g(b: Int) = a
		func f(x: Int , a: Int) = x + g(a)
		let arr = [1,2,3,4,5]
		fold_20(arr, 0, f) == 20
	*/
	code := "BgEEAWEABAoBAWcBAWIFAWEKAQFmAgF4AWEJAGQCBQF4CQEBZwEFAWEEA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgAAAgFmABSQ7ChM"
	evaluateFold(t, code)
}

func TestFoldNestedFunctions(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func f() = {
		    func f() = {
		        func f() = {1}
		        f()
		    }
		    f()
		}
		func s(x: Int , a: Int) = x + f()
		let arr = [1,2,3,4,5]
		fold_20(arr, 0, s) == 5
	*/
	code := "BgEKAQFmAAoBAWYACgEBZgAAAQkBAWYACQEBZgAKAQFzAgF4AWEJAGQCBQF4CQEBZgAEA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgAAAgFzAAWhEymR"
	evaluateFold(t, code)
}

func TestNestedFolds(t *testing.T) {
	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func f(a: Int, n: Int) = a + n
		func f1(a: Int, n: List[Int]) = {
			a + fold_20(n, 0, f)
		}
		let arr = [[1, 2, 3], [1, 2, 3], [1, 2, 3]]
		fold_20(arr, 0, f1) == 18
	*/
	code := "BgEKAQFmAgFhAW4JAGQCBQFhBQFuCgECZjECAWEBbgkAZAIFAWEJAMIDAwUBbgAAAgFmBANhcnIJAMwIAgkAzAgCAAEJAMwIAgACCQDMCAIAAwUDbmlsCQDMCAIJAMwIAgABCQDMCAIAAgkAzAgCAAMFA25pbAkAzAgCCQDMCAIAAQkAzAgCAAIJAMwIAgADBQNuaWwFA25pbAkAAAIJAMIDAwUDYXJyAAACAmYxABJAge10"
	evaluateFold(t, code)

	/*
		{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}
		func f1(a: Int, n: List[Int]) = {
		    func f(a: Int, n: Int) = a + n
		    a + fold_20(n, 0, f)
		}
		let arr = [[1, 2, 3], [1, 2, 3], [1, 2, 3]]
		fold_20(arr, 0, f1) == 18
	*/
	code = "BgEKAQJmMQIBYQFuCgEBZgIBYQFuCQBkAgUBYQUBbgkAZAIFAWEJAMIDAwUBbgAAAgFmBANhcnIJAMwIAgkAzAgCAAEJAMwIAgACCQDMCAIAAwUDbmlsCQDMCAIJAMwIAgABCQDMCAIAAgkAzAgCAAMFA25pbAkAzAgCCQDMCAIAAQkAzAgCAAIJAMwIAgADBQNuaWwFA25pbAkAAAIJAMIDAwUDYXJyAAACAmYxABJ/1dDi"
	evaluateFold(t, code)
}

func TestEvaluateInvalidNativeFoldCall(t *testing.T) {
	code := "BgEKAQNzdW0CAWEBbgkAZAIFAWEFAW4EA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwJAAACCQDCAwMFA2FycgAAAAAAD55COAs="

	src, err := base64.StdEncoding.DecodeString(code)
	require.NoError(t, err)

	tree, err := Parse(src)
	require.NoError(t, err)
	assert.NotNil(t, tree)

	_, err = CallVerifier(nativeFoldTestEnv, tree)
	require.Error(t, err)
}
