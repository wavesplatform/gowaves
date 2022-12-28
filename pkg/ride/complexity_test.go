package ride

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func checkVerifierSpentComplexity(t *testing.T, env environment, code string, complexity int, comment string) {
	_, tree := parseBase64Script(t, code)
	r, err := CallVerifier(env, tree)
	require.NoError(t, err, comment)
	require.NotNil(t, r, comment)
	assert.True(t, r.Result(), comment)
	assert.Equal(t, complexity, r.Complexity(), comment)
}

func checkVerifierSpentComplexityV5(t *testing.T, code string, complexity int, comment string) {
	env := newTestEnv(t).withLibVersion(ast.LibV5).withComplexityLimit(ast.LibV5, 2000).toEnv()
	checkVerifierSpentComplexity(t, env, code, complexity, comment)
}

func checkVerifierSpentComplexityV6(t *testing.T, code string, complexity int, comment string) {
	env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(ast.LibV6, 2000).withRideV6Activated().toEnv()
	checkVerifierSpentComplexity(t, env, code, complexity, comment)
}

func checkFunctionCallComplexity(t *testing.T, env environment, code, fn string, fa proto.Arguments, complexity int) {
	_, tree := parseBase64Script(t, code)
	r, err := CallFunction(env, tree, fn, fa)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, complexity, r.Complexity())
}

func checkFunctionCallComplexityV5(t *testing.T, code, fn string, fa proto.Arguments, complexity int) {
	env := newTestEnv(t).withLibVersion(ast.LibV5).withComplexityLimit(ast.LibV5, 2000).toEnv()
	checkFunctionCallComplexity(t, env, code, fn, fa, complexity)
}

func checkFunctionCallComplexityV6(t *testing.T, code, fn string, fa proto.Arguments, complexity int) {
	env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(ast.LibV6, 2000).withRideV6Activated().toEnv()
	checkFunctionCallComplexity(t, env, code, fn, fa, complexity)
}

func TestSimpleScriptsComplexity(t *testing.T) {
	for _, test := range []struct {
		comment    string
		source     string
		complexity int
	}{
		{`V4: let a = 1 + 10 + 100; let b = 1000 + a + 10000; let c = a + b + 100000; c + a == 111333`, "BAQAAAABYQkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAoAAAAAAAAAAGQEAAAAAWIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAPoBQAAAAFhAAAAAAAAACcQBAAAAAFjCQAAZAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAGGoAkAAAAAAAACCQAAZAAAAAIFAAAAAWMFAAAAAWEAAAAAAAABsuVAqr8m", 13},
		{`V4: func f(a: Int, b: Int) = {let c = a + b; let d = a - b; c * d - 1}; f(1, 2) == -4`, "BAoBAAAAAWYAAAACAAAAAWEAAAABYgQAAAABYwkAAGQAAAACBQAAAAFhBQAAAAFiBAAAAAFkCQAAZQAAAAIFAAAAAWEFAAAAAWIJAABlAAAAAgkAAGgAAAACBQAAAAFjBQAAAAFkAAAAAAAAAAABCQAAAAAAAAIJAQAAAAFmAAAAAgAAAAAAAAAAAQAAAAAAAAAAAgD//////////Pwcs2o=", 11},
		{`V4:  let x = 1 + 1 + 1; let a = 1 + 1; func f(a: Int, b: Int) = a - b + x; let b = 4; func g(a: Int, b: Int) = a * b; let expected = (a - b + x) * (b - a + x); let actual = g(f(a, b), f(b, a)); actual == expected && actual == expected && x == 3 && a == 2 && b == 4`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEEAAAAAWEJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWYAAAACAAAAAWEAAAABYgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgEAAAAAWIAAAAAAAAAAAQKAQAAAAFnAAAAAgAAAAFhAAAAAWIJAABoAAAAAgUAAAABYQUAAAABYgQAAAAIZXhwZWN0ZWQJAABoAAAAAgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgJAABkAAAAAgkAAGUAAAACBQAAAAFiBQAAAAFhBQAAAAF4BAAAAAZhY3R1YWwJAQAAAAFnAAAAAgkBAAAAAWYAAAACBQAAAAFhBQAAAAFiCQEAAAABZgAAAAIFAAAAAWIFAAAAAWEDAwMDCQAAAAAAAAIFAAAABmFjdHVhbAUAAAAIZXhwZWN0ZWQJAAAAAAAAAgUAAAAGYWN0dWFsBQAAAAhleHBlY3RlZAcJAAAAAAAAAgUAAAABeAAAAAAAAAAAAwcJAAAAAAAAAgUAAAABYQAAAAAAAAAAAgcJAAAAAAAAAgUAAAABYgAAAAAAAAAABAd/cU2j", 47},
		{`V4:  let x = 1 + 1 + 1 + 1 + 1; let y = x + 1; func f(x: Int) = x + 1; func g(x: Int) = x + 1 + 1; func h(x: Int) = x + 1 + 1 + 1; f(g(h(y))) == x + x + 2`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABBAAAAAF5CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFmAAAAAQAAAAF4CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFnAAAAAQAAAAF4CQAAZAAAAAIJAABkAAAAAgUAAAABeAAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWgAAAABAAAAAXgJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgkBAAAAAWYAAAABCQEAAAABZwAAAAEJAQAAAAFoAAAAAQUAAAABeQkAAGQAAAACCQAAZAAAAAIFAAAAAXgFAAAAAXgAAAAAAAAAAAJBsCoy", 21},
		{`V4:  let x = 1 + 1 + 1; let y = {let z = 1; z}; y + x == 4`, "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEEAAAAAXkEAAAAAXoAAAAAAAAAAAEFAAAAAXoJAAAAAAAAAgkAAGQAAAACBQAAAAF5BQAAAAF4AAAAAAAAAAAECwZAYA==", 7},
		{`V4:  let address = Address(base58'aaaa'); address.bytes == base58'aaaa'`, "BAQAAAAHYWRkcmVzcwkBAAAAB0FkZHJlc3MAAAABAQAAAANj+GcJAAAAAAAAAggFAAAAB2FkZHJlc3MAAAAFYnl0ZXMBAAAAA2P4Z/7QEyM=", 3},
		{`V4:  let x = (1, 2, 3); x._2 == 2`, "BAQAAAABeAkABRUAAAADAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADCQAAAAAAAAIIBQAAAAF4AAAAAl8yAAAAAAAAAAACXAdyJg==", 4},
		{`V4:  let a = 1 + 1; let b = a; func g() = {let a1 = 2 + 2 + 2; let c = a1; c + b + a1}; g() + a == 16`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZwAAAAAEAAAAAmExCQAAZAAAAAIJAABkAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgQAAAABYwUAAAACYTEJAABkAAAAAgkAAGQAAAACBQAAAAFjBQAAAAFiBQAAAAJhMQkAAAAAAAACCQAAZAAAAAIJAQAAAAFnAAAAAAUAAAABYQAAAAAAAAAAEGbAh7s=", 13},
		{`V4:  let a = 1 + 1; let b = a; func g() = {let c = 2 + 2; let d = c; d + b + c}; g() + a == 12`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZwAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgQAAAABZAUAAAABYwkAAGQAAAACCQAAZAAAAAIFAAAAAWQFAAAAAWIFAAAAAWMJAAAAAAAAAgkAAGQAAAACCQEAAAABZwAAAAAFAAAAAWEAAAAAAAAAAAxjZ1Td", 12},
		{`V4:  let a = 1 + 1; let b = a; func f() = {let c = 1 + 1; c + b}; a + f() == 6`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQkAAGQAAAACBQAAAAFjBQAAAAFiCQAAAAAAAAIJAABkAAAAAgUAAAABYQkBAAAAAWYAAAAAAAAAAAAAAAAGZR1Q1A==", 9},
		{`V4:  let a = 1 + 1; let b = a; func f() = {let c = 1 + 1; c + b}; f() + a == 6`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAEAAAAAWMJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQkAAGQAAAACBQAAAAFjBQAAAAFiCQAAAAAAAAIJAABkAAAAAgkBAAAAAWYAAAAABQAAAAFhAAAAAAAAAAAGeznbzA==", 9},
		{`V4: let a = 1 + 1; let b = a; a + b == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCQAAAAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAAABClbyII=", 6},
		{`V4: let a = 1 + 1; let b = a; b + a == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCQAAAAAAAAIJAABkAAAAAgUAAAABYgUAAAABYQAAAAAAAAAABApVv5E=", 6},
		{`V4: let a = 1 + 1; let b = a; func f() = b; a + f() == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAFAAAAAWIJAAAAAAAAAgkAAGQAAAACBQAAAAFhCQEAAAABZgAAAAAAAAAAAAAAAASZ9mVe", 6},
		{`V4: let a = 1 + 1; let b = a; func f() = b; f() + a == 4`, "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZgAAAAAFAAAAAWIJAAAAAAAAAgkAAGQAAAACCQEAAAABZgAAAAAFAAAAAWEAAAAAAAAAAASvoK6u", 6},
	} {
		_, tree := parseBase64Script(t, test.source)

		env := newTestEnv(t).withLibVersion(ast.LibV4).withComplexityLimit(ast.LibV4, 2000).
			withTransaction(testTransferWithProofs(t)).toEnv()
		res, err := CallVerifier(env, tree)
		require.NoError(t, err, test.comment)
		require.NotNil(t, res, test.comment)

		r, ok := res.(ScriptResult)
		assert.True(t, ok, test.comment)
		assert.Equal(t, test.complexity, r.Complexity(), test.comment)
	}
}

func TestMultipleLets(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let a = 1 + 10 + 100
		let b = 1000 + a + 10000
		let c = a + b + 100000
		c + a == 111333
	*/
	code := "BAQAAAABYQkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAoAAAAAAAAAAGQEAAAAAWIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAPoBQAAAAFhAAAAAAAAACcQBAAAAAFjCQAAZAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAGGoAkAAAAAAAACCQAAZAAAAAIFAAAAAWMFAAAAAWEAAAAAAAABsuVAqr8m"
	checkVerifierSpentComplexityV5(t, code, 13, "")
	checkVerifierSpentComplexityV6(t, code, 8, "")
}

func TestUserFunction(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func f(a: Int, b: Int) = {
		    let c = a + b
		    let d = a - b
		    c * d - 1
		}
		f(1, 2) == -4
	*/
	code := "BAoBAAAAAWYAAAACAAAAAWEAAAABYgQAAAABYwkAAGQAAAACBQAAAAFhBQAAAAFiBAAAAAFkCQAAZQAAAAIFAAAAAWEFAAAAAWIJAABlAAAAAgkAAGgAAAACBQAAAAFjBQAAAAFkAAAAAAAAAAABCQAAAAAAAAIJAQAAAAFmAAAAAgAAAAAAAAAAAQAAAAAAAAAAAgD//////////Pwcs2o="
	checkVerifierSpentComplexityV5(t, code, 11, "")
	checkVerifierSpentComplexityV6(t, code, 5, "")
}

func TestMultipleUserFunctionsAndRefs(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let x = 1 + 1 + 1                          # 2 (should be calculated once)
		let a = 1 + 1                              # 1 (should be calculated once)
		func f(a: Int, b: Int) = a - b + x         # 5
		let b = 4                                  #
		func g(a: Int, b: Int) = a * b             # 3
		let expected = (a - b + x) * (b - a + x)   # 11
		let actual = g(f(a, b), f(b, a))           # 3 + 5 * 2 + 4 = 17
		actual == expected &&                      # 11 + 17 + 4 = 32
		actual == expected &&                      # 4
		x == 3             &&                      # 3
		a == 2             &&                      # 3
		b == 4                                     # 2  Total: 32 + 4 + 3 + 3 + 2 + 2 (x value) + 1 (a value) = 47
	*/
	code := "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEEAAAAAWEJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWYAAAACAAAAAWEAAAABYgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgEAAAAAWIAAAAAAAAAAAQKAQAAAAFnAAAAAgAAAAFhAAAAAWIJAABoAAAAAgUAAAABYQUAAAABYgQAAAAIZXhwZWN0ZWQJAABoAAAAAgkAAGQAAAACCQAAZQAAAAIFAAAAAWEFAAAAAWIFAAAAAXgJAABkAAAAAgkAAGUAAAACBQAAAAFiBQAAAAFhBQAAAAF4BAAAAAZhY3R1YWwJAQAAAAFnAAAAAgkBAAAAAWYAAAACBQAAAAFhBQAAAAFiCQEAAAABZgAAAAIFAAAAAWIFAAAAAWEDAwMDCQAAAAAAAAIFAAAABmFjdHVhbAUAAAAIZXhwZWN0ZWQJAAAAAAAAAgUAAAAGYWN0dWFsBQAAAAhleHBlY3RlZAcJAAAAAAAAAgUAAAABeAAAAAAAAAAAAwcJAAAAAAAAAgUAAAABYQAAAAAAAAAAAgcJAAAAAAAAAgUAAAABYgAAAAAAAAAABAd/cU2j"
	checkVerifierSpentComplexityV5(t, code, 47, "")
	checkVerifierSpentComplexityV6(t, code, 18, "")
}

func TestLetOverlapThroughFunctionParam(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let x = 1 + 1 + 1 + 1 + 1         # 4
		let y = x + 1                     # 2
		func f(x: Int) = x + 1            # 2
		func g(x: Int) = x + 1 + 1        # 3
		func h(x: Int) = x + 1 + 1 + 1    # 4
		f(g(h(y))) == x + x + 2           # Total: 2 (f) + 3 (g) + 4(h) + 1 (y ref) + 2 (y value) + 1 (==) + 2 (2 x ref) + 4 (x value) + 2 (2 +)
	*/
	code := "BAQAAAABeAkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABBAAAAAF5CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFmAAAAAQAAAAF4CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEKAQAAAAFnAAAAAQAAAAF4CQAAZAAAAAIJAABkAAAAAgUAAAABeAAAAAAAAAAAAQAAAAAAAAAAAQoBAAAAAWgAAAABAAAAAXgJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgkBAAAAAWYAAAABCQEAAAABZwAAAAEJAQAAAAFoAAAAAQUAAAABeQkAAGQAAAACCQAAZAAAAAIFAAAAAXgFAAAAAXgAAAAAAAAAAAJBsCoy"
	checkVerifierSpentComplexityV5(t, code, 21, "")
	checkVerifierSpentComplexityV6(t, code, 14, "")
}

func TestLetOverlapInsideLetValueBlock(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		6 == if (2 > 1) then 1 + 2 + 3 else 3 + 4
	*/
	code := "BAkAAAAAAAACAAAAAAAAAAAGAwkAAGYAAAACAAAAAAAAAAACAAAAAAAAAAABCQAAZAAAAAIJAABkAAAAAgAAAAAAAAAAAQAAAAAAAAAAAgAAAAAAAAAAAwkAAGQAAAACAAAAAAAAAAADAAAAAAAAAAAEc4rTAQ=="
	checkVerifierSpentComplexityV5(t, code, 5, "")
	checkVerifierSpentComplexityV6(t, code, 4, "")
}

func TestGetterComplexity(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let address = Address(base58'aaaa')
		address.bytes == base58'aaaa'
	*/
	code := "BAQAAAAHYWRkcmVzcwkBAAAAB0FkZHJlc3MAAAABAQAAAANj+GcJAAAAAAAAAggFAAAAB2FkZHJlc3MAAAAFYnl0ZXMBAAAAA2P4Z/7QEyM="
	checkVerifierSpentComplexityV5(t, code, 3, "")
	checkVerifierSpentComplexityV6(t, code, 2, "")
}

func TestLetContextComplexity(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		let a = 1 + 1               # 1 (once)
		let b = a                   # 1 (once)
		func g() = {
		    let a1 = 2 + 2 + 2      # 2 (once)
		    let c = a1              # 1 (once)
		    c + b + a1              # 5
		}

		g() + a == 16               # Total: 13
	*/
	code := "BAQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABBAAAAAFiBQAAAAFhCgEAAAABZwAAAAAEAAAAAmExCQAAZAAAAAIJAABkAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgAAAAAAAAAAAgQAAAABYwUAAAACYTEJAABkAAAAAgkAAGQAAAACBQAAAAFjBQAAAAFiBQAAAAJhMQkAAAAAAAACCQAAZAAAAAIJAQAAAAFnAAAAAAUAAAABYQAAAAAAAAAAEGbAh7s="
	checkVerifierSpentComplexityV5(t, code, 13, "")
	checkVerifierSpentComplexityV6(t, code, 7, "")
}

func TestStrictComplexity(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func testFunc() = {
		  strict a = 100500 + 42
		  a
		}
		testFunc() == 100542
	*/
	code := "BAoBAAAACHRlc3RGdW5jAAAAAAQAAAABYQkAAGQAAAACAAAAAAAAAYiUAAAAAAAAAAAqAwkAAAAAAAACBQAAAAFhBQAAAAFhBQAAAAFhCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkAAAAAAAACCQEAAAAIdGVzdEZ1bmMAAAAAAAAAAAAAAYi+iKfaDQ=="
	checkVerifierSpentComplexityV5(t, code, 7, "")
	checkVerifierSpentComplexityV6(t, code, 3, "")
}

func TestStrictThrow(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func testFunc() = {
		  strict a = throw("Strict executed error")
		  true
		}
		testFunc()
	*/
	code := "BAoBAAAACHRlc3RGdW5jAAAAAAQAAAABYQkAAAIAAAABAgAAABVTdHJpY3QgZXhlY3V0ZWQgZXJyb3IDCQAAAAAAAAIFAAAAAWEFAAAAAWEGCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkBAAAACHRlc3RGdW5jAAAAABn7LqM="
	_, tree := parseBase64Script(t, code)
	_, err := CallVerifier(newTestEnv(t).withComplexityLimit(ast.LibV5, 2000).toEnv(), tree)
	require.Errorf(t, err, "Strict executed error")
}

func TestUnusedStrictComplexity(t *testing.T) {
	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func testFunc() = {
		  let a = 1 + 2 + 3 + 4 + 5 + 5 + 6 + 7 + 8 + 9 + 100500
		  let z = "42"
		  z
		}
		testFunc() == "42"
	*/
	code := "BAoBAAAACHRlc3RGdW5jAAAAAAQAAAABYQkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADAAAAAAAAAAAEAAAAAAAAAAAFAAAAAAAAAAAFAAAAAAAAAAAGAAAAAAAAAAAHAAAAAAAAAAAIAAAAAAAAAAAJAAAAAAAAAYiUBAAAAAF6AgAAAAI0MgUAAAABegkAAAAAAAACCQEAAAAIdGVzdEZ1bmMAAAAAAgAAAAI0MsUmBxw="
	checkVerifierSpentComplexityV5(t, code, 2, "")
	checkVerifierSpentComplexityV6(t, code, 2, "") // Zero complexity user function adds one

	/*
		{-# STDLIB_VERSION 4 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func testFunc() = {
		  strict a = 1 + 2 + 3 + 4 + 5 + 5 + 6 + 7 + 8 + 9 + 100500
		  let z = "42"
		  z
		}
		testFunc() == "42"
	*/
	code = "BAoBAAAACHRlc3RGdW5jAAAAAAQAAAABYQkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADAAAAAAAAAAAEAAAAAAAAAAAFAAAAAAAAAAAFAAAAAAAAAAAGAAAAAAAAAAAHAAAAAAAAAAAIAAAAAAAAAAAJAAAAAAAAAYiUAwkAAAAAAAACBQAAAAFhBQAAAAFhBAAAAAF6AgAAAAI0MgUAAAABegkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4JAAAAAAAAAgkBAAAACHRlc3RGdW5jAAAAAAIAAAACNDLeu/hu"
	checkVerifierSpentComplexityV5(t, code, 16, "")
	checkVerifierSpentComplexityV6(t, code, 12, "")
}

func TestSimpleDAppComplexity1(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func call() = {
		  [BooleanEntry("abc", true)]
		}

		@Verifier(tx)
		func verify() = true
	*/
	code := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAARMAAAAAgkBAAAADEJvb2xlYW5FbnRyeQAAAAICAAAAA2FiYwYFAAAAA25pbAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAGzqWv4w=="
	checkVerifierSpentComplexityV5(t, code, 0, "")
	checkFunctionCallComplexityV5(t, code, "call", proto.Arguments{}, 2)
	checkVerifierSpentComplexityV6(t, code, 0, "")
	checkFunctionCallComplexityV6(t, code, "call", proto.Arguments{}, 2)
}

func TestSimpleDAppComplexity2(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE DAPP #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		@Callable(i)
		func call() = {
			let message = base58'emsY'
			let pub = base58'HnU9jfhpMcQNaG5yQ46eR43RnkWKGxerw2zVrbpnbGof'
			let sig = base58'4uXfw7162zaopAkTNa7eo6YK2mJsTiHGJL3dCtRRH63z1nrdoHBHyhbvrfZovkxf2jKsi2vPsaP2XykfZmUiwPeg'
			[BooleanEntry("abc", sigVerify(message, sig, pub))]
		}
	*/
	code := "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAAB21lc3NhZ2UBAAAAA3B1awQAAAADcHViAQAAACD5YNOXUn1M/ChkrTwitfmd6lTogTr5Kin2wziOrP9OLgQAAAADc2lnAQAAAEDDWY/WuKhHs0AtBSX1V+rNgqDJOKCpqd11SbYDX7PnMl0Cv6DiIppUq8PhqA4/g2y/zgjwe3XOAORP032NFTSBCQAETAAAAAIJAQAAAAxCb29sZWFuRW50cnkAAAACAgAAAANhYmMJAAH0AAAAAwUAAAAHbWVzc2FnZQUAAAADc2lnBQAAAANwdWIFAAAAA25pbAAAAAAu8AfS"
	checkFunctionCallComplexityV5(t, code, "call", proto.Arguments{}, 205)
	checkFunctionCallComplexityV6(t, code, "call", proto.Arguments{}, 202)
}

func TestOnEdgeComplexity1(t *testing.T) {
	/*
		{-#STDLIB_VERSION 6 #-}
		{-#SCRIPT_TYPE ACCOUNT #-}
		{-# CONTENT_TYPE DAPP #-}

		 @Callable(inv)
		 func foo(n: Int) = {
		   let complexInt1 = 1 + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + log(1625, 2, 27, 1, 2, HALFUP) + log(1625, 2, 27, 1, 2, HALFUP) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 # 916
		   # 82 = 1 for "n > 1", 81 for branches (without overflow)
		   let complexInt2 = if (n > 1) then {
		     # 81 = 2 for valueOrElse, 75 for invoke, 1 for Address, 1 for "n - 1", 1 for list, 1 for as
		     valueOrElse(invoke(Address(base58'3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz'), "foo", [n - 1], []).as[Int], 0)
		   } else {
		     1 + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + 1 + 1 + 1 + 1 # 82
		   }
		   # 2 = 1 for tuple, 1 for "+"
		   ([], complexInt1 + complexInt2)
		 }
	*/
	dApp := newTestAccount(t, "DAPP1") // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, tree := parseBase64Script(t, "BgIHCAISAwoBAQABA2ludgEDZm9vAQFuBAtjb21wbGV4SW50MQkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgABCQCgAwEJAHcGCQCnAwECBDE2MjUAAgkApwMBAgIyNwABAAIFBkhBTEZVUAkAoAMBCQB3BgkApwMBAgQxNjI1AAIJAKcDAQICMjcAAQACBQZIQUxGVVAJAG0GANkMAAIAGwABAAIFBkhBTEZVUAkAbQYA2QwAAgAbAAEAAgUGSEFMRlVQCQELdmFsdWVPckVsc2UCCQCfCAECAWsAAAkBC3ZhbHVlT3JFbHNlAgkAnwgBAgFrAAAJAQt2YWx1ZU9yRWxzZQIJAJ8IAQIBawAAAAEAAQABAAEAAQABAAEAAQABAAEAAQQLY29tcGxleEludDIDCQBmAgUBbgABCQELdmFsdWVPckVsc2UCCgABQAkA/AcECQEHQWRkcmVzcwEBGgFUcQ97e0JWLZUBUuI05V2T+HgxB8fHAvABAgNmb28JAMwIAgkAZQIFAW4AAQUDbmlsBQNuaWwDCQABAgUBQAIDSW50BQFABQR1bml0AAAJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCAAEJAQt2YWx1ZU9yRWxzZQIJAJ8IAQIBawAACQELdmFsdWVPckVsc2UCCQCfCAECAWsAAAkBC3ZhbHVlT3JFbHNlAgkAnwgBAgFrAAAJAQt2YWx1ZU9yRWxzZQIJAJ8IAQIBawAACQELdmFsdWVPckVsc2UCCQCfCAECAWsAAAkBC3ZhbHVlT3JFbHNlAgkAnwgBAgFrAAAAAQABAAEAAQkAlAoCBQNuaWwJAGQCBQtjb21wbGV4SW50MQULY29tcGxleEludDIApaxBJw==")

	env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(ast.LibV6, 52000).
		withBlockV5Activated().withProtobufTx().withDataEntriesSizeV2().
		withRideV6Activated().withValidateInternalPayments().withThis(dApp).withDApp(dApp).withSender(dApp).
		withInvocation("foo").withDataEntries(dApp, &proto.IntegerDataEntry{Key: "k", Value: 1}).
		withTree(dApp, tree).withWrappedState().toEnv()

	r, err := CallFunction(env, tree, "foo", proto.Arguments{proto.NewIntegerArgument(52)})
	require.EqualError(t, err, "evaluation complexity 52001 exceeds the limit 52000")
	assert.Equal(t, GetEvaluationErrorType(err), ComplexityLimitExceed)
	assert.Nil(t, r)
	assert.Equal(t, 52000, EvaluationErrorSpentComplexity(err))
}

func TestOnEdgeComplexity2(t *testing.T) {
	/*
		{-#STDLIB_VERSION 6 #-}
		{-#SCRIPT_TYPE ACCOUNT #-}
		{-# CONTENT_TYPE DAPP #-}

		 @Callable(inv)
		 func foo(n: Int) = {
		   let result = if (n > 1) then { # 1
		     let complexInt1 = 1 + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + log(1625, 2, 27, 1, 2, HALFUP) + log(1625, 2, 27, 1, 2, HALFUP) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + valueOrElse(getInteger("k"), 0) + 1
		     if (complexInt1 > 0) # 79 = 1 for ">", 78 for branches
		       # 78 = 75 for invoke, 1 for Address, 1 for list, 1 for "n - 1"
		       then invoke(Address(base58'3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz'), "foo", [n - 1], [])
		       else 0
		   } else {
		     1 + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + toInt(log(parseBigIntValue("1625"), 2, parseBigIntValue("27"), 1, 2, HALFUP)) + 1 + 1 + 1
		   }
		   ([], result) # 1
		 }
	*/
	dApp := newTestAccount(t, "DAPP1") // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	_, tree := parseBase64Script(t, "BgIHCAISAwoBAQABA2ludgEDZm9vAQFuBAZyZXN1bHQDCQBmAgUBbgABBAtjb21wbGV4SW50MQkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIJAGQCCQBkAgABCQCgAwEJAHcGCQCnAwECBDE2MjUAAgkApwMBAgIyNwABAAIFBkhBTEZVUAkAoAMBCQB3BgkApwMBAgQxNjI1AAIJAKcDAQICMjcAAQACBQZIQUxGVVAJAG0GANkMAAIAGwABAAIFBkhBTEZVUAkAbQYA2QwAAgAbAAEAAgUGSEFMRlVQCQELdmFsdWVPckVsc2UCCQCfCAECAWsAAAkBC3ZhbHVlT3JFbHNlAgkAnwgBAgFrAAAJAQt2YWx1ZU9yRWxzZQIJAJ8IAQIBawAACQELdmFsdWVPckVsc2UCCQCfCAECAWsAAAABAwkAZgIFC2NvbXBsZXhJbnQxAAAJAPwHBAkBB0FkZHJlc3MBARoBVHEPe3tCVi2VAVLiNOVdk/h4MQfHxwLwAQIDZm9vCQDMCAIJAGUCBQFuAAEFA25pbAUDbmlsAAAJAGQCCQBkAgkAZAIJAGQCCQBkAgkAZAIAAQkAoAMBCQB3BgkApwMBAgQxNjI1AAIJAKcDAQICMjcAAQACBQZIQUxGVVAJAKADAQkAdwYJAKcDAQIEMTYyNQACCQCnAwECAjI3AAEAAgUGSEFMRlVQCQCgAwEJAHcGCQCnAwECBDE2MjUAAgkApwMBAgIyNwABAAIFBkhBTEZVUAABAAEAAQkAlAoCBQNuaWwFBnJlc3VsdADR0fAN")
	env := newTestEnv(t).withLibVersion(ast.LibV6).withComplexityLimit(ast.LibV6, 52000).
		withBlockV5Activated().withProtobufTx().withDataEntriesSizeV2().
		withRideV6Activated().withValidateInternalPayments().withThis(dApp).withDApp(dApp).withSender(dApp).
		withInvocation("foo").withDataEntries(dApp, &proto.IntegerDataEntry{Key: "k", Value: 1}).
		withTree(dApp, tree).withWrappedState().toEnv()
	r, err := CallFunction(env, tree, "foo", proto.Arguments{proto.NewIntegerArgument(52)})
	require.EqualError(t, err, "evaluation complexity 52001 exceeds the limit 52000")
	assert.Equal(t, GetEvaluationErrorType(err), ComplexityLimitExceed)
	assert.Nil(t, r)
	assert.Equal(t, 52000, EvaluationErrorSpentComplexity(err))
}

func TestComplexities(t *testing.T) {
	for _, test := range []struct {
		comment      string
		source       string
		complexityV5 int
		complexityV6 int
	}{
		{`V5: true`, "BQbtKNoM", 0, 0},
		{`V3: unit == Unit()`, "AwkAAAAAAAACBQAAAAR1bml0CQEAAAAEVW5pdAAAAACd7sMa", 2, 2},
		{`V3: 12345 == 12345`, "AwkAAAAAAAACAAAAAAAAADA5AAAAAAAAADA5+DindQ==", 1, 1},
		{`V3: let x = 2 * 2; x == 4`, "AwQAAAABeAkAAGgAAAACAAAAAAAAAAACAAAAAAAAAAACCQAAAAAAAAIFAAAAAXgAAAAAAAAAAARdrwMC", 3, 2},
		{`V3: let a = "A"; let b = "B"; a + b == "AB"`, "AwQAAAABYQIAAAABQQQAAAABYgIAAAABQgkAAAAAAAACCQABLAAAAAIFAAAAAWEFAAAAAWICAAAAAkFC8C4jQA==", 13, 11},
		{`V3: if true then if true then true else false else false`, "AwMGAwYGBwdYjCji", 2, 0},
		{`V5: let a = {let b = {let c = 1; 0}; 0}; true`, "BQQAAAABYQQAAAABYgQAAAABYwAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAbdLmrq", 0, 0},
		{`toString(Address(base58'3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf')) == "3P3336rNSSU8bDAqDb6S5jNs8DJb2bfNmpf"`, "BQkAAAAAAAACCQAEJQAAAAEJAQAAAAdBZGRyZXNzAAAAAQEAAAAaAVcMIZxOsk2Gw5Avd0ztqi+phtb1Bb83MiQCAAAAIzNQMzMzNnJOU1NVOGJEQXFEYjZTNWpOczhESmIyYmZObXBmA2i8OQ==", 11, 12},
		{`V3: addressFromStringValue("3N5gLQdnHpJtk3uFpfiyUMsatT81zGuyhqL") == Address(base58'3N5gLQdnHpJtk3uFpfiyUMsatT81zGuyhqL')`, "AwkAAAAAAAACCQEAAAAcQGV4dHJVc2VyKGFkZHJlc3NGcm9tU3RyaW5nKQAAAAECAAAAIzNONWdMUWRuSHBKdGszdUZwZml5VU1zYXRUODF6R3V5aHFMCQEAAAAHQWRkcmVzcwAAAAEBAAAAGgFUrOhncsHOXnAEh5eecx07NcnKZJ0FJqAzoIvl0A==", 27, 126},
		{`V5: addressFromStringValue("3N5gLQdnHpJtk3uFpfiyUMsatT81zGuyhqL") == Address(base58'3N5gLQdnHpJtk3uFpfiyUMsatT81zGuyhqL')`, "BQkAAAAAAAACCQEAAAARQGV4dHJOYXRpdmUoMTA2MikAAAABAgAAACMzTjVnTFFkbkhwSnRrM3VGcGZpeVVNc2F0VDgxekd1eWhxTAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVKzoZ3LBzl5wBIeXnnMdOzXJymSdBSagMxtfMCE=", 8, 3},
		{`V5: parseIntValue("012345") == 12345`, "BAkAAAAAAAACCQEAAAANcGFyc2VJbnRWYWx1ZQAAAAECAAAABjAxMjM0NQAAAAAAAAAwOWLjTTs=", 9, 3},
		{`V5: let x = parseIntValue("12345"); x - x == 0`, "BQQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAUxMjM0NQkAAAAAAAACCQAAZQAAAAIFAAAAAXgFAAAAAXgAAAAAAAAAAAD38ehz", 12, 4},
		{`V3: let x = parseIntValue("12345"); 0 == 0`, "AwQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAUxMjM0NQkAAAAAAAACAAAAAAAAAAAAAAAAAAAAAAAAk6EsIQ==", 1, 1},
		{`V3: let x = parseIntValue("123"); let y = parseIntValue("456");  x + y == y + x`, "AwQAAAABeAkBAAAADXBhcnNlSW50VmFsdWUAAAABAgAAAAMxMjMEAAAAAXkJAQAAAA1wYXJzZUludFZhbHVlAAAAAQIAAAADNDU2CQAAAAAAAAIJAABkAAAAAgUAAAABeAUAAAABeQkAAGQAAAACBQAAAAF5BQAAAAF4sUY0sQ==", 59, 43},
		{`V4: let d = ["integer", "boolean", "binary", "string"]; d[0] == "integer"`, "BAQAAAABZAkABEwAAAACAgAAAAdpbnRlZ2VyCQAETAAAAAICAAAAB2Jvb2xlYW4JAARMAAAAAgIAAAAGYmluYXJ5CQAETAAAAAICAAAABnN0cmluZwUAAAADbmlsCQAAAAAAAAIJAAGRAAAAAgUAAAABZAAAAAAAAAAAAAIAAAAHaW50ZWdlcj/hEVY=", 9, 7},
		{`V3: let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getInteger(d, "integer") == 100500`, "AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEAAAAAIFAAAAAWQCAAAAB2ludGVnZXIAAAAAAAABiJSeStXa", 21, 23},
		{`V3: let d = [DataEntry("integer", 100500), DataEntry("boolean", true), DataEntry("binary", base16'68656c6c6f'), DataEntry("string", "world")]; getString(d, "string") == "world"`, "AwQAAAABZAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHaW50ZWdlcgAAAAAAAAGIlAkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAAHYm9vbGVhbgYJAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABmJpbmFyeQEAAAAFaGVsbG8JAARMAAAAAgkBAAAACURhdGFFbnRyeQAAAAICAAAABnN0cmluZwIAAAAFd29ybGQFAAAAA25pbAkAAAAAAAACCQAEEwAAAAIFAAAAAWQCAAAABnN0cmluZwIAAAAFd29ybGRFTMLs", 21, 23},
		{`V3: let x = 1 + 2; x == 3`, "AwQAAAABeAkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACCQAAAAAAAAIFAAAAAXgAAAAAAAAAAAOZ3gHv", 3, 2},
		{`V3: let x = 2 + 2; let y = x - x; x - y == x`, "AwQAAAABeAkAAGQAAAACAAAAAAAAAAACAAAAAAAAAAACBAAAAAF5CQAAZQAAAAIFAAAAAXgFAAAAAXgJAAAAAAAAAgkAAGUAAAACBQAAAAF4BQAAAAF5BQAAAAF4G74APQ==", 9, 4},
		{`V3: let a = 1 + 2; let b = 2; let c = a + b; b == 2`, "AwQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACBAAAAAFiAAAAAAAAAAACBAAAAAFjCQAAZAAAAAIFAAAAAWEFAAAAAWIJAAAAAAAAAgUAAAABYgAAAAAAAAAAAuTY7N4=", 2, 1},
		{`V3: let x = if true then 1 else 1 + 1; x == 1`, "AwQAAAABeAMGAAAAAAAAAAABCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgUAAAABeAAAAAAAAAAAAQZLIuM=", 3, 1},
		{`V3: let x = if true then if false then 1 + 1 + 1 else 1 + 1 else 1; x == 2`, "AwQAAAABeAMGAwcJAABkAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAABAAAAAAAAAAABCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEJAAAAAAAAAgUAAAABeAAAAAAAAAAAAgr3wMQ=", 5, 2},
		{`V3: let a = 1 + 2 + 3; let b = 4 + 5; let c = if false then a else b; c == 9`, "AwQAAAABYQkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAIAAAAAAAAAAAMEAAAAAWIJAABkAAAAAgAAAAAAAAAABAAAAAAAAAAABQQAAAABYwMHBQAAAAFhBQAAAAFiCQAAAAAAAAIFAAAAAWMAAAAAAAAAAAl/11/T", 5, 2},
		{`V3: let a = unit; let b = unit; let c = unit; let d = unit; let x = if true then a else b; let y = if false then c else d; x == y`, "AwQAAAABYQUAAAAEdW5pdAQAAAABYgUAAAAEdW5pdAQAAAABYwUAAAAEdW5pdAQAAAABZAUAAAAEdW5pdAQAAAABeAMGBQAAAAFhBQAAAAFiBAAAAAF5AwcFAAAAAWMFAAAAAWQJAAAAAAAAAgUAAAABeAUAAAABeei/I5Y=", 9, 1},
		{`V3: let s = size(toString(1000)); s != 0`, "AwQAAAABcwkAATEAAAABCQABpAAAAAEAAAAAAAAAA+gJAQAAAAIhPQAAAAIFAAAAAXMAAAAAAAAAAACmTwkf", 8, 3},
		{`V3: let a = "A"; let x = a + if true then {let c = "C"; c} else {let b = "B"; b}; x == "AC"`, "AwQAAAABYQIAAAABQQQAAAABeAkAASwAAAACBQAAAAFhAwYEAAAAAWMCAAAAAUMFAAAAAWMEAAAAAWICAAAAAUIFAAAAAWIJAAAAAAAAAgUAAAABeAIAAAACQUNpy4Pz", 15, 11},
		{`V3: func first(a: Int, b: Int) = {let x = a + b; x}; first(1, 2) == 3`, "AwoBAAAABWZpcnN0AAAAAgAAAAFhAAAAAWIEAAAAAXgJAABkAAAAAgUAAAABYQUAAAABYgUAAAABeAkAAAAAAAACCQEAAAAFZmlyc3QAAAACAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADefozrQ==", 5, 2},
		{`V3: func f(a: Int) = 1; func g(a: Int) = 2; f(g(1)) == 1`, "AwoBAAAAAWYAAAABAAAAAWEAAAAAAAAAAAEKAQAAAAFnAAAAAQAAAAFhAAAAAAAAAAACCQAAAAAAAAIJAQAAAAFmAAAAAQkBAAAAAWcAAAABAAAAAAAAAAABAAAAAAAAAAABRfhbwA==", 1, 3},
		{`V3: func inc(y: Int) = y + 1; let xxx = 5; inc(xxx) == 6`, "AwoBAAAAA2luYwAAAAEAAAABeQkAAGQAAAACBQAAAAF5AAAAAAAAAAABBAAAAAN4eHgAAAAAAAAAAAUJAAAAAAAAAgkBAAAAA2luYwAAAAEFAAAAA3h4eAAAAAAAAAAABu6Xgew=", 4, 2},
		{`V3: func f() = {func f() = {func f() = {1}; f()}; f()}; f() == 1`, "AwoBAAAAAWYAAAAACgEAAAABZgAAAAAKAQAAAAFmAAAAAAAAAAAAAAAAAQkBAAAAAWYAAAAACQEAAAABZgAAAAAJAAAAAAAAAgkBAAAAAWYAAAAAAAAAAAAAAAABHY7j7w==", 1, 2},
		{`V3: func f(a: Int) = a; f(1) == 1`, "AwoBAAAAAWYAAAABAAAAAWEFAAAAAWEJAAAAAAAAAgkBAAAAAWYAAAABAAAAAAAAAAABAAAAAAAAAAABAYVjTw==", 2, 2},
		{`V3: func inc(xxx: Int) = xxx + 1; let xxx = 5; inc(xxx) == 6`, "AwoBAAAAA2luYwAAAAEAAAADeHh4CQAAZAAAAAIFAAAAA3h4eAAAAAAAAAAAAQQAAAADeHh4AAAAAAAAAAAFCQAAAAAAAAIJAQAAAANpbmMAAAABBQAAAAN4eHgAAAAAAAAAAAZNSkZq", 4, 2},
		{`V3: func inc(y: Int) = y + 1; let xxx = 5; inc(xxx) == 6`, "AwoBAAAAA2luYwAAAAEAAAABeQkAAGQAAAACBQAAAAF5AAAAAAAAAAABBAAAAAN4eHgAAAAAAAAAAAUJAAAAAAAAAgkBAAAAA2luYwAAAAEFAAAAA3h4eAAAAAAAAAAABu6Xgew=", 4, 2},
		{`V3: func inc(y: Int) = y + 1; inc({let x = 5; x}) == 6`, "AwoBAAAAA2luYwAAAAEAAAABeQkAAGQAAAACBQAAAAF5AAAAAAAAAAABCQAAAAAAAAIJAQAAAANpbmMAAAABBAAAAAF4AAAAAAAAAAAFBQAAAAF4AAAAAAAAAAAGOrtXsw==", 4, 2},
		{`V3: func add(x: Int, y: Int) = x + y; let a = 2; let b = 3; add(a, b) == 5`, "AwoBAAAAA2FkZAAAAAIAAAABeAAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkEAAAAAWEAAAAAAAAAAAIEAAAAAWIAAAAAAAAAAAMJAAAAAAAAAgkBAAAAA2FkZAAAAAIFAAAAAWEFAAAAAWIAAAAAAAAAAAXSOexF", 6, 2},
		{`V3: func add(x: Int, y: Int) = x + y; let a = 2; let y = 3; add(a, y) == 5`, "AwoBAAAAA2FkZAAAAAIAAAABeAAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkEAAAAAWEAAAAAAAAAAAIEAAAAAXkAAAAAAAAAAAMJAAAAAAAAAgkBAAAAA2FkZAAAAAIFAAAAAWEFAAAAAXkAAAAAAAAAAAVtyJg5", 6, 2},
		{`V3: func add(x: Int, y: Int) = x + y; let x = 2; let y = 3; add(x, y) == 5`, "AwoBAAAAA2FkZAAAAAIAAAABeAAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkEAAAAAXgAAAAAAAAAAAIEAAAAAXkAAAAAAAAAAAMJAAAAAAAAAgkBAAAAA2FkZAAAAAIFAAAAAXgFAAAAAXkAAAAAAAAAAAVMfO15", 6, 2},
		{`V3: let me = 1 + 1 + 1 + 1; func third(p: Int) = me; func second(me: Int) = third(me); func first() = second(1); first() + first() + first() + first() + first() + first() == 24`, "BAQAAAACbWUJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEKAQAAAAV0aGlyZAAAAAEAAAABcAUAAAACbWUKAQAAAAZzZWNvbmQAAAABAAAAAm1lCQEAAAAFdGhpcmQAAAABBQAAAAJtZQoBAAAABWZpcnN0AAAAAAkBAAAABnNlY29uZAAAAAEAAAAAAAAAAAEJAAAAAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAQAAAAVmaXJzdAAAAAAJAQAAAAVmaXJzdAAAAAAJAQAAAAVmaXJzdAAAAAAJAQAAAAVmaXJzdAAAAAAJAQAAAAVmaXJzdAAAAAAJAQAAAAVmaXJzdAAAAAAAAAAAAAAAABikW/Rn", 21, 14},
		{`V3: let me = 1 + 1 + 1 + 1; func third(p: Int) = me; func second(me: Int) = third(me); func first() = second(1); first() + first() == 8`, "BAQAAAACbWUJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEAAAAAAAAAAAEKAQAAAAV0aGlyZAAAAAEAAAABcAUAAAACbWUKAQAAAAZzZWNvbmQAAAABAAAAAm1lCQEAAAAFdGhpcmQAAAABBQAAAAJtZQoBAAAABWZpcnN0AAAAAAkBAAAABnNlY29uZAAAAAEAAAAAAAAAAAEJAAAAAAAAAgkAAGQAAAACCQEAAAAFZmlyc3QAAAAACQEAAAAFZmlyc3QAAAAAAAAAAAAAAAAIg917jQ==", 9, 6},
		{`V3: let b = false; let x = if b then {func aaa(i:Int) = i + i + i + i + i + i; aaa(1)} else {func aaa(i: Int) = i + i + i + i; aaa(2)}; x == 8`, "AwQAAAABYgcEAAAAAXgDBQAAAAFiCgEAAAADYWFhAAAAAQAAAAFpCQAAZAAAAAIJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIJAABkAAAAAgUAAAABaQUAAAABaQUAAAABaQUAAAABaQUAAAABaQUAAAABaQkBAAAAA2FhYQAAAAEAAAAAAAAAAAEKAQAAAANhYWEAAAABAAAAAWkJAABkAAAAAgkAAGQAAAACCQAAZAAAAAIFAAAAAWkFAAAAAWkFAAAAAWkFAAAAAWkJAQAAAANhYWEAAAABAAAAAAAAAAACCQAAAAAAAAIFAAAAAXgAAAAAAAAAAAgfLlvD", 11, 4},
		{`V3: let x = 0; let y = if true then x else x + 1; y == 0`, "AgQAAAABeAAAAAAAAAAAAAQAAAABeQMGBQAAAAF4CQAAZAAAAAIFAAAAAXgAAAAAAAAAAAEJAAAAAAAAAgUAAAABeQAAAAAAAAAAALitwEo=", 4, 1},
		{`V3: let a = false; if false then a else !a`, "AwQAAAABYQcDBwUAAAABYQkBAAAAASEAAAABBQAAAAFhaKH61g==", 4, 1},
		{`V3: let a = 1; if true then {let b = 2; a == 1} else {let b = 2; a + b == 3}`, "AgQAAAABYQAAAAAAAAAAAQMGBAAAAAFiAAAAAAAAAAACCQAAAAAAAAIFAAAAAWEAAAAAAAAAAAEEAAAAAWIAAAAAAAAAAAIJAAAAAAAAAgkAAGQAAAACBQAAAAFhBQAAAAFiAAAAAAAAAAADxhrdbw==", 3, 1},
		{`V3: let a = 1; if true then a == 1 else {let b = 2; a + b == 3}`, "AgQAAAABYQAAAAAAAAAAAQMGCQAAAAAAAAIFAAAAAWEAAAAAAAAAAAEEAAAAAWIAAAAAAAAAAAIJAAAAAAAAAgkAAGQAAAACBQAAAAFhBQAAAAFiAAAAAAAAAAADBu60OQ==", 3, 1},
		{`V3: let a = 1; let b = 2; let c = if true then a else a + b; c == 1`, "AwQAAAABYQAAAAAAAAAAAQQAAAABYgAAAAAAAAAAAgQAAAABYwMGBQAAAAFhCQAAZAAAAAIFAAAAAWEFAAAAAWIJAAAAAAAAAgUAAAABYwAAAAAAAAAAAUWOLX8=", 4, 1},
		{`V3: let a = 1; let b = 2; let c = if false then a else a + b; c == 3`, "AwQAAAABYQAAAAAAAAAAAQQAAAABYgAAAAAAAAAAAgQAAAABYwMHBQAAAAFhCQAAZAAAAAIFAAAAAWEFAAAAAWIJAAAAAAAAAgUAAAABYwAAAAAAAAAAA+RWBBg=", 6, 2},
		{`V3: let a = 1; if true then {let b = 2; a + b == 3} else {let b = 2; a == 1}`, "AwQAAAABYQAAAAAAAAAAAQMGBAAAAAFiAAAAAAAAAAACCQAAAAAAAAIJAABkAAAAAgUAAAABYQUAAAABYgAAAAAAAAAAAwQAAAABYgAAAAAAAAAAAgkAAAAAAAACBQAAAAFhAAAAAAAAAAABQ00EmQ==", 5, 2},
		{`V3: let a = 1; let b = a; let c = a + b; c == 2`, "AwQAAAABYQAAAAAAAAAAAQQAAAABYgUAAAABYQQAAAABYwkAAGQAAAACBQAAAAFhBQAAAAFiCQAAAAAAAAIFAAAAAWMAAAAAAAAAAAJgDWGp", 6, 2},
		{`V3: let a = 1; func f() = {if true then {func f() = {let b = 2; a == 1}; f()} else {func f() = {let b = 2; a + b == 3}; f()}}; f()`, "AwQAAAABYQAAAAAAAAAAAQoBAAAAAWYAAAAAAwYKAQAAAAFmAAAAAAQAAAABYgAAAAAAAAAAAgkAAAAAAAACBQAAAAFhAAAAAAAAAAABCQEAAAABZgAAAAAKAQAAAAFmAAAAAAQAAAABYgAAAAAAAAAAAgkAAAAAAAACCQAAZAAAAAIFAAAAAWEFAAAAAWIAAAAAAAAAAAMJAQAAAAFmAAAAAAkBAAAAAWYAAAAACEd93A==", 3, 1},
		{`V3: func a(v: Int) = v; func b(x: Int, y: Int) = a(x) + a(y); let x = 1; let y = 2; b(x, y) == 3`, "AwoBAAAAAWEAAAABAAAAAXYFAAAAAXYKAQAAAAFiAAAAAgAAAAF4AAAAAXkJAABkAAAAAgkBAAAAAWEAAAABBQAAAAF4CQEAAAABYQAAAAEFAAAAAXkEAAAAAXgAAAAAAAAAAAEEAAAAAXkAAAAAAAAAAAIJAAAAAAAAAgkBAAAAAWIAAAACBQAAAAF4BQAAAAF5AAAAAAAAAAADMSWjrA==", 8, 4},
		{"V3: @Verifier(tx) func verify() = true", "AAIDAAAAAAAAAAIIAQAAAAAAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAaQqCec", 0, 0},
		{`V3: let a = 1\n@Verifier(tx) func verify() = true`, "AAIDAAAAAAAAAAIIAQAAAAEAAAAAAWEAAAAAAAAAAAEAAAAAAAAAAQAAAAJ0eAEAAAAGdmVyaWZ5AAAAAAZyQF8r", 0, 0},
		{`V3: let a = 1\nfunc inc(v: Int) = {v + 1}\n@Verifier(tx) func verify() = false`, "AAIDAAAAAAAAAAIIAQAAAAIAAAAAAWEAAAAAAAAAAAEBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABAAAAAAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAGwI5hqw==", 0, 0},
		{`V3: let a = 1\nfunc inc(v: Int) = {v + 1}\n@Verifier(tx) func verify() = inc(a) == 2`, "AAIDAAAAAAAAAAIIAQAAAAIAAAAAAWEAAAAAAAAAAAEBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABAAAAAAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAAAAAAAAgkBAAAAA2luYwAAAAEFAAAAAWEAAAAAAAAAAAJtD5WX", 4, 2},
		{`V3: let a = 1\nlet b = 1\nfunc inc(v: Int) = {v + 1}\nfunc add(x: Int, y: Int) = {x + y}\n@Verifier(tx) func verify() = inc(a) == add(a, b)`, "AAIDAAAAAAAAAAIIAQAAAAQAAAAAAWEAAAAAAAAAAAEAAAAAAWIAAAAAAAAAAAEBAAAAA2luYwAAAAEAAAABdgkAAGQAAAACBQAAAAF2AAAAAAAAAAABAQAAAANhZGQAAAACAAAAAXgAAAABeQkAAGQAAAACBQAAAAF4BQAAAAF5AAAAAAAAAAEAAAACdHgBAAAABnZlcmlmeQAAAAAJAAAAAAAAAgkBAAAAA2luYwAAAAEFAAAAAWEJAQAAAANhZGQAAAACBQAAAAFhBQAAAAFiDbIkmw==", 9, 3},
		{`V3: func b(x: Int) = {func a(y: Int) = x + y; a(1) + a(2)}; b(2) + b(3) == 16`, "AwoBAAAAAWIAAAABAAAAAXgKAQAAAAFhAAAAAQAAAAF5CQAAZAAAAAIFAAAAAXgFAAAAAXkJAABkAAAAAgkBAAAAAWEAAAABAAAAAAAAAAABCQEAAAABYQAAAAEAAAAAAAAAAAIJAAAAAAAAAgkAAGQAAAACCQEAAAABYgAAAAEAAAAAAAAAAAIJAQAAAAFiAAAAAQAAAAAAAAAAAwAAAAAAAAAAEHsYhwk=", 16, 8},
		{`V3: func a(v: Int) = 1; func b(x: Int) = a(1) + a(x); let x = 1; b(x) == 2`, "AwoBAAAAAWEAAAABAAAAAXYAAAAAAAAAAAEKAQAAAAFiAAAAAQAAAAF4CQAAZAAAAAIJAQAAAAFhAAAAAQAAAAAAAAAAAQkBAAAAAWEAAAABBQAAAAF4BAAAAAF4AAAAAAAAAAABCQAAAAAAAAIJAQAAAAFiAAAAAQUAAAABeAAAAAAAAAAAAoNKT2c=", 4, 4},
		{`V3: func a(v: Int) = 1; func b(x: Int) = a(x) + a(1); let x = 1; b(x) == 2`, "AwoBAAAAAWEAAAABAAAAAXYAAAAAAAAAAAEKAQAAAAFiAAAAAQAAAAF4CQAAZAAAAAIJAQAAAAFhAAAAAQUAAAABeAkBAAAAAWEAAAABAAAAAAAAAAABBAAAAAF4AAAAAAAAAAABCQAAAAAAAAIJAQAAAAFiAAAAAQUAAAABeAAAAAAAAAAAAjrCFFA=", 4, 4},
		{`V3: let x = 1; let y = 2; func a(x: Int) = x; func b(x: Int, y: Int) = {let r = a(x) + a(y); r}; b(x, y) == 3`, "AwQAAAABeAAAAAAAAAAAAQQAAAABeQAAAAAAAAAAAgoBAAAAAWEAAAABAAAAAXgFAAAAAXgKAQAAAAFiAAAAAgAAAAF4AAAAAXkEAAAAAXIJAABkAAAAAgkBAAAAAWEAAAABBQAAAAF4CQEAAAABYQAAAAEFAAAAAXkFAAAAAXIJAAAAAAAAAgkBAAAAAWIAAAACBQAAAAF4BQAAAAF5AAAAAAAAAAAD32P71Q==", 9, 4},
		{`V3: let x = 1; let y = 2; func a(x: Int) = x; func b(x: Int, y: Int) = {let r = a(y) + a(x); r}; b(x, y) == 3`, "AwQAAAABeAAAAAAAAAAAAQQAAAABeQAAAAAAAAAAAgoBAAAAAWEAAAABAAAAAXgFAAAAAXgKAQAAAAFiAAAAAgAAAAF4AAAAAXkEAAAAAXIJAABkAAAAAgkBAAAAAWEAAAABBQAAAAF5CQEAAAABYQAAAAEFAAAAAXgFAAAAAXIJAAAAAAAAAgkBAAAAAWIAAAACBQAAAAF4BQAAAAF5AAAAAAAAAAADoJh89A==", 9, 4},
		{`V4: let x = (1, "Two", true); x._3`, "BAQAAAABeAkABRUAAAADAAAAAAAAAAABAgAAAANUd28GCAUAAAABeAAAAAJfM5iN5Ik=", 3, 1},
		{`V4: let x = if true then (1, 2) else (true, "q"); match x {case _: (Boolean, String) => false; case _: (Int, Int) => true}`, "BAQAAAABeAMGCQAFFAAAAAIAAAAAAAAAAAEAAAAAAAAAAAIJAAUUAAAAAgYCAAAAAXEEAAAAByRtYXRjaDAFAAAAAXgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAEShCb29sZWFuLCBTdHJpbmcpBwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAKKEludCwgSW50KQYJAAACAAAAAQIAAAALTWF0Y2ggZXJyb3IMWMC4", 9, 3},
		{`V4: let t = ((1, "Two", true), (5, "Six", false)); t._1._3`, "BAQAAAABdAkABRQAAAACCQAFFQAAAAMAAAAAAAAAAAECAAAAA1R3bwYJAAUVAAAAAwAAAAAAAAAABQIAAAADU2l4BwgIBQAAAAF0AAAAAl8xAAAAAl8zuG3UeQ==", 6, 3},
		{`V4: !sigVerify_8Kb(base58'', base58'',base58'')`, "BAkBAAAAASEAAAABCQAJxAAAAAMBAAAAAAEAAAAAAQAAAADm58fQ", 49, 48},
		{`V4: !sigVerify_64Kb(base58'', base58'',base58'')`, "BAkBAAAAASEAAAABCQAJxwAAAAMBAAAAAAEAAAAAAQAAAACYsebz", 104, 103},
		{`V4: containsElement([base58'', base58''],base58'')`, "BAkBAAAAD2NvbnRhaW5zRWxlbWVudAAAAAIJAARMAAAAAgEAAAAACQAETAAAAAIBAAAAAAUAAAADbmlsAQAAAAAXL3j5", 15, 7},
		{`V5: pow((100), 4, 5, 1, 2, FLOOR) == 10`, "BQkAAAAAAAACCQAAbAAAAAYAAAAAAAAAAGQAAAAAAAAAAAQAAAAAAAAAAAUAAAAAAAAAAAEAAAAAAAAAAAIFAAAABUZMT09SAAAAAAAAAAAK3GfUhw==", 102, 101},
	} {
		checkVerifierSpentComplexityV5(t, test.source, test.complexityV5, test.comment)
		checkVerifierSpentComplexityV6(t, test.source, test.complexityV6, test.comment)
	}
}

func TestFold(t *testing.T) {
	/*
		{-# STDLIB_VERSION 5 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ACCOUNT #-}

		func sum(accum: Int, next: Int) = accum + next
		let arr = [1,2,3,4,5]
		FOLD<5>(arr, 0, sum) == 15
	*/

	code := "BQoBAAAAA3N1bQAAAAIAAAAFYWNjdW0AAAAEbmV4dAkAAGQAAAACBQAAAAVhY2N1bQUAAAAEbmV4dAQAAAADYXJyCQAETAAAAAIAAAAAAAAAAAEJAARMAAAAAgAAAAAAAAAAAgkABEwAAAACAAAAAAAAAAADCQAETAAAAAIAAAAAAAAAAAQJAARMAAAAAgAAAAAAAAAABQUAAAADbmlsCQAAAAAAAAIKAAAAAAIkbAUAAAADYXJyCgAAAAACJHMJAAGQAAAAAQUAAAACJGwKAAAAAAUkYWNjMAAAAAAAAAAAAAoBAAAAATEAAAACAAAAAiRhAAAAAiRpAwkAAGcAAAACBQAAAAIkaQUAAAACJHMFAAAAAiRhCQEAAAADc3VtAAAAAgUAAAACJGEJAAGRAAAAAgUAAAACJGwFAAAAAiRpCgEAAAABMgAAAAIAAAACJGEAAAACJGkDCQAAZwAAAAIFAAAAAiRpBQAAAAIkcwUAAAACJGEJAAACAAAAAQIAAAATTGlzdCBzaXplIGV4Y2VlZHMgNQkBAAAAATIAAAACCQEAAAABMQAAAAIJAQAAAAExAAAAAgkBAAAAATEAAAACCQEAAAABMQAAAAIJAQAAAAExAAAAAgUAAAAFJGFjYzAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAIAAAAAAAAAAAMAAAAAAAAAAAQAAAAAAAAAAAUAAAAAAAAAAA/IH77b"
	checkVerifierSpentComplexityV5(t, code, 77, "")
	checkVerifierSpentComplexityV6(t, code, 29, "")
}

func TestUserFunctionsArguments(t *testing.T) {
	for _, test := range []struct {
		comment      string
		source       string
		complexityV5 int
		complexityV6 int
	}{
		{`func A(x: Int, y: Int) = x + y; A(1, 2) == 3`, "BQoBAAAAAUEAAAACAAAAAXgAAAABeQkAAGQAAAACBQAAAAF4BQAAAAF5CQAAAAAAAAIJAQAAAAFBAAAAAgAAAAAAAAAAAQAAAAAAAAAAAgAAAAAAAAAAAzMPzEQ=", 4, 2},
		{`func A(x: Int, y: Int, z: Int) = x + y; A(1, 2, 3) == 3`, "BQoBAAAAAUEAAAADAAAAAXgAAAABeQAAAAF6CQAAZAAAAAIFAAAAAXgFAAAAAXkJAAAAAAAAAgkBAAAAAUEAAAADAAAAAAAAAAABAAAAAAAAAAACAAAAAAAAAAADAAAAAAAAAAADZO4krg==", 4, 2},
		{`func A(x: Int, y: Int, z: Int) = x + y + z; A(1, 2, 3) == 6`, "BQoBAAAAAUEAAAADAAAAAXgAAAABeQAAAAF6CQAAZAAAAAIJAABkAAAAAgUAAAABeAUAAAABeQUAAAABegkAAAAAAAACCQEAAAABQQAAAAMAAAAAAAAAAAEAAAAAAAAAAAIAAAAAAAAAAAMAAAAAAAAAAAbT+Mrp", 6, 3},
		{`func A(x: Int, y: Int) = x + y; A(1+2, 3+4) == 10`, "BQoBAAAAAUEAAAACAAAAAXgAAAABeQkAAGQAAAACBQAAAAF4BQAAAAF5CQAAAAAAAAIJAQAAAAFBAAAAAgkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACCQAAZAAAAAIAAAAAAAAAAAMAAAAAAAAAAAQAAAAAAAAAAApvNs+6", 6, 4},
		{`func A(x: Int, y: Int, z: Int) = x + y; A(1+2, 3+4, 5+6) == 10`, "BQoBAAAAAUEAAAADAAAAAXgAAAABeQAAAAF6CQAAZAAAAAIFAAAAAXgFAAAAAXkJAAAAAAAAAgkBAAAAAUEAAAADCQAAZAAAAAIAAAAAAAAAAAEAAAAAAAAAAAIJAABkAAAAAgAAAAAAAAAAAwAAAAAAAAAABAkAAGQAAAACAAAAAAAAAAAFAAAAAAAAAAAGAAAAAAAAAAAKtr2G5Q==", 7, 5},
		{`let a = 1 + 2;let b = 3 + 4;let c = 5 + 6;func A(x: Int, y: Int) = x + y; A(a, b) == 10`, "BQQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACBAAAAAFiCQAAZAAAAAIAAAAAAAAAAAMAAAAAAAAAAAQEAAAAAWMJAABkAAAAAgAAAAAAAAAABQAAAAAAAAAABgoBAAAAAUEAAAACAAAAAXgAAAABeQkAAGQAAAACBQAAAAF4BQAAAAF5CQAAAAAAAAIJAQAAAAFBAAAAAgUAAAABYQUAAAABYgAAAAAAAAAACvKnofI=", 8, 4},
		{`let a = 1 + 2;let b = 3 + 4;let c = 5 + 6;func A(x: Int, y: Int, z: Int) = x + y; A(a, b, c) == 10`, "BQQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACBAAAAAFiCQAAZAAAAAIAAAAAAAAAAAMAAAAAAAAAAAQEAAAAAWMJAABkAAAAAgAAAAAAAAAABQAAAAAAAAAABgoBAAAAAUEAAAADAAAAAXgAAAABeQAAAAF6CQAAZAAAAAIFAAAAAXgFAAAAAXkJAAAAAAAAAgkBAAAAAUEAAAADBQAAAAFhBQAAAAFiBQAAAAFjAAAAAAAAAAAKOeIr0Q==", 10, 5},
		{`let a = 1 + 2;let b = 3 + 4;let c = 5 + 6;func A(x: Int, y: Int, z: Int) = x + x; A(a+1, b+2, c+3) == 8`, "BQQAAAABYQkAAGQAAAACAAAAAAAAAAABAAAAAAAAAAACBAAAAAFiCQAAZAAAAAIAAAAAAAAAAAMAAAAAAAAAAAQEAAAAAWMJAABkAAAAAgAAAAAAAAAABQAAAAAAAAAABgoBAAAAAUEAAAADAAAAAXgAAAABeQAAAAF6CQAAZAAAAAIFAAAAAXgFAAAAAXgJAAAAAAAAAgkBAAAAAUEAAAADCQAAZAAAAAIFAAAAAWEAAAAAAAAAAAEJAABkAAAAAgUAAAABYgAAAAAAAAAAAgkAAGQAAAACBQAAAAFjAAAAAAAAAAADAAAAAAAAAAAIY2FB4Q==", 13, 8},
	} {
		checkVerifierSpentComplexityV5(t, test.source, test.complexityV5, test.comment)
		checkVerifierSpentComplexityV6(t, test.source, test.complexityV6, test.comment)
	}
}

func TestComplexityOverflow(t *testing.T) {
	dApp1 := newTestAccount(t, "DAPP1")   // 3MzDtgL5yw73C2xVLnLJCrT5gCL4357a4sz
	dApp2 := newTestAccount(t, "DAPP2")   // 3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1
	sender := newTestAccount(t, "SENDER") // 3N8CkZAyS4XcDoJTJoKNuNk2xmNKmQj7myW

	/* On dApp1 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	@Callable(i)
	func call() = {
	  strict a = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict b = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict c = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict d = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict e = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict f = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  strict g = invoke(Address(base58'3N7Te7NXtGVoQqFqktwrFhQWAkc6J8vfPQ1'),  "call", [], [])
	  []
	}
	*/
	_, tree1 := parseBase64Script(t, "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAAAWEJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWEFAAAAAWEEAAAAAWIJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWIFAAAAAWIEAAAAAWMJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWMFAAAAAWMEAAAAAWQJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWQFAAAAAWQEAAAAAWUJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWUFAAAAAWUEAAAAAWYJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWYFAAAAAWYEAAAAAWcJAAP8AAAABAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVMByBn03y+jAvm4M5s8/31mxeRh33VavrgIAAAAEY2FsbAUAAAADbmlsBQAAAANuaWwDCQAAAAAAAAIFAAAAAWcFAAAAAWcFAAAAA25pbAkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4JAAACAAAAAQIAAAAkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuCQAAAgAAAAECAAAAJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkAAAIAAAABAgAAACRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4AAAAAETSeDA==")

	/* On dApp2 address
	{-# STDLIB_VERSION 5 #-}
	{-# CONTENT_TYPE DAPP #-}
	{-# SCRIPT_TYPE ACCOUNT #-}

	let msg = base16'135212a9cf00d0a05220be7323bfa4a5ba7fc5465514007702121a9c92e46bd473062f00841af83cb7bc4b2cd58dc4d5b151244cc8293e795796835ed36822c6e09893ec991b38ada4b21a06e691afa887db4e9d7b1d2afc65ba8d2f5e6926ff53d2d44d55fa095f3fad62545c714f0f3f59e4bfe91af8'
	let sig = base16'd971ec27c5bfc384804c8d8d6a2de9edc3d957b25e488e954a71ef4c4a87f5fb09cfdf6bd26cffc49d03048e8edb0c918061be158d737c2e11cc7210263efb85'
	let bad = base16'44164f23a95ed2662c5b1487e8fd688be9032efa23dd2ef29b018d33f65d0043df75f3ac1d44b4bda50e8b07e0b49e2898bec80adbf7604e72ef6565bd2f8189'
	let pk = base16'ba9e7203ca62efbaa49098ec408bdf8a3dfed5a7fa7c200ece40aade905e535f'

	@Callable(i)
	func call() = {
	  strict a = sigVerify(msg, sig, pk)
	  strict b = sigVerify(msg, bad, pk)
	  strict c = sigVerify(msg, sig, pk)
	  strict d = sigVerify(msg, bad, pk)
	  strict e = sigVerify(msg, sig, pk)
	  strict f = sigVerify(msg, bad, pk)
	  strict g = sigVerify(msg, sig, pk)
	  strict h = sigVerify(msg, bad, pk)
	  strict ii = sigVerify(msg, sig, pk)
	  strict j = sigVerify(msg, bad, pk)
	  strict k = sigVerify(msg, sig, pk)
	  strict l = sigVerify(msg, bad, pk)
	  strict m = sigVerify(msg, sig, pk)
	  strict n = sigVerify(msg, bad, pk)
	  strict p = sigVerify(msg, sig, pk)
	  strict q = sigVerify(msg, bad, pk)
	  strict r = sigVerify(msg, sig, pk)
	  strict s = sigVerify(msg, bad, pk)
	  strict t = sigVerify(msg, sig, pk)
	  ([], true)
	}
	*/
	_, tree2 := parseBase64Script(t, "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")

	env := newTestEnv(t).withLibVersion(ast.LibV5).withComplexityLimit(ast.LibV5, 26000).
		withBlockV5Activated().withProtobufTx().
		withDataEntriesSizeV2().withMessageLengthV3().
		withValidateInternalPayments().withThis(dApp1).
		withDApp(dApp1).withAdditionalDApp(dApp2).withSender(sender).
		withInvocation("call").
		withTree(dApp1, tree1).withTree(dApp2, tree2).
		withWavesBalance(dApp1, 0).withWavesBalance(dApp2, 1_00000000).withWavesBalance(sender, 0).
		withWrappedState()

	_, err := CallFunction(env.toEnv(), tree1, "call", proto.Arguments{})
	require.EqualError(t, err, "evaluation complexity 26149 exceeds the limit 26000")
}
