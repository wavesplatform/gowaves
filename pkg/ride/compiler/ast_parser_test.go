package compiler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

func parseBase64Script(t *testing.T, src string) (proto.Script, *ast.Tree) {
	script, err := base64.StdEncoding.DecodeString(src)
	require.NoError(t, err)
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	require.NotNil(t, tree)
	return script, tree
}

const (
	DappV6Directive = `{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
`
)

func TestDirectivesCompileFail(t *testing.T) {
	for _, test := range []struct {
		code     string
		errorMsg []string
	}{
		{`
{-# STDLIB_VERSION 7 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:21): invalid STDLIB_VERSION \"7\""}},
		{`
{-# STDLIB_VERSION 0 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:21): invalid STDLIB_VERSION \"0\""}},
		{`
{-# STDLIB_VERSION XXX #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:23): failed to parse version \"XXX\" : strconv.ParseInt: parsing \"XXX\": invalid syntax"}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE XXX #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:5, 3:17): Illegal directive value \"XXX\" for key \"CONTENT_TYPE\""}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# XXX XXX #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:5, 3:8): Illegal directive key \"XXX\""}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE XXX #-}`, []string{"(4:5, 4:16): Illegal directive value \"XXX\" for key \"SCRIPT_TYPE\""}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:1, 4:0): Directive key STDLIB_VERSION is used more than once"}},
	} {
		code := test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		assert.Equal(t, len(astParser.ErrorsList), len(test.errorMsg))
		for i, err := range astParser.ErrorsList {
			assert.Equal(t, err.Error(), test.errorMsg[i])
		}
	}
}

func TestDirectivesCompile(t *testing.T) {
	for _, test := range []struct {
		code     string
		expected ast.Tree
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, ast.Tree{ContentType: ast.ContentTypeApplication, LibVersion: ast.LibV6}},
		{`
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, ast.Tree{ContentType: ast.ContentTypeApplication, LibVersion: ast.LibV6}},
		{`
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, ast.Tree{ContentType: ast.ContentTypeExpression, LibVersion: ast.LibV4}},
	} {
		code := test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		assert.Equal(t, astParser.Tree.LibVersion, test.expected.LibVersion)
		assert.Equal(t, astParser.Tree.ContentType, test.expected.ContentType)
	}
}

func TestConstDeclaration(t *testing.T) {
	for _, test := range []struct {
		code       string
		fail       bool
		base64code string
	}{
		{`let a = 1`, false, "BgICCAIBAAFhAAEAAChWE0Q="},
		{`let a = true`, false, "BgICCAIBAAFhBgAAS/fwTw=="},
		{`let a = base64'SGVsbG8gd29ybGQhISE='`, false, "BgICCAIBAAFhAQ5IZWxsbyB3b3JsZCEhIQAASDmhkA=="},
		{`let a = base58'ABCDEFGHJKLMNPQRSTUVWXYZ'`, false, "BgICCAIBAAFhARID0HDIGzEBUoTjE/P/ScGTlIYAAAAolgA="},
		{`let a = base16'ABCDEFabcdef'`, false, "BgICCAIBAAFhAQarze+rze8AADZaKzc="},
		{`let a = [1, 2, 3, 4, 5]`, false, "BgICCAIBAAFhCQDMCAIAAQkAzAgCAAIJAMwIAgADCQDMCAIABAkAzAgCAAUFA25pbAAAPZacBA=="},
		{`let a = [1, "test", true, base64'', 5]`, false, "BgICCAIBAAFhCQDMCAIAAQkAzAgCAgR0ZXN0CQDMCAIGCQDMCAIBAAkAzAgCAAUFA25pbAAAbD363g=="},
	} {
		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		_, tree := parseBase64Script(t, test.base64code)
		assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
	}
}

func TestStringDeclaration(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`let a = "test"`, false, "BgICCAIBAAFhAgR0ZXN0AABM5UxM"},
		{`let a = ""`, false, "BgICCAIBAAFhAgAAALkZwZw="},
		{`let a = "\t\f\b\r\n"`, false, "BgICCAIBAAFhAgUJDAgNCgAAlYWq5w=="},
		{`let a = "\a"`, true, "(4:10, 4:12): unknown escaped symbol: '\\a'. The valid are \\b, \\f, \\n, \\r, \\t"},
		{`let a = "\u1234"`, false, "BgICCAIBAAFhAgPhiLQAAKUbIjo="},
		{`let a = "\u1234a\t"`, false, "BgICCAIBAAFhAgXhiLRhCQAADF+pNw=="},
	} {
		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestTupleDeclaration(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`let a = (1, 2, 3)`, false, "BgICCAIBAAFhCQCVCgMAAQACAAMAAI6t9SE="},
		{`let a = (1, "2", true)`, false, "BgICCAIBAAFhCQCVCgMAAQIBMgYAAIERlqw="},
		{`let a = (1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23)`, true, "(4:9, 4:92): invalid tuple len \"23\"(allowed 2 to 22)"},
		{`let (a, b, c) = (1, 2, 3)`, false, "BgICCAIEAAgkdDA3OTEwNAkAlQoDAAEAAgADAAFhCAUIJHQwNzkxMDQCXzEAAWIIBQgkdDA3OTEwNAJfMgABYwgFCCR0MDc5MTA0Al8zAAB8+W2s"},
		{`let a = (1, 2, 3)
let (b, c, d) = a`, false, "BgICCAIFAAFhCQCVCgMAAQACAAMACCR0MDk3MTE0BQFhAAFiCAUIJHQwOTcxMTQCXzEAAWMIBQgkdDA5NzExNAJfMgABZAgFCCR0MDk3MTE0Al8zAAAU7y0b"},
		{`let (a, b) = (1, "2", true)`, false, "BgICCAIDAAgkdDA3OTEwNgkAlQoDAAECATIGAAFhCAUIJHQwNzkxMDYCXzEAAWIIBQgkdDA3OTEwNgJfMgAAdj+WZg=="},
		{`let (a, b, c, d) = (1, "2", true)`, true, "(4:1, 4:34): Number of Identifiers must be <= tuple length"},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestOperators(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`let a = 1 + 2 + 3 + 4`, false, "BgICCAIBAAFhCQBkAgkAZAIJAGQCAAEAAgADAAQAADk9Pyk="},
		{`let a = "a" + "b"`, false, "BgICCAIBAAFhCQCsAgICAWECAWIAABJCapY="},
		{`let a = "a" + 1`, true, "(4:9, 4:16): Unexpected types for + operator: String, Int"},
		{`let a = 1 > 2`, false, "BgICCAIBAAFhCQBmAgABAAIAAKf+6ug="},
		{`let a = 1 < 2`, false, "BgICCAIBAAFhCQBmAgACAAEAAAO8zuo="},
		{`let a = 1 <= 2`, false, "BgICCAIBAAFhCQBnAgACAAEAAJShBI8="},
		{`let a = 1 >= 2`, false, "BgICCAIBAAFhCQBnAgABAAIAAPdIIeU="},
		{`let a = 1 >= "a"`, true, "(4:14, 4:17): Unexpected type, required: Int, but String found"},
		{`let a = 1 == "a"`, true, "(4:14, 4:17): Unexpected type, required: Int, but String found"},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestFOLD(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`func sum(accum: Int, next: Int) = accum + next
let arr = [1,2,3,4,5]
let a = FOLD<5>(arr, 0, sum)`, false, "BgICCAIDAQNzdW0CBWFjY3VtBG5leHQJAGQCBQVhY2N1bQUEbmV4dAADYXJyCQDMCAIAAQkAzAgCAAIJAMwIAgADCQDMCAIABAkAzAgCAAUFA25pbAABYQoAAiRsBQNhcnIKAAIkcwkAkAMBBQIkbAoABSRhY2MwAAAKAQUkZjBfMQICJGECJGkDCQBnAgUCJGkFAiRzBQIkYQkBA3N1bQIFAiRhCQCRAwIFAiRsBQIkaQoBBSRmMF8yAgIkYQIkaQMJAGcCBQIkaQUCJHMFAiRhCQACAQITTGlzdCBzaXplIGV4Y2VlZHMgNQkBBSRmMF8yAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgUFJGFjYzAAAAABAAIAAwAEAAUAABK5ZXo="},
		{`func filterEven(accum: List[Int], next: Int) =
   if (next % 2 == 0) then accum :+ next else accum
let arr = [1,2,3,4,5]
let a = FOLD<5>(arr, [], filterEven)`, false, "BgICCAIDAQpmaWx0ZXJFdmVuAgVhY2N1bQRuZXh0AwkAAAIJAGoCBQRuZXh0AAIAAAkAzQgCBQVhY2N1bQUEbmV4dAUFYWNjdW0AA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwAAWEKAAIkbAUDYXJyCgACJHMJAJADAQUCJGwKAAUkYWNjMAUDbmlsCgEFJGYwXzECAiRhAiRpAwkAZwIFAiRpBQIkcwUCJGEJAQpmaWx0ZXJFdmVuAgUCJGEJAJEDAgUCJGwFAiRpCgEFJGYwXzICAiRhAiRpAwkAZwIFAiRpBQIkcwUCJGEJAAIBAhNMaXN0IHNpemUgZXhjZWVkcyA1CQEFJGYwXzICCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECBQUkYWNjMAAAAAEAAgADAAQABQAAWwkCmw=="},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestExprSimple(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`{-# STDLIB_VERSION 6 #-}
		{-# CONTENT_TYPE EXPRESSION #-}
		{-# SCRIPT_TYPE ASSET #-}
		
		1 == 1`, false, "BgEJAAACAAEAAb+26yY="},
	} {

		rawAST, buf, err := buildAST(t, test.code, false)
		assert.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Verifier, astParser.Tree.Verifier)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestBuildInVars(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`let a = height`, false, "BgICCAIBAAFhBQZoZWlnaHQAABNT5zQ="},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestTypesInFuncs(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func test(a : BigInt) = true`, true, "(6:15, 6:21): Undefinded type BigInt"},
	} {
		rawAST, buf, err := buildAST(t, test.code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestFuncCalls(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`let a = 1.toBytes()`, false, "BgICCAIBAAFhCQCaAwEAAQAAWQ+cBQ=="},
		{`let a = addressFromPublicKey(base58'')`, false, "BgICCAIBAAFhCQCnCAEBAAAAG+9EKQ=="},
		{`let a = AssetPair(base58'', base58'')
		let b = a.amountAsset`, false, "AAIEAAAAAAAAAAIIAgAAAAIAAAAAAWEJAQAAAAlBc3NldFBhaXIAAAACAQAAAAABAAAAAAAAAAABYggFAAAAAWEAAAALYW1vdW50QXNzZXQAAAAAAAAAAIKGPR8="},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestMatchCase(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => true
	case _ => false
	case _ => false
}`,
			true, "(10:2, 11:0): Match should have at most one default case"},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => true
}`,
			true, "(7:9, 9:2): Match should have default case"},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => true
	case _ => false
}`,
			false, "BgICCAICAAFhAwYCAAAKAAFiBAckbWF0Y2gwBQFhAwkAAQIFByRtYXRjaDACA0ludAQBeAUHJG1hdGNoMAYHAACgeGuK"},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => true
	case x: String => true
	case _ => false
}`,
			false, "BgICCAICAAFhAwYCAAAKAAFiBAckbWF0Y2gwBQFhAwkAAQIFByRtYXRjaDACA0ludAQBeAUHJG1hdGNoMAYDCQABAgUHJG1hdGNoMAIGU3RyaW5nBAF4BQckbWF0Y2gwBgcAANXZst4="},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => true
	case x: Boolean => true
	case _ => false
}`,
			true, "(9:10, 9:17): Matching not exhaustive: possibleTypes are \"String|Int\", while matched are \"Boolean\""},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => x
	case x: String => true
	case _ => false
}`,
			false, "BgICCAICAAFhAwYCAAAKAAFiBAckbWF0Y2gwBQFhAwkAAQIFByRtYXRjaDACA0ludAQBeAUHJG1hdGNoMAUBeAMJAAECBQckbWF0Y2gwAgZTdHJpbmcEAXgFByRtYXRjaDAGBwAAbSLfzg=="},
		{`
let a = if true then "" else 10

let b = match a {
	case x: Int => y
	case x: String => true
	case _ => false
}`,
			true, "(8:17, 9:0): Variable \"y\" doesnt't exist"},
		{`
let a = if true then "" else 10

let b = match a {
case x: Int|String => true
case _ => false
}`,
			false, "BgICCAICAAFhAwYCAAAKAAFiBAckbWF0Y2gwBQFhAwMJAAECBQckbWF0Y2gwAgZTdHJpbmcGCQABAgUHJG1hdGNoMAIDSW50BAF4BQckbWF0Y2gwBgcAACOY/Cg="},
		{`
let a = if true then "" else 10

let b = match a {
case x: Int|Boolean => true
case _ => false
}`,
			true, "(8:9, 8:20): Matching not exhaustive: possibleTypes are \"String|Int\", while matched are \"Int|Boolean\""},
		{`
let a = if true then "" else 10

let c = if true then true else a

let b = match c {
	case x: Int|String => true
	case _ => false
}`,
			false, "BgICCAIDAAFhAwYCAAAKAAFjAwYGBQFhAAFiBAckbWF0Y2gwBQFjAwMJAAECBQckbWF0Y2gwAgZTdHJpbmcGCQABAgUHJG1hdGNoMAIDSW50BAF4BQckbWF0Y2gwBgcAALnOhIw="},
		{`
let a = (1, "")

let b = match a {
	case (Int, String) => true
	case _ => false
}`,
			false, "BgICCAICAAFhCQCUCgIAAQIAAAFiBAckbWF0Y2gwBQFhAwMGCQAAAgkAxgoBBQckbWF0Y2gwAAIHBANJbnQIBQckbWF0Y2gwAl8xBAZTdHJpbmcIBQckbWF0Y2gwAl8yBgcAAGRbLC8="},
		{`
let a = (1, "")

let b = match a {
	case (x: Int, y: String) => true
	case _ => false
}`,
			false, "BgICCAICAAFhCQCUCgIAAQIAAAFiBAckbWF0Y2gwBQFhAwMDCQABAggFByRtYXRjaDACXzECA0ludAkAAQIIBQckbWF0Y2gwAl8yAgZTdHJpbmcHCQABAgUHJG1hdGNoMAINKEludCwgU3RyaW5nKQcEAXgIBQckbWF0Y2gwAl8xBAF5CAUHJG1hdGNoMAJfMgYHAADymI82"},
		{`
let a = (1, "")

let b = match a {
	case (_: Int, _: String) => true
	case _ => false
}`,
			false, "BgICCAICAAFhCQCUCgIAAQIAAAFiBAckbWF0Y2gwBQFhAwMDCQABAggFByRtYXRjaDACXzECA0ludAkAAQIIBQckbWF0Y2gwAl8yAgZTdHJpbmcHCQABAgUHJG1hdGNoMAINKEludCwgU3RyaW5nKQcGBwAATi06Ag=="},
		{`
let a = (1, "")

let b = match a {
	case (_: Int, "") => true
	case _ => false
}`,
			false, "BgICCAICAAFhCQCUCgIAAQIAAAFiBAckbWF0Y2gwBQFhAwMDCQABAggFByRtYXRjaDACXzECA0ludAkAAAICAAgFByRtYXRjaDACXzIHCQABAgUHJG1hdGNoMAINKEludCwgU3RyaW5nKQcGBwAAAi3jiw=="},
		{`
let a = (1, "")

let b = match a {
	case (10, base16'') => true
	case _ => false
}`,
			true, "(8:12, 8:20): Matching not exhaustive: possibleTypes are \"(Int, String)\", while matched are \"(Int, ByteVector)\""},
		{`
let a = if true then AssetPair(base58'', base58'') else 10

let b = match a {
	case AssetPair(amountAsset = base16'', priceAsset = base16'') => true
	case _ => false
}`,
			false, "BgICCAICAAFhAwYJAQlBc3NldFBhaXICAQABAAAKAAFiBAckbWF0Y2gwBQFhAwMJAAECBQckbWF0Y2gwAglBc3NldFBhaXIEByRtYXRjaDAFByRtYXRjaDADCQAAAgEACAUHJG1hdGNoMAthbW91bnRBc3NldAkAAAIBAAgFByRtYXRjaDAKcHJpY2VBc3NldAcHBAckbWF0Y2gwBQckbWF0Y2gwBgcAAEe/y1c="},
		{`
let a = if true then AssetPair(base58'', base58'') else 10

let b = match a {
	case AssetPair(amountAsset = x, priceAsset = base16'') => x
	case _ => false
}`,
			false, "BgICCAICAAFhAwYJAQlBc3NldFBhaXICAQABAAAKAAFiBAckbWF0Y2gwBQFhAwMJAAECBQckbWF0Y2gwAglBc3NldFBhaXIEByRtYXRjaDAFByRtYXRjaDAJAAACAQAIBQckbWF0Y2gwCnByaWNlQXNzZXQHBAckbWF0Y2gwBQckbWF0Y2gwBAF4CAUHJG1hdGNoMAthbW91bnRBc3NldAUBeAcAAMEpH0Q="},
		{`
let a = if true then AssetPair(base58'', base58'') else 10

let b = match a {
	case AssetPair() => true
	case _ => false
}`,
			false, "BgICCAICAAFhAwYJAQlBc3NldFBhaXICAQABAAAKAAFiBAckbWF0Y2gwBQFhAwMJAAECBQckbWF0Y2gwAglBc3NldFBhaXIEByRtYXRjaDAFByRtYXRjaDAGBwQHJG1hdGNoMAUHJG1hdGNoMAYHAAD1dYoP"},
	} {

		code := DappV6Directive + test.code
		rawAST, buf, err := buildAST(t, code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestFuncCallsPrevVers(t *testing.T) {
	// addressFromPublicKey has id "addressFromPublicKey" up to version 6, and in 6 version id = 1063
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = addressFromPublicKey(base58'')`,
			false, "AAIEAAAAAAAAAAIIAgAAAAEAAAAAAWEJAQAAABRhZGRyZXNzRnJvbVB1YmxpY0tleQAAAAEBAAAAAAAAAAAAAAAATbuPXQ=="},
	} {

		rawAST, buf, err := buildAST(t, test.code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestFuncCallableFunc(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test() = {
	([StringEntry("a", "a")], unit)
}
`,
			false, "BgIECAISAAABAWkBBHRlc3QACQCUCgIJAMwIAgkBC1N0cmluZ0VudHJ5AgIBYQIBYQUDbmlsBQR1bml0ACkqhGo="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int, b: List[Int]) = {
	([StringEntry("a", "a")], unit)
}
`,
			false, "BgIICAISBAoCAREAAQFpAQR0ZXN0AgFhAWIJAJQKAgkAzAgCCQELU3RyaW5nRW50cnkCAgFhAgFhBQNuaWwFBHVuaXQAZIKXrA=="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int|String) = {
	([StringEntry("a", "a")], unit)
}
`,
			true, "(7:1, 10:0): Unexpected type in callable args : Int|String"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: List[Int|String]) = {
	([StringEntry("a", "a")], unit)
}
`,
			true, "(7:1, 10:0): Unexpected type in callable args : List[Int|String]"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int) = {
	(10, unit)
}
`,
			true, "(7:1, 10:0): CallableFunc must return (List[BinaryEntry|BooleanEntry|Burn|DeleteEntry|IntegerEntry|Issue|Lease|LeaseCancel|Reissue|ScriptTransfer|SponsorFee|StringEntry], Any)|List[BinaryEntry|BooleanEntry|Burn|DeleteEntry|IntegerEntry|Issue|Lease|LeaseCancel|Reissue|ScriptTransfer|SponsorFee|StringEntry], but return (Int, Unit)"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test() = {
	[StringEntry("a", "a")]
}
`,
			false, "BgIECAISAAABAWkBBHRlc3QACQDMCAIJAQtTdHJpbmdFbnRyeQICAWECAWEFA25pbAD/5CEG"},
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int|String) = {
	([StringEntry("a", "a")], unit)
}
`,
			false, "AAIFAAAAAAAAAAcIAhIDCgEJAAAAAAAAAAEAAAABaQEAAAAEdGVzdAAAAAEAAAABYQkABRQAAAACCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAICAAAAAWECAAAAAWEFAAAAA25pbAUAAAAEdW5pdAAAAAAR1sWX"},
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int|List[Int]) = {
	([StringEntry("a", "a")], unit)
}
`,
			true, "(7:1, 10:0): Unexpected type in callable args : List[Int]"},
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: List[Int|String]) = {
	([StringEntry("a", "a")], unit)
}
`,
			false, "AAIFAAAAAAAAAAcIAhIDCgEZAAAAAAAAAAEAAAABaQEAAAAEdGVzdAAAAAEAAAABYQkABRQAAAACCQAETAAAAAIJAQAAAAtTdHJpbmdFbnRyeQAAAAICAAAAAWECAAAAAWEFAAAAA25pbAUAAAAEdW5pdAAAAADfRbxk"},
		{`
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: Int) = {
	WriteSet([DataEntry("a", a)])
}
`,
			false, "AAIDAAAAAAAAAAcIARIDCgEBAAAAAAAAAAEAAAABaQEAAAAEdGVzdAAAAAEAAAABYQkBAAAACFdyaXRlU2V0AAAAAQkABEwAAAACCQEAAAAJRGF0YUVudHJ5AAAAAgIAAAABYQUAAAABYQUAAAADbmlsAAAAACP3g+E="},
	} {
		rawAST, buf, err := buildAST(t, test.code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			assert.Equal(t, tree.Declarations, astParser.Tree.Declarations)
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}

func TestSimpleAST(t *testing.T) {
	src := `{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
func sum(accum: Int, next: Int) = accum + next
let arr = [1,2,3,4,5]
let a = FOLD<5>(arr, 0, sum)
`
	ast, buf, err := buildAST(t, src, false)

	require.NoError(t, err)
	astParser := NewASTParser(ast, buf)
	astParser.Parse()
	for _, err := range astParser.ErrorsList {
		fmt.Println(err.Error())
	}
}

func TestBigScript(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

#
# Main Smart Contract of Neutrino Protocol
# Implemented actions: Swap, Bond Liquidation, Leasing
#
let revisionNum = ""

#-------------------Base functions----------------------
func getStringOrFail(address: Address, key: String)  = address.getString(key).valueOrErrorMessage(makeString(["mandatory ", address.toString(), ".", key, " is not defined"], ""))

# workaround to reduce size
func lcalc(l: Lease) = {
  calculateLeaseId(l)
}

func getNumberByKey(key: String) = {
    getInteger(this, key).valueOrElse(0)
}
func getStringByKey(key: String) = {
    getString(this, key).valueOrElse("")
}
func getBoolByKey(key: String) = {
    getBoolean(this, key).valueOrElse(false)
}
func getNumberByAddressAndKey(address: Address, key: String) = {
    getInteger(address, key).valueOrElse(0)
}
func getStringByAddressAndKey(address: String, key: String) = {
     getString(addressFromStringValue(address), key).valueOrElse("")
}
func getBoolByAddressAndKey(address: Address, key: String) = {
     getBoolean(address, key).valueOrElse(false)
}
func asAnyList(v: Any) = {
  match v {
    case l: List[Any] => l
    case _ => throw("fail to cast into List[Any]")
  }
}
func asString(v: Any) = {
  match v {
    case s: String => s
    case _ => throw("fail to cast into String")
  }
}
func asInt(v: Any) = {
  match v {
    case i: Int => i
    case _ => throw("fail to cast into Int")
  }
}
func asBytes(val: Any) = {
  match val {
    case valByte: ByteVector => valByte
    case _ => throw("fail to cast into ByteVector")
  }
}
func asPayment(v: Any) = {
  match v {
    case p: AttachedPayment => p
    case _ => throw("fail to cast into AttachedPayment")
  }
}

func asSwapParamsSTRUCT(v: Any) = {
  match v {
    case struct: (Int, Int, Int, Int, Int, Int, Int) => struct
    case _ => throw("fail to cast into Tuple5 ints")
  }
}

#-------------------Constants---------------------------
let SEP = "__"
let LISTSEP = ":"
let WAVELET = 100000000
let PAULI = 1000000
let PRICELET = 1000000 # 10^6
let DEFAULTSWAPFEE = 20000 # 0.02 * 1000000 or 2%
let BRPROTECTED = 100000 # if BR <= 10% then use SURF during swap USDN->WAVES

let IdxNetAmount = 0
let IdxFeeAmount = 1
let IdxGrossAmount = 2

# data indices from controlConfig
let IdxControlCfgNeutrinoDapp     = 1
let IdxControlCfgAuctionDapp      = 2
let IdxControlCfgRpdDapp          = 3
let IdxControlCfgMathDapp         = 4
let IdxControlCfgLiquidationDapp  = 5
let IdxControlCfgRestDapp         = 6
let IdxControlCfgNodeRegistryDapp = 7
let IdxControlCfgNsbtStakingDapp  = 8
let IdxControlCfgMediatorDapp     = 9
let IdxControlCfgSurfStakingDapp  = 10
let IdxControlCfgGnsbtControllerDapp = 11
let IdxControlCfgRestV2Dapp       = 12
let IdxControlCfgGovernanceDapp   = 13

func keyControlAddress() = "%s%s__config__controlAddress"
func keyControlCfg()     = "%s__controlConfig"

func readControlCfgOrFail(control: Address) = split_4C(control.getStringOrFail(keyControlCfg()), SEP)
func getContractAddressOrFail(controlCfg: List[String], idx: Int) = controlCfg[idx].addressFromString()
  .valueOrErrorMessage("Control cfg doesn't contain address at index " + idx.toString())

# GLOBAL VARIABLES
let controlContract = this.getString(keyControlAddress()).valueOrElse("3P5Bfd58PPfNvBM2Hy8QfbcDqMeNtzg7KfP").addressFromStringValue()
let controlCfg = controlContract.readControlCfgOrFail()
let mathContract = controlCfg.getContractAddressOrFail(IdxControlCfgMathDapp)
let nsbtStakingContract = controlCfg.getContractAddressOrFail(IdxControlCfgNsbtStakingDapp)
let surfStakingContract = controlCfg.getContractAddressOrFail(IdxControlCfgSurfStakingDapp)
let gnsbtControllerContract = controlCfg.getContractAddressOrFail(IdxControlCfgGnsbtControllerDapp)
let auctionContract = controlCfg.getContractAddressOrFail(IdxControlCfgAuctionDapp)
let nodeRegistryContract = controlCfg.getContractAddressOrFail(IdxControlCfgNodeRegistryDapp)
let govContract = controlCfg.getContractAddressOrFail(IdxControlCfgGovernanceDapp)

#-------------------Constructor-------------------------
let NeutrinoAssetIdKey = "neutrino_asset_id"
let BondAssetIdKey = "bond_asset_id"
let AuctionContractKey = "auction_contract"
let NsbtStakingContractKey = "nsbtStakingContract"
let LiquidationContractKey = "liquidation_contract"
let RPDContractKey = "rpd_contract"
let ContolContractKey = "control_contract"
let MathContractKey = "math_contract"
let BalanceWavesLockIntervalKey = "balance_waves_lock_interval"
let BalanceNeutrinoLockIntervalKey = "balance_neutrino_lock_interval"
let MinWavesSwapAmountKey = "min_waves_swap_amount"
let MinNeutrinoSwapAmountKey = "min_neutrino_swap_amount"
let NodeOracleProviderPubKeyKey = "node_oracle_provider"
let NeutrinoOutFeePartKey = "neutrinoOut_swap_feePart"
let WavesOutFeePartKey = "wavesOut_swap_feePart"

#------Common----------------

#---Nodes Registry contract--
func keyNodeRegistry(address: String)       = "%s__" + address

#------Control contract-------
let PriceKey = "price"

let PriceIndexKey = "price_index"
let IsBlockedKey = "is_blocked"
func getPriceHistoryKey(block: Int) = PriceKey + "_" + toString(block)
func getHeightPriceByIndexKey(index: Int) = PriceIndexKey + "_" + toString(index)
func getStakingNodeByIndex(idx: Int) = getStringByKey(makeString(["%s%d%s", "lease", toString(idx), "nodeAddress"], SEP))
func getStakingNodeAddressByIndex(idx: Int) = addressFromStringValue(getStakingNodeByIndex(idx))

func getReservedAmountForSponsorship() =
    getInteger(this, makeString(["%s%s", "lease", "sponsorshipWavesReserve"], SEP)).valueOrElse(1000 * WAVELET)

#------This contract----------
#-------------------Keys-------------------
# TODO need to move into zero
func getBalanceUnlockBlockKey(owner: String)               = "balance_unlock_block_" + owner
func getLeaseIdKey(nodeIndex: Int)                         = makeString(["%s%d%s", "lease", toString(nodeIndex), "id"], SEP)
func getLeaseIdByAddressKey(nodeAddress: String)           = makeString(["%s%s%s", "leaseByAddress", nodeAddress, "id"], SEP)
func getLeaseAmountKey(nodeIndex: Int)                     = makeString(["%s%d%s", "lease", toString(nodeIndex), "amount"], SEP)
func getLeaseAmountByAddressKey(nodeAddress: String)       = makeString(["%s%s%s", "leaseByAddress", nodeAddress, "amount"], SEP)
func getLeaseGroupNodeListKey(groupNum: Int)               = makeString(["%s%d%s", "leaseGroup", groupNum.toString(), "nodeList"], SEP)

func minSwapAmountKEY(swapType: String)                    = "min_" + swapType + "_swap_amount"
func totalLockedKEY(swapType: String)                      = "balance_lock_" + swapType
func totalLockedByUserKEY(swapType: String, owner: String) = makeString(["balance_lock", swapType, owner], "_")
func balanceLockIntervalKEY(swapType: String)              = "balance_" + swapType + "_lock_interval" # number of blocks after user could withdraw funds
func nodeBalanceLockIntervalKEY()                          = "balance_node_lock_interval"
func outFeePartKEY(swapType: String)                       = swapType + "Out_swap_feePart"
func swapsTimeframeKEY()                                   = "swaps_timeframe"
func brProtectedKEY()                                      = "min_BR_protection_level"

#-------------------State Reading functions-------------------
func minSwapAmountREAD(swapType: String) = this.getInteger(minSwapAmountKEY(swapType)).valueOrElse(0)
func swapsTimeframeREAD() = this.getInteger(swapsTimeframeKEY()).valueOrElse(1440)
func totalLockedREAD(swapType: String) = this.getInteger(totalLockedKEY(swapType)).valueOrElse(0)
func totalLockedByUserREAD(swapType: String, owner: String) = this.getInteger(totalLockedByUserKEY(swapType, owner)).valueOrElse(0)
func balanceLockIntervalREAD(swapType: String) = this.getInteger(balanceLockIntervalKEY(swapType)).valueOrElse(1440) # number og blocks after user could withdraw funds
func nodeBalanceLockIntervalREAD() = this.getInteger(nodeBalanceLockIntervalKEY()).valueOrElse(1)
func keySwapUserSpentInPeriod(userAddress: String) = ["%s%s", "swapUserSpentInPeriod", userAddress].makeString(SEP)
func keyUserLastSwapHeight(userAddress: String) = ["%s%s", "userLastSwapHeight", userAddress].makeString(SEP)

#-------------------Convert functions-------------------
func convertNeutrinoToWaves(amount: Int, price: Int) = fraction(fraction(amount, PRICELET, price),WAVELET, PAULI)
func convertWavesToNeutrino(amount: Int, price: Int) = fraction(fraction(amount, price, PRICELET), PAULI, WAVELET)
func convertWavesToBond(amount: Int, price: Int) = convertWavesToNeutrino(amount, price) # it's here to be more explicit with convertation
func convertJsonArrayToList(jsonArray: String) = {
   jsonArray.split(",")
}

#-------------------Failures-------------------
func minSwapAmountFAIL(swapType: String, minSwapAmount: Int) = throw("The specified amount in " + swapType + " swap is less than the required minimum of " + toString(minSwapAmount))
func emergencyShutdownFAIL() = throw("contract is blocked by EMERGENCY SHUTDOWN actions untill reactivation by emergency oracles")

func priceIndexFAIL(index: Int, priceIndex: Int, indexHeight: Int, unlockHeight: Int, prevIndexHeight: Int) =
            throw("invalid price history index: index=" + toString(index)
                + " priceIndex=" + toString(priceIndex)
                + " indexHeight=" + toString(indexHeight)
                + " unlockHeight=" + toString(unlockHeight)
                + " prevIndexHeight=" + toString(prevIndexHeight))

#-------------------Global vars-------------------------

let neutrinoAssetId = getStringByKey(NeutrinoAssetIdKey).fromBase58String()
let priceIndex = getNumberByAddressAndKey(controlContract, PriceIndexKey) # Last price history iterator from control.ride
let isBlocked = getBoolByAddressAndKey(controlContract, IsBlockedKey) # Checks for contract locks that might happen after attacks.  The var is read from control contract
let nodeOracleProviderPubKey = fromBase58String(getStringByKey(NodeOracleProviderPubKeyKey))
let bondAssetId = fromBase58String("6nSpVyNH7yM69eg446wrQR94ipbbcmZMU1ENPwanC97g") # NSBT with 6 decimals as USDN does
let deprecatedBondAssetId = fromBase58String("975akZBfnMj513U7MZaHKzQrmsEx5aE3wdWKTrHBhbjF") # USDNB with 0 decimals

let neutrinoContract = this
#-------------------Global vars deficit, locked & supply -------------------------
let currentPrice = getNumberByAddressAndKey(controlContract, PriceKey) # The value from control.ride

#-------------------Verifier Functions----------------------
func checkIsValidMinSponsoredFee(tx: SponsorFeeTransaction) = {
    let MINTRANSFERFEE = 100000 #wavelets (to support smart assets)
    let SponsoredFeeUpperBound = 1000 # % of fee profits higther than real fee for transfer
    let realNeutrinoFee = convertWavesToNeutrino(MINTRANSFERFEE, currentPrice) # in paulis
    let minNeutrinoFee = realNeutrinoFee * 2 # 100%
    let maxNeutrinoFee = fraction(realNeutrinoFee, SponsoredFeeUpperBound, 100)

    let inputFee = tx.minSponsoredAssetFee.value()

    inputFee >= minNeutrinoFee && inputFee <= maxNeutrinoFee && tx.assetId == neutrinoAssetId
}

#------Control contract------
# The func is reading price from control.ride price history
func getPriceHistory(block: Int) = getNumberByAddressAndKey(controlContract, getPriceHistoryKey(block))
# The func is reading from control.ride price history heights
func getHeightPriceByIndex(index: Int) = getNumberByAddressAndKey(controlContract, getHeightPriceByIndexKey(index))

#------NSBT Staking contract------
func keyLockParamUserAmount(userAddress: String) = ["%s%s%s", "paramByUser", userAddress, "amount"].makeString(SEP)


#------This contract---------
let sIdxSwapType                 = 1
let sIdxStatus                   = 2
let sIdxInAmount                 = 3
let sIdxStartHeight              = 7
let sIdxStartTimestamp           = 8
let sIdxSelfUnlockHeight         = 11
let sIdxMinRand                  = 15
let sIdxMaxRand                  = 16

func swapKEY(userAddress: String, txId: String) = {
  makeString(["%s%s", userAddress, txId], SEP)
}

func strSwapDATA(swapType: String, status: String, inAmount: String, price: String, outNetAmount: String, outFeeAmount: String,
                 startHeight: String, startTimestamp: String, endHeight: String, endTimestamp: String,
                 selfUnlockHeight: String, randUnlockHeight: String, index: String, withdrawTxId: String,
                 randMin: String, randMax: String, outSurfAmt: String, br: String) = {
  makeString(["%s%s%d%d%d%d%d%d%d%d%d%d%d%s%d%d%d%d",
      swapType,                     # 1
      status,                       # 2
      inAmount,                     # 3
      price,                        # 4
      outNetAmount,                 # 5
      outFeeAmount,                 # 6
      startHeight,                  # 7
      startTimestamp,               # 8
      endHeight,                    # 9
      endTimestamp,                 # 10
      selfUnlockHeight,             # 11
      randUnlockHeight,             # 12
      index,                        # 13
      withdrawTxId,                 # 14
      randMin,                      # 15
      randMax,                      # 16
      outSurfAmt,                   # 17
      br                            # 18
      ],
  SEP)
}

func pendingSwapDATA(swapType: String, inAssetAmount: Int, selfUnlockHeight: Int) = {
  strSwapDATA(
      swapType,                       # 1
      "PENDING",                      # 2
      inAssetAmount.toString(),       # 3
      "0",                            # 4
      "0",                            # 5
      "0",                            # 6
      height.toString(),              # 7
      lastBlock.timestamp.toString(), # 8
      "0",                            # 9
      "0",                            # 10
      selfUnlockHeight.toString(),    # 11
      "0",                            # 12
      "0",                            # 13
      "NULL",                         # 14
      "0",                            # 15
      "0",                            # 16
      "0",                            # 17
      "0"                             # 18
  )
}

func finishSwapDATA(dataArray: List[String], price: Int, outNetAmount: Int, outFeeAmount: Int, randUnlockHeight: Int,
                    index: Int, withdrawTxId: String, outSurfAmt: Int, br: Int) = {
  strSwapDATA(
      dataArray[sIdxSwapType],        # 1
      "FINISHED",                     # 2
      dataArray[sIdxInAmount],        # 3
      price.toString(),               # 4
      outNetAmount.toString(),        # 5
      outFeeAmount.toString(),        # 6
      dataArray[sIdxStartHeight],     # 7
      dataArray[sIdxStartTimestamp],  # 8
      height.toString(),              # 9
      lastBlock.timestamp.toString(), # 10
      dataArray[sIdxSelfUnlockHeight],# 11
      randUnlockHeight.toString(),    # 12
      index.toString(),               # 13
      withdrawTxId,                   # 14
      dataArray[sIdxMinRand],         # 15
      dataArray[sIdxMaxRand],         # 16
      outSurfAmt.toString(),          # 17
      br.toString()                   # 18
  )
}

func swapDataFailOrREAD(userAddress: String, swapTxId: String) = {
  let swapKey = swapKEY(userAddress, swapTxId)
  this.getString(swapKey)
    .valueOrErrorMessage("no swap data for " + swapKey)
    .split(SEP)
}

func applyFees(amountOutGross: Int, inAmtToSURF: Int, feePart: Int) = {
  let feeAmount = fraction(amountOutGross, feePart, PAULI)
  [amountOutGross - feeAmount, feeAmount]
}

func abs(x: Int) = if (x < 0) then -x else x

func selectNode(unleaseAmount: Int) = {
    let amountToLease = wavesBalance(neutrinoContract).available - unleaseAmount - getReservedAmountForSponsorship()

    let oldLeased0 = getNumberByKey(getLeaseAmountKey(0))
    let oldLeased1 = getNumberByKey(getLeaseAmountKey(1))
    let newLeased0 = amountToLease + oldLeased0
    let newLeased1 = amountToLease + oldLeased1

    if (newLeased0 > 0 || newLeased1 > 0) then {
        # balancing the nodes
        let delta0 = abs(newLeased0 - oldLeased1)
        let delta1 = abs(newLeased1 - oldLeased0)
        # 0 node is a priority
        if (delta0 <= delta1) then (0, newLeased0) else (1, newLeased1)
    } else (-1, 0)
}

func thisOnly(i: Invocation) = {
  if (i.caller != this) then {
    throw("Permission denied: this contract only allowed")
  } else true
}

# prepare list of actions to lease available waves or cancel lease in case of usdn2waves swap
func prepareUnleaseAndLease(unleaseAmount: Int) = {
    let nodeTuple       = selectNode(unleaseAmount) # balancing waves by 2 nodes
    let nodeIndex       = nodeTuple._1
    let newLeaseAmount  = nodeTuple._2

    if (newLeaseAmount > 0) then {
        let leaseIdKey = getLeaseIdKey(nodeIndex)
        let oldLease = getBinary(this, leaseIdKey)
        let unleaseOrEmpty = if (oldLease.isDefined()) then [LeaseCancel(oldLease.value())] else []
        let leaseAmountKey = getLeaseAmountKey(nodeIndex)
        let lease = Lease(getStakingNodeAddressByIndex(nodeIndex), newLeaseAmount)

        unleaseOrEmpty ++ [
            lease,
            BinaryEntry(leaseIdKey, lcalc(lease)),
            IntegerEntry(getLeaseAmountKey(nodeIndex), newLeaseAmount)]
    } else []
}

func readNodeInfo(nodeIdx: Int) = {
  let nodeAddress = getStakingNodeAddressByIndex(nodeIdx)
  let leasedAmtKEY = getLeaseAmountKey(nodeIdx)
  let leasedAmt = leasedAmtKEY.getNumberByKey()

  let leaseIdKEY = getLeaseIdKey(nodeIdx)
  let leaseId = this.getBinary(leaseIdKEY).value()

  (nodeAddress, leasedAmtKEY, leasedAmt, leaseIdKEY, leaseId)
}

#-------------------MAIN LOGIC----------------------

func commonSwap(swapType: String, pmtAmount: Int, userAddressStr: String, txId58: String, swapParamsByUserSYSREADONLY: (Int,Int,Int,Int,Int,Int,Int)) = {
  let swapLimitSpent    = swapParamsByUserSYSREADONLY._2
  let blcks2LmtReset    = swapParamsByUserSYSREADONLY._3
  let wavesSwapLimitMax = swapParamsByUserSYSREADONLY._6
  let usdnSwapLimitMax  = swapParamsByUserSYSREADONLY._7

  let minSwapAmount         = minSwapAmountREAD(swapType)
  let totalLocked           = totalLockedREAD(swapType)
  let totalLockedByUser     = totalLockedByUserREAD(swapType, userAddressStr)
  let nodeAddress           = getStakingNodeByIndex(0)
  let priceByIndex          = priceIndex.getHeightPriceByIndex().getPriceHistory()
  let isSwapByNode          = nodeAddress == userAddressStr

  let balanceLockMaxInterval = if (isSwapByNode) then nodeBalanceLockIntervalREAD() else balanceLockIntervalREAD(swapType)
  let selfUnlockHeight       = height + balanceLockMaxInterval
  let swapUsdnVolume         = if (swapType == "neutrino") then pmtAmount else pmtAmount.convertWavesToNeutrino(priceByIndex)
  let swapLimitMax           = if (swapType == "neutrino") then usdnSwapLimitMax else wavesSwapLimitMax.convertWavesToNeutrino(priceByIndex)

  if (pmtAmount < minSwapAmount) then minSwapAmountFAIL(swapType, minSwapAmount) else
  if (!isSwapByNode && swapLimitSpent > 0) then throw("You have exceeded swap limit! Next allowed swap height is " + (height + blcks2LmtReset).toString()) else
  if (!isSwapByNode && swapUsdnVolume > swapLimitMax) then throw("You have exceeded your swap limit! Requested: "+ toString(swapUsdnVolume) + ", available: " + toString(swapLimitMax)) else
  if (isBlocked) then emergencyShutdownFAIL() else  # see control.ride

  let leasePart = if (swapType == "waves") then prepareUnleaseAndLease(0) else []

  ([
      IntegerEntry(keySwapUserSpentInPeriod(userAddressStr), swapUsdnVolume),
      IntegerEntry(keyUserLastSwapHeight(userAddressStr), height),
      IntegerEntry(totalLockedByUserKEY(swapType, userAddressStr), totalLockedByUser + pmtAmount),
      IntegerEntry(getBalanceUnlockBlockKey(userAddressStr), selfUnlockHeight),
      IntegerEntry(totalLockedKEY(swapType), totalLocked + pmtAmount),
      StringEntry(
        swapKEY(userAddressStr, txId58),
        pendingSwapDATA(swapType, pmtAmount, selfUnlockHeight))
    ] ++ leasePart, unit)

}

#indices for calcNeutinoMetricsREADONLY result array
let nMetricIdxPrice = 0
let nMetricIdxUsdnLockedBalance = 1
let nMetricIdxWavesLockedBalance = 2
let nMetricIdxReserve = 3
let nMetricIdxReserveInUsdn = 4
let nMetricIdxUsdnSupply = 5
let nMetricIdxSurplus = 6
let nMetricIdxSurplusPercent = 7
let nMetricIdxBR = 8 # BR with 6 decimals
let nMetricIdxNsbtSupply = 9
let nMetricIdxMaxNsbtSupply = 10
let nMetricIdxSurfSupply = 11

# surfFunctionREADONLY result array indices
let bFuncIdxSurf = 0
let bFuncIdxWaves = 1
let bFuncIdxUsdn = 2
let bFuncIdxReserveStart = 3
let bFuncIdxSupplyStart = 4
let bFuncIdxBRStart = 5
let bFuncIdxReserveEnd = 6
let bFuncIdxSupplyEnd = 7
let bFuncIdxBREnd = 8
let bFuncIdxRest = 9
let bFuncIdxWavesPrice = 10

func calcWithdrawW2U(wavesIn: Int, price: Int) = {
  let outAmtGross = convertWavesToNeutrino(wavesIn, price)
  (
    outAmtGross,      # gross outAmount (fees are not applied yet)
    neutrinoAssetId,  # outAssetId is USDN
    0,                # part of inAmount that is converted into SURF to protect BR
    unit,             # inAssetId is WAVES
    0,                # amount to unlease
    wavesIn,          # debug - part of inAmount that is swapped into out asset
    0,                # debug - max allowed usdn amount to reach BR protection level
    0,                # debug - part of inAmount that is used BEFORE reaching BR protection level
    0                 # debug - part of inAmount that is used AFTER reaching BR protection level
  )
}

func calcWithdrawU2W(usdnIn: Int, price: Int, br: Int, reservesInUsdn: Int, usdnSupply: Int) = {
  let brProtected       = this.getInteger(brProtectedKEY()).valueOrElse(BRPROTECTED)

  let maxAllowedUsdnBeforeMinBr = if (br <= brProtected) then 0 else {
    fraction(reservesInUsdn - fraction(brProtected, usdnSupply, PAULI), PAULI, PAULI - brProtected)
  }

  let allowedUsdnBeforeMinBr =
      if (usdnIn > maxAllowedUsdnBeforeMinBr) then maxAllowedUsdnBeforeMinBr else usdnIn

  let allowedUsdnAfterMinBr =
      if (usdnIn > maxAllowedUsdnBeforeMinBr) then fraction(usdnIn - maxAllowedUsdnBeforeMinBr, br, PAULI) else 0

  let allowedUsdn = allowedUsdnBeforeMinBr + allowedUsdnAfterMinBr
  let usdn2SURF = usdnIn - allowedUsdn

  let outAmtGross = convertNeutrinoToWaves(allowedUsdn, price)
  (
    outAmtGross,                # gross outAmount (fees are not applied yet)
    unit,                       # waves_id
    usdn2SURF,                  # part of inAmount that is converted into SURF to protect BR
    neutrinoAssetId,            # inAssetId is WAVES
    outAmtGross,                # amount to unlease
    allowedUsdn,                # debug - part of inAmount that is swapped into out asset
    maxAllowedUsdnBeforeMinBr,  # debug - max allowed usdn amount to reach BR protection level
    allowedUsdnBeforeMinBr,     # debug - part of inAmount that is used BEFORE reaching BR protection level
    allowedUsdnAfterMinBr       # debug - part of inAmount that is used AFTER reaching BR protection level
  )
}

func calcWithdraw(swapType: String, inAmount: Int, price: Int, neutrinoMetrics: List[Any]) = {
  let outFeePart        = this.getInteger(outFeePartKEY(swapType)).valueOrElse(DEFAULTSWAPFEE)
  if (outFeePart < 0 || outFeePart >= PAULI) then throw("invalid outFeePart config for " + swapType + " swap: outFeePart=" + outFeePart.toString()) else

  let brProtected       = this.getInteger(brProtectedKEY()).valueOrElse(BRPROTECTED)

  let BR                = neutrinoMetrics[nMetricIdxBR].asInt()
  let reservesInUsdn    = neutrinoMetrics[nMetricIdxReserveInUsdn].asInt()
  let usdnSupply        = neutrinoMetrics[nMetricIdxUsdnSupply].asInt()

  let outDataTuple =
    if (swapType == "waves")    then calcWithdrawW2U(inAmount, price) else
    if (swapType == "neutrino") then calcWithdrawU2W(inAmount, price, BR, reservesInUsdn, usdnSupply)
    else throw("Unsupported swap type " + swapType)

  let outAmtGross       = outDataTuple._1
  let outAssetId        = outDataTuple._2
  let inAmtToSurfPart   = outDataTuple._3
  let inAssetId         = outDataTuple._4
  let unleaseAmt        = outDataTuple._5

  let payoutsArray = applyFees(outAmtGross, inAmtToSurfPart, outFeePart)
  let outNetAmt = payoutsArray[IdxNetAmount]
  let outFeeAmt = payoutsArray[IdxFeeAmount]

  let outSurfAmt = if (inAmtToSurfPart <= 0) then 0 else {
    let surfResult = mathContract.invoke("surfFunctionREADONLY", [inAmtToSurfPart, inAssetId], []).asAnyList()
    surfResult[bFuncIdxSurf].asInt()
  }

  # WARNING: if u modify then need to check RestV2
  (outNetAmt, outAssetId, outSurfAmt, inAmtToSurfPart, unleaseAmt, outFeeAmt, outAmtGross)
}

# TODO move everything into withdraw - no need to keep separate function
func commonWithdraw(account : String, index: Int, swapTxId: String, withdrawTxId: String, neutrinoMetrics: List[Any]) = {
    let userAddress = addressFromStringValue(account)

    let dataArray         = swapDataFailOrREAD(account, swapTxId)
    let selfUnlockHeight  = dataArray[sIdxSelfUnlockHeight].parseIntValue()
    let swapType          = dataArray[sIdxSwapType]
    let inAmount          = dataArray[sIdxInAmount].parseIntValue()
    let swapStatus        = dataArray[sIdxStatus]
    let startHeight       = dataArray[sIdxStartHeight].parseIntValue()

    let outFeePart        = this.getInteger(outFeePartKEY(swapType)).valueOrElse(DEFAULTSWAPFEE)
    let totalLocked       = totalLockedREAD(swapType)
    let totalLockedByUser = totalLockedByUserREAD(swapType, account)

    let unlockHeight = selfUnlockHeight

    let indexHeight = getHeightPriceByIndex(index)
    let prevIndexHeight = getHeightPriceByIndex(index-1)
    let priceByIndex = getPriceHistory(indexHeight)

    if (isBlocked) then emergencyShutdownFAIL() else
    if (swapStatus != "PENDING") then throw("swap has been already processed") else
    if (unlockHeight > height) then throw("please wait for: " + toString(unlockHeight) + " block height to withdraw funds") else
    if (index > priceIndex
          || indexHeight < unlockHeight
          || (prevIndexHeight != 0 && unlockHeight <= prevIndexHeight)) then priceIndexFAIL(index, priceIndex, indexHeight, unlockHeight, prevIndexHeight) else

    let withdrawTuple = calcWithdraw(swapType, inAmount, priceByIndex, neutrinoMetrics)
    let outNetAmount    = withdrawTuple._1
    let outAssetId      = withdrawTuple._2
    let outSurfAmt      = withdrawTuple._3
    #let inAmtToSurfPart = withdrawTuple._4
    let unleaseAmt      = withdrawTuple._5
    let outFeeAmount    = withdrawTuple._6
    let outAmtGross     = withdrawTuple._7

    if (outAmtGross <= 0) then throw("balance equals zero") else

    let BR = neutrinoMetrics[nMetricIdxBR].asInt()
    let state = [
      IntegerEntry(totalLockedByUserKEY(swapType, account), totalLockedByUser - inAmount),
      IntegerEntry(totalLockedKEY(swapType), totalLocked - inAmount),
      ScriptTransfer(userAddress, outNetAmount, outAssetId),
      StringEntry(
        swapKEY(account, swapTxId),
        finishSwapDATA(dataArray, priceByIndex, outNetAmount, outFeeAmount, unlockHeight, index, withdrawTxId, outSurfAmt, BR))
    ]

    strict surfCondition = if (outSurfAmt > 0) then {
       strict issueResult = auctionContract.invoke("issueSurf", [outSurfAmt, account], [])
       0
    } else 0

    (state, AttachedPayment(outAssetId, outFeeAmount), unleaseAmt)
}

# governance contract
func keyApplyInProgress() = "%s__applyInProgress"
func keyProposalDataById(proposalId: Int) = "%s%d__proposalData__" + proposalId.toString()

# indices to access proposal data fields (static)
let govIdxTxIds = 9

# The transaction cannot be added to the blockchain if the timestamp value is more than 2 hours behind 
# or 1.5 hours ahead of current block timestamp
func validateUpdate(tx: Transaction|Order) = {
    match(tx) {
        case o: Order => throw("Orders aren't allowed")
        case t: Transaction => {
            let txId = toBase58String(t.id)
            let proposalId = govContract.getInteger(keyApplyInProgress()).valueOrErrorMessage("Apply is not happening")
            let txList = govContract.getStringOrFail(keyProposalDataById(proposalId)).split(SEP)[govIdxTxIds].split(LISTSEP)
            if (!txList.indexOf(txId).isDefined()) then throw("Unknown txId: " + txId + " for proposalId=" + proposalId.toString()) else

            true
        }
    }
}

#-------------------Callable----------------------

@Callable(i)
func constructor(
  neutrinoAssetIdPrm: String,
  bondAssetIdPrm: String,
  auctionContractPrm: String,
  liquidationContractPrm: String,
  rpdContractPrm: String,
  nodeOracleProviderPubKeyPrm: String,
  balanceWavesLockIntervalPrm: Int,
  balanceNeutrinoLockIntervalPrm: Int,
  minWavesSwapAmountPrm: Int,
  minNeutrinoSwapAmountPrm: Int,
  neutrinoOutFeePartPrm: Int,
  wavesOutFeePartPrm: Int) = {

  strict checkCaller = i.thisOnly()
  if (i.payments.size() != 0) then throw("no payments allowed") else

  [
    StringEntry(NeutrinoAssetIdKey, neutrinoAssetIdPrm),
    StringEntry(BondAssetIdKey, bondAssetIdPrm),
    StringEntry(AuctionContractKey, auctionContractPrm), # ignored
    StringEntry(LiquidationContractKey, liquidationContractPrm), # ignored
    StringEntry(RPDContractKey, rpdContractPrm), #ignored
    StringEntry(NodeOracleProviderPubKeyKey, nodeOracleProviderPubKeyPrm),
    IntegerEntry(BalanceWavesLockIntervalKey, balanceWavesLockIntervalPrm),
    IntegerEntry(BalanceNeutrinoLockIntervalKey, balanceNeutrinoLockIntervalPrm),
    IntegerEntry(MinWavesSwapAmountKey, minWavesSwapAmountPrm),
    IntegerEntry(MinNeutrinoSwapAmountKey, minNeutrinoSwapAmountPrm),
    IntegerEntry(NeutrinoOutFeePartKey, neutrinoOutFeePartPrm),
    IntegerEntry(WavesOutFeePartKey, wavesOutFeePartPrm)
  ]
}

@Callable(i)
func constructorV2(mathContract: String, nsbtStakingContract: String, swapsTimeframeBlocks: Int) = {
  strict checkCaller = i.thisOnly()
  if (i.payments.size() != 0) then throw("no payments allowed") else
  [
    StringEntry(MathContractKey, mathContract),
    StringEntry(NsbtStakingContractKey, nsbtStakingContract),
    IntegerEntry(swapsTimeframeKEY(), swapsTimeframeBlocks)
  ]
}

# Instant swap of WAVES to Neutrino token at the current price on the smart contract
# [called by user]
@Callable(i)
func swapWavesToNeutrino() = {
    if (i.payments.size() != 1) then throw("swapWavesToNeutrino require only one payment") else
    let pmt = i.payments[0].value()
    if (isDefined(pmt.assetId)) then throw("Only Waves token is allowed for swapping.") else

    let userAddress = i.caller.toString()
    let txId58 = i.transactionId.toBase58String()

    let swapParamsSTRUCT = this.invoke("swapParamsByUserSYSREADONLY", [userAddress, 0], []).asSwapParamsSTRUCT()

    let commonSwapResult =  commonSwap("waves", pmt.amount, userAddress, txId58, swapParamsSTRUCT)
    commonSwapResult
}

# Swap request of Neutrino to WAVES. After {balanceLockInterval} blocks, WAVES tokens will be available for withdrawal
# via {withdraw(account : String)} method at the price that is current at the time when {balanceLockInterval} is reached
# [called by user]
@Callable(i)
func swapNeutrinoToWaves() = {
    if (i.payments.size() != 1) then throw("swapNeutrinoToWaves require only one payment") else
    let pmt = i.payments[0].value()
    if (pmt.assetId != neutrinoAssetId) then throw("Only appropriate Neutrino tokens are allowed for swapping.") else

    let userAddress = i.caller.toString()
    let txId58 = i.transactionId.toBase58String()

    let swapParamsSTRUCT = this.invoke("swapParamsByUserSYSREADONLY", [userAddress, 0], []).asSwapParamsSTRUCT()

    let commonSwapResult = commonSwap("neutrino", pmt.amount, userAddress, txId58, swapParamsSTRUCT)
    commonSwapResult
}

# Withdraw WAVES from smart contract after {swapNeutrinoToWaves()} request has reached {balanceLockInterval} height
# at the price that is current at the time when {balanceLockInterval} is reached
# [called by user]
@Callable(i)
func withdraw(account: String, index: Int, swapTxId: String) = {
    let txId = i.transactionId.toBase58String()
    if (i.payments.size() != 0) then throw("no payments allowed") else

    let neutrinoMetrics = mathContract.invoke("calcNeutinoMetricsREADONLY", [], []).asAnyList()
    let BR = neutrinoMetrics[nMetricIdxBR].asInt()

    let commonTuple = commonWithdraw(account, index, swapTxId, txId, neutrinoMetrics)
    let state       = commonTuple._1
    let fee         = commonTuple._2
    let unleaseAmt  = commonTuple._3

    strict unleaseInvOrEmpty = this.invoke("internalUnleaseAndLease", [unleaseAmt], [])
    let gnsbtData = gnsbtControllerContract.invoke("gnsbtInfoSYSREADONLY", ["", 0, 0], []).asAnyList()
    let gnsbtAmtTotal           = gnsbtData[1].asInt()
    let gnsbtAmtFromSurfTotal   = gnsbtData[3].asAnyList()[3].asInt()

    let surfFeeAmt1 = if (gnsbtAmtTotal != 0) then fraction(fee.amount, gnsbtAmtFromSurfTotal, gnsbtAmtTotal) else 0
    let surfFeeAmt2 = if (gnsbtAmtTotal != 0) then fraction(fee.amount, PAULI - BR, PAULI) else 0
    let surfFeeAmt = max([surfFeeAmt1, surfFeeAmt2])
    let nsbtFeeAmt = fee.amount - surfFeeAmt

    strict surfDeposit = if (surfFeeAmt > 0) then {
      strict surfInv = surfStakingContract.invoke("deposit", [], [AttachedPayment(fee.assetId, surfFeeAmt)])
      []
    } else {[]}

    strict nsbtDeposit = if (nsbtFeeAmt > 0) then {
      strict nsbtInv = nsbtStakingContract.invoke("deposit", [], [AttachedPayment(fee.assetId, nsbtFeeAmt)])
      []
    } else {[]}

    state
}

@Callable(i)
func internalUnleaseAndLease(unleaseAmount: Int) = {
  if (i.caller != this) then throw("internalUnleaseAndLease is not public method") else
  prepareUnleaseAndLease(unleaseAmount)
}

# Callback for auction contract to transfer USDN to user
@Callable(i)
func transferUsdnToUser(amount: Int, addr: String) = {
    if (i.caller != auctionContract) then throw("Only auction contract is authorized") else

    [ScriptTransfer(addressFromStringValue(addr), amount, neutrinoAssetId)]
}

# Accept waves from auction after buyNsbt/buySurf to lease them immediately
# also from governance after creating new voting
@Callable(i)
func acceptWaves() = {
    if (i.caller != auctionContract && i.caller != govContract)
        then throw("Currently only auction and governance contracts are allowed to call")
    else
        (prepareUnleaseAndLease(0), "success")
}

@Callable(i)
func approveLeasings(nListS: String, groupNum: Int, lAmt: Int) = {
  let nIdxs = [0, 1, 2, 3, 4, 5, 6, 7]

  let mngPubS = getString("%s%s__cfg__leasingManagerPub").valueOrElse("7AUMX54ukYMYvPmma7yoFf5NjZhs4Bu5nz3Ez9EV8sur")
  let mngPub = mngPubS.fromBase58String()

  let nodeRegAddrStr = getString("%s%s__cfg__nodesRegistryAddress").valueOrElse("3P9vKqQKjUdmpXAfiWau8krREYAY1Xr69pE")
  let nodeRegAddr = nodeRegAddrStr.addressFromStringValue()

  let lGroupNodeListKEY = getLeaseGroupNodeListKey(groupNum)
  let lGrNodeOpt = this.getString(lGroupNodeListKEY)
  if (lGrNodeOpt.isDefined()) then throw("group " + groupNum.toString() + " already initialized") else

  let nList = nListS.split(SEP)
  let expCount = nIdxs.size()

  if (i.callerPublicKey != mngPub) then throw("approveLeasings not authorized") else

  let (nAddr0, lAmtKEY0, lAmt0, lIdKEY0, lId0) = readNodeInfo(0)

  let newL0 = Lease(nAddr0, lAmt0 - lAmt * expCount)

  strict validation = nodeRegAddr.invoke("validateAndApproveLeasings", [nListS], [])

  func forEachNodeValidateAndGenerateLease(a: List[Lease|BinaryEntry|IntegerEntry], i: Int) = {
    let node = nList[i]
    let la = Lease(node.addressFromStringValue(), lAmt)
    a++[la,
        BinaryEntry(getLeaseIdByAddressKey(node), lcalc(la)),
        IntegerEntry(getLeaseAmountByAddressKey(node), lAmt)]
  }

  [StringEntry(lGroupNodeListKEY, nListS),
    BinaryEntry(lIdKEY0, lcalc(newL0)),
    IntegerEntry(lAmtKEY0, newL0.amount),
    LeaseCancel(lId0),
    newL0
  ]
    ++ FOLD<8>(nIdxs, [], forEachNodeValidateAndGenerateLease)
}

@Callable(i)
func rebalanceLeasings(amount: Int, groupNum: Int) = {
  let nIdxs = [0, 1, 2, 3, 4, 5, 6, 7]

  let mngPubS = getString("%s%s__cfg__leasingManagerPub").valueOrElse("7AUMX54ukYMYvPmma7yoFf5NjZhs4Bu5nz3Ez9EV8sur")
  let mngPub = mngPubS.fromBase58String()

  let lGroupNodeListKEY = getLeaseGroupNodeListKey(groupNum)
  let nListS = this.getStringOrFail(lGroupNodeListKEY)
  let nList = nListS.split(SEP)

  if (i.callerPublicKey != mngPub) then throw("rebalanceLeasings not authorized") else

  let unleaseAmt = amount / nList.size() + 1
  let (nAddr0, lAmtKEY0, lAmt0, lIdKEY0, lId0) = readNodeInfo(0)

  let newL0 = Lease(nAddr0, lAmt0 + unleaseAmt * nList.size())

  func forEachNodeDoUnlease(a: List[Lease|BinaryEntry|IntegerEntry], i: Int) = {
    let node = nList[i]
    let lIdKEY = getLeaseIdByAddressKey(node)
    let lId = this.getBinaryValue(lIdKEY)
    let lAmtKEY = getLeaseAmountByAddressKey(node)
    let lAmt = this.getIntegerValue(lAmtKEY)

    let ula = LeaseCancel(lId.value())
    let la  = Lease(node.addressFromStringValue(), lAmt - unleaseAmt)
    a++[LeaseCancel(lId.value()),
        la,
        BinaryEntry(lIdKEY, lcalc(la)),
        IntegerEntry(lAmtKEY, la.amount)]
  }

  FOLD<8>(nIdxs, [], forEachNodeDoUnlease)
    ++ [
      BinaryEntry(lIdKEY0, lcalc(newL0)),
      IntegerEntry(lAmtKEY0, newL0.amount),
      LeaseCancel(lId0),
      newL0]
}

# READONLY methods
@Callable(i)
func swapParamsByUserSYSREADONLY(userAddressStr: String, gnsbtDiff: Int) = {
  let gnsbtData = gnsbtControllerContract.invoke("gnsbtInfoSYSREADONLY", [userAddressStr, 0, 0], []).asAnyList()

  let gnsbtAmt      = gnsbtData[0].asInt() + gnsbtDiff
  let gnsbtAmtTotal = gnsbtData[1].asInt() + gnsbtDiff

  let swapLimitData = mathContract.invoke("calcSwapLimitREADONLY", [gnsbtAmt], []).asAnyList()
  let wavesSwapLimitInUsdnMax = swapLimitData[0].asInt()
  let wavesSwapLimitMax       = swapLimitData[1].asInt()
  let usdnSwapLimitMax        = swapLimitData[2].asInt()

  let lastSwapHeight = this.getInteger(keyUserLastSwapHeight(userAddressStr)).valueOrElse(0)
  let swapLimitTimelifeBlocks = swapsTimeframeREAD()
  let passedBlocksAfterLastSwap = height - lastSwapHeight
  let isSwapTimelifeNew = passedBlocksAfterLastSwap >= swapLimitTimelifeBlocks
  let swapLimitSpentInUsdn = if (isSwapTimelifeNew) then 0 else this.getInteger(keySwapUserSpentInPeriod(userAddressStr)).valueOrElse(0)
  let blcks2LmtReset = if (isSwapTimelifeNew) then 0 else swapLimitTimelifeBlocks - passedBlocksAfterLastSwap

  # WARNING if you change returned value - MUST have to change "asSwapParamsSTRUCT" function
  ([], (wavesSwapLimitInUsdnMax, swapLimitSpentInUsdn, blcks2LmtReset, gnsbtAmt, gnsbtAmtTotal, wavesSwapLimitMax, usdnSwapLimitMax))
}

@Callable(i)
func calcWithdrawResultSYSREADONLY(swapType: String, inAmount: Int, price: Int) = {
  let neutrinoMetrics = mathContract.invoke("calcNeutinoMetricsREADONLY", [], []).asAnyList()
  ([], calcWithdraw(swapType, inAmount, price, neutrinoMetrics))
}

@Callable(i)
func replaceCommunityNode(oldAddrStr: String, newAddrStr: String, groupNum: Int, penaltyAmount: Int) = {
  let mngPubS = getString("%s%s__cfg__leasingManagerPub").valueOrElse("7AUMX54ukYMYvPmma7yoFf5NjZhs4Bu5nz3Ez9EV8sur")
  let mngPub = mngPubS.fromBase58String()
  if (i.callerPublicKey != mngPub) then throw("replaceCommunityNode not authorized") else

  let groupKey = getLeaseGroupNodeListKey(groupNum)
  let groupNodeListS = this.getStringOrFail(groupKey)
  if (!groupNodeListS.contains(oldAddrStr)) then throw("Group " + groupNum.toString() + " does not contain address " + oldAddrStr) else

  strict doReplace = nodeRegistryContract.invoke("replaceApprovedNode", [oldAddrStr, newAddrStr, groupNum, penaltyAmount], [])

  let oldLeaseIdKey = getLeaseIdByAddressKey(oldAddrStr)
  let oldLeaseAmtKey = getLeaseAmountByAddressKey(oldAddrStr)
  let leaseAmt = getIntegerValue(oldLeaseAmtKey)
  let newLeaseIdKey = getLeaseIdByAddressKey(oldAddrStr)
  let newLeaseAmtKey = getLeaseAmountByAddressKey(oldAddrStr)
  let newLease = Lease(newAddrStr.addressFromStringValue(), leaseAmt)
  let updatedGroupNodeListS = groupNodeListS.split(oldAddrStr).makeString(newAddrStr)

  ([LeaseCancel(getBinaryValue(oldLeaseIdKey)),
    DeleteEntry(oldLeaseIdKey),
    DeleteEntry(oldLeaseAmtKey),
    StringEntry(groupKey, updatedGroupNodeListS),
    newLease,
    BinaryEntry(newLeaseIdKey, lcalc(newLease)),
    IntegerEntry(newLeaseAmtKey, leaseAmt)
  ], unit)
}

@Verifier(tx)
func verify() = {
    let pubKeyAdminsListStr = makeString([
        "GJdLSaLiv5K7xuejac8mcRcHoyo3dPrESrvktG3a6MAR",
        "EYwZmURd5KKaQRBjsVa6g8DPisFoS6SovRJtFiL5gMHU",
        "DtmAfuDdCrHK8spdAeAYzq6MsZegeD9gnsrpuTRkCbVA",
        "5WRXFSjwcTbNfKcJs8ZqXmSSWYsSVJUtMvMqZj5hH4Nc"
    ], SEP)

    let pubKeyAdminsList = controlContract.getString("%s__multisig")
          .valueOrElse(pubKeyAdminsListStr)
          .split(SEP)

    let count =
        (if(sigVerify(tx.bodyBytes, tx.proofs[0], fromBase58String(pubKeyAdminsList[0]))) then 1 else 0) +
        (if(sigVerify(tx.bodyBytes, tx.proofs[1], fromBase58String(pubKeyAdminsList[1]))) then 1 else 0) +
        (if(sigVerify(tx.bodyBytes, tx.proofs[2], fromBase58String(pubKeyAdminsList[2]))) then 1 else 0) +
        (if(sigVerify(tx.bodyBytes, tx.proofs[3], fromBase58String(pubKeyAdminsList[3]))) then 2 else 0)

    if (isBlocked && 
      controlContract.getStringValue("is_blocked_caller") == govContract.toString()) then validateUpdate(tx) else {
        match tx {
            case sponsorTx: SponsorFeeTransaction =>
                checkIsValidMinSponsoredFee(sponsorTx) && count >= 3
            case _ => 
                count >= 3
        }
    }
}`,
			false, "BgJTCAISDgoMCAgICAgIAQEBAQEBEgUKAwgIARIAEgASBQoDCAEIEgMKAQESBAoCAQgSABIFCgMIAQESBAoCAQESBAoCCAESBQoDCAEBEgYKBAgIAQGnAQALcmV2aXNpb25OdW0CAAEPZ2V0U3RyaW5nT3JGYWlsAgdhZGRyZXNzA2tleQkBE3ZhbHVlT3JFcnJvck1lc3NhZ2UCCQCdCAIFB2FkZHJlc3MFA2tleQkAuQkCCQDMCAICCm1hbmRhdG9yeSAJAMwIAgkApQgBBQdhZGRyZXNzCQDMCAICAS4JAMwIAgUDa2V5CQDMCAICDyBpcyBub3QgZGVmaW5lZAUDbmlsAgABBWxjYWxjAQFsCQC5CAEFAWwBDmdldE51bWJlckJ5S2V5AQNrZXkJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwUDa2V5AAABDmdldFN0cmluZ0J5S2V5AQNrZXkJAQt2YWx1ZU9yRWxzZQIJAJ0IAgUEdGhpcwUDa2V5AgABDGdldEJvb2xCeUtleQEDa2V5CQELdmFsdWVPckVsc2UCCQCbCAIFBHRoaXMFA2tleQcBGGdldE51bWJlckJ5QWRkcmVzc0FuZEtleQIHYWRkcmVzcwNrZXkJAQt2YWx1ZU9yRWxzZQIJAJoIAgUHYWRkcmVzcwUDa2V5AAABGGdldFN0cmluZ0J5QWRkcmVzc0FuZEtleQIHYWRkcmVzcwNrZXkJAQt2YWx1ZU9yRWxzZQIJAJ0IAgkBEUBleHRyTmF0aXZlKDEwNjIpAQUHYWRkcmVzcwUDa2V5AgABFmdldEJvb2xCeUFkZHJlc3NBbmRLZXkCB2FkZHJlc3MDa2V5CQELdmFsdWVPckVsc2UCCQCbCAIFB2FkZHJlc3MFA2tleQcBCWFzQW55TGlzdAEBdgQHJG1hdGNoMAUBdgMJAAECBQckbWF0Y2gwAglMaXN0W0FueV0EAWwFByRtYXRjaDAFAWwJAAIBAhtmYWlsIHRvIGNhc3QgaW50byBMaXN0W0FueV0BCGFzU3RyaW5nAQF2BAckbWF0Y2gwBQF2AwkAAQIFByRtYXRjaDACBlN0cmluZwQBcwUHJG1hdGNoMAUBcwkAAgECGGZhaWwgdG8gY2FzdCBpbnRvIFN0cmluZwEFYXNJbnQBAXYEByRtYXRjaDAFAXYDCQABAgUHJG1hdGNoMAIDSW50BAFpBQckbWF0Y2gwBQFpCQACAQIVZmFpbCB0byBjYXN0IGludG8gSW50AQdhc0J5dGVzAQN2YWwEByRtYXRjaDAFA3ZhbAMJAAECBQckbWF0Y2gwAgpCeXRlVmVjdG9yBAd2YWxCeXRlBQckbWF0Y2gwBQd2YWxCeXRlCQACAQIcZmFpbCB0byBjYXN0IGludG8gQnl0ZVZlY3RvcgEJYXNQYXltZW50AQF2BAckbWF0Y2gwBQF2AwkAAQIFByRtYXRjaDACD0F0dGFjaGVkUGF5bWVudAQBcAUHJG1hdGNoMAUBcAkAAgECIWZhaWwgdG8gY2FzdCBpbnRvIEF0dGFjaGVkUGF5bWVudAESYXNTd2FwUGFyYW1zU1RSVUNUAQF2BAckbWF0Y2gwBQF2AwkAAQIFByRtYXRjaDACIyhJbnQsIEludCwgSW50LCBJbnQsIEludCwgSW50LCBJbnQpBAZzdHJ1Y3QFByRtYXRjaDAFBnN0cnVjdAkAAgECHWZhaWwgdG8gY2FzdCBpbnRvIFR1cGxlNSBpbnRzAANTRVACAl9fAAdMSVNUU0VQAgE6AAdXQVZFTEVUAIDC1y8ABVBBVUxJAMCEPQAIUFJJQ0VMRVQAwIQ9AA5ERUZBVUxUU1dBUEZFRQCgnAEAC0JSUFJPVEVDVEVEAKCNBgAMSWR4TmV0QW1vdW50AAAADElkeEZlZUFtb3VudAABAA5JZHhHcm9zc0Ftb3VudAACABlJZHhDb250cm9sQ2ZnTmV1dHJpbm9EYXBwAAEAGElkeENvbnRyb2xDZmdBdWN0aW9uRGFwcAACABRJZHhDb250cm9sQ2ZnUnBkRGFwcAADABVJZHhDb250cm9sQ2ZnTWF0aERhcHAABAAcSWR4Q29udHJvbENmZ0xpcXVpZGF0aW9uRGFwcAAFABVJZHhDb250cm9sQ2ZnUmVzdERhcHAABgAdSWR4Q29udHJvbENmZ05vZGVSZWdpc3RyeURhcHAABwAcSWR4Q29udHJvbENmZ05zYnRTdGFraW5nRGFwcAAIABlJZHhDb250cm9sQ2ZnTWVkaWF0b3JEYXBwAAkAHElkeENvbnRyb2xDZmdTdXJmU3Rha2luZ0RhcHAACgAgSWR4Q29udHJvbENmZ0duc2J0Q29udHJvbGxlckRhcHAACwAXSWR4Q29udHJvbENmZ1Jlc3RWMkRhcHAADAAbSWR4Q29udHJvbENmZ0dvdmVybmFuY2VEYXBwAA0BEWtleUNvbnRyb2xBZGRyZXNzAAIcJXMlc19fY29uZmlnX19jb250cm9sQWRkcmVzcwENa2V5Q29udHJvbENmZwACESVzX19jb250cm9sQ29uZmlnARRyZWFkQ29udHJvbENmZ09yRmFpbAEHY29udHJvbAkAvAkCCQEPZ2V0U3RyaW5nT3JGYWlsAgUHY29udHJvbAkBDWtleUNvbnRyb2xDZmcABQNTRVABGGdldENvbnRyYWN0QWRkcmVzc09yRmFpbAIKY29udHJvbENmZwNpZHgJARN2YWx1ZU9yRXJyb3JNZXNzYWdlAgkApggBCQCRAwIFCmNvbnRyb2xDZmcFA2lkeAkArAICAi1Db250cm9sIGNmZyBkb2Vzbid0IGNvbnRhaW4gYWRkcmVzcyBhdCBpbmRleCAJAKQDAQUDaWR4AA9jb250cm9sQ29udHJhY3QJARFAZXh0ck5hdGl2ZSgxMDYyKQEJAQt2YWx1ZU9yRWxzZQIJAJ0IAgUEdGhpcwkBEWtleUNvbnRyb2xBZGRyZXNzAAIjM1A1QmZkNThQUGZOdkJNMkh5OFFmYmNEcU1lTnR6ZzdLZlAACmNvbnRyb2xDZmcJARRyZWFkQ29udHJvbENmZ09yRmFpbAEFD2NvbnRyb2xDb250cmFjdAAMbWF0aENvbnRyYWN0CQEYZ2V0Q29udHJhY3RBZGRyZXNzT3JGYWlsAgUKY29udHJvbENmZwUVSWR4Q29udHJvbENmZ01hdGhEYXBwABNuc2J0U3Rha2luZ0NvbnRyYWN0CQEYZ2V0Q29udHJhY3RBZGRyZXNzT3JGYWlsAgUKY29udHJvbENmZwUcSWR4Q29udHJvbENmZ05zYnRTdGFraW5nRGFwcAATc3VyZlN0YWtpbmdDb250cmFjdAkBGGdldENvbnRyYWN0QWRkcmVzc09yRmFpbAIFCmNvbnRyb2xDZmcFHElkeENvbnRyb2xDZmdTdXJmU3Rha2luZ0RhcHAAF2duc2J0Q29udHJvbGxlckNvbnRyYWN0CQEYZ2V0Q29udHJhY3RBZGRyZXNzT3JGYWlsAgUKY29udHJvbENmZwUgSWR4Q29udHJvbENmZ0duc2J0Q29udHJvbGxlckRhcHAAD2F1Y3Rpb25Db250cmFjdAkBGGdldENvbnRyYWN0QWRkcmVzc09yRmFpbAIFCmNvbnRyb2xDZmcFGElkeENvbnRyb2xDZmdBdWN0aW9uRGFwcAAUbm9kZVJlZ2lzdHJ5Q29udHJhY3QJARhnZXRDb250cmFjdEFkZHJlc3NPckZhaWwCBQpjb250cm9sQ2ZnBR1JZHhDb250cm9sQ2ZnTm9kZVJlZ2lzdHJ5RGFwcAALZ292Q29udHJhY3QJARhnZXRDb250cmFjdEFkZHJlc3NPckZhaWwCBQpjb250cm9sQ2ZnBRtJZHhDb250cm9sQ2ZnR292ZXJuYW5jZURhcHAAEk5ldXRyaW5vQXNzZXRJZEtleQIRbmV1dHJpbm9fYXNzZXRfaWQADkJvbmRBc3NldElkS2V5Ag1ib25kX2Fzc2V0X2lkABJBdWN0aW9uQ29udHJhY3RLZXkCEGF1Y3Rpb25fY29udHJhY3QAFk5zYnRTdGFraW5nQ29udHJhY3RLZXkCE25zYnRTdGFraW5nQ29udHJhY3QAFkxpcXVpZGF0aW9uQ29udHJhY3RLZXkCFGxpcXVpZGF0aW9uX2NvbnRyYWN0AA5SUERDb250cmFjdEtleQIMcnBkX2NvbnRyYWN0ABFDb250b2xDb250cmFjdEtleQIQY29udHJvbF9jb250cmFjdAAPTWF0aENvbnRyYWN0S2V5Ag1tYXRoX2NvbnRyYWN0ABtCYWxhbmNlV2F2ZXNMb2NrSW50ZXJ2YWxLZXkCG2JhbGFuY2Vfd2F2ZXNfbG9ja19pbnRlcnZhbAAeQmFsYW5jZU5ldXRyaW5vTG9ja0ludGVydmFsS2V5Ah5iYWxhbmNlX25ldXRyaW5vX2xvY2tfaW50ZXJ2YWwAFU1pbldhdmVzU3dhcEFtb3VudEtleQIVbWluX3dhdmVzX3N3YXBfYW1vdW50ABhNaW5OZXV0cmlub1N3YXBBbW91bnRLZXkCGG1pbl9uZXV0cmlub19zd2FwX2Ftb3VudAAbTm9kZU9yYWNsZVByb3ZpZGVyUHViS2V5S2V5AhRub2RlX29yYWNsZV9wcm92aWRlcgAVTmV1dHJpbm9PdXRGZWVQYXJ0S2V5AhhuZXV0cmlub091dF9zd2FwX2ZlZVBhcnQAEldhdmVzT3V0RmVlUGFydEtleQIVd2F2ZXNPdXRfc3dhcF9mZWVQYXJ0AQ9rZXlOb2RlUmVnaXN0cnkBB2FkZHJlc3MJAKwCAgIEJXNfXwUHYWRkcmVzcwAIUHJpY2VLZXkCBXByaWNlAA1QcmljZUluZGV4S2V5AgtwcmljZV9pbmRleAAMSXNCbG9ja2VkS2V5Agppc19ibG9ja2VkARJnZXRQcmljZUhpc3RvcnlLZXkBBWJsb2NrCQCsAgIJAKwCAgUIUHJpY2VLZXkCAV8JAKQDAQUFYmxvY2sBGGdldEhlaWdodFByaWNlQnlJbmRleEtleQEFaW5kZXgJAKwCAgkArAICBQ1QcmljZUluZGV4S2V5AgFfCQCkAwEFBWluZGV4ARVnZXRTdGFraW5nTm9kZUJ5SW5kZXgBA2lkeAkBDmdldFN0cmluZ0J5S2V5AQkAuQkCCQDMCAICBiVzJWQlcwkAzAgCAgVsZWFzZQkAzAgCCQCkAwEFA2lkeAkAzAgCAgtub2RlQWRkcmVzcwUDbmlsBQNTRVABHGdldFN0YWtpbmdOb2RlQWRkcmVzc0J5SW5kZXgBA2lkeAkBEUBleHRyTmF0aXZlKDEwNjIpAQkBFWdldFN0YWtpbmdOb2RlQnlJbmRleAEFA2lkeAEfZ2V0UmVzZXJ2ZWRBbW91bnRGb3JTcG9uc29yc2hpcAAJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwkAuQkCCQDMCAICBCVzJXMJAMwIAgIFbGVhc2UJAMwIAgIXc3BvbnNvcnNoaXBXYXZlc1Jlc2VydmUFA25pbAUDU0VQCQBoAgDoBwUHV0FWRUxFVAEYZ2V0QmFsYW5jZVVubG9ja0Jsb2NrS2V5AQVvd25lcgkArAICAhViYWxhbmNlX3VubG9ja19ibG9ja18FBW93bmVyAQ1nZXRMZWFzZUlkS2V5AQlub2RlSW5kZXgJALkJAgkAzAgCAgYlcyVkJXMJAMwIAgIFbGVhc2UJAMwIAgkApAMBBQlub2RlSW5kZXgJAMwIAgICaWQFA25pbAUDU0VQARZnZXRMZWFzZUlkQnlBZGRyZXNzS2V5AQtub2RlQWRkcmVzcwkAuQkCCQDMCAICBiVzJXMlcwkAzAgCAg5sZWFzZUJ5QWRkcmVzcwkAzAgCBQtub2RlQWRkcmVzcwkAzAgCAgJpZAUDbmlsBQNTRVABEWdldExlYXNlQW1vdW50S2V5AQlub2RlSW5kZXgJALkJAgkAzAgCAgYlcyVkJXMJAMwIAgIFbGVhc2UJAMwIAgkApAMBBQlub2RlSW5kZXgJAMwIAgIGYW1vdW50BQNuaWwFA1NFUAEaZ2V0TGVhc2VBbW91bnRCeUFkZHJlc3NLZXkBC25vZGVBZGRyZXNzCQC5CQIJAMwIAgIGJXMlcyVzCQDMCAICDmxlYXNlQnlBZGRyZXNzCQDMCAIFC25vZGVBZGRyZXNzCQDMCAICBmFtb3VudAUDbmlsBQNTRVABGGdldExlYXNlR3JvdXBOb2RlTGlzdEtleQEIZ3JvdXBOdW0JALkJAgkAzAgCAgYlcyVkJXMJAMwIAgIKbGVhc2VHcm91cAkAzAgCCQCkAwEFCGdyb3VwTnVtCQDMCAICCG5vZGVMaXN0BQNuaWwFA1NFUAEQbWluU3dhcEFtb3VudEtFWQEIc3dhcFR5cGUJAKwCAgkArAICAgRtaW5fBQhzd2FwVHlwZQIMX3N3YXBfYW1vdW50AQ50b3RhbExvY2tlZEtFWQEIc3dhcFR5cGUJAKwCAgINYmFsYW5jZV9sb2NrXwUIc3dhcFR5cGUBFHRvdGFsTG9ja2VkQnlVc2VyS0VZAghzd2FwVHlwZQVvd25lcgkAuQkCCQDMCAICDGJhbGFuY2VfbG9jawkAzAgCBQhzd2FwVHlwZQkAzAgCBQVvd25lcgUDbmlsAgFfARZiYWxhbmNlTG9ja0ludGVydmFsS0VZAQhzd2FwVHlwZQkArAICCQCsAgICCGJhbGFuY2VfBQhzd2FwVHlwZQIOX2xvY2tfaW50ZXJ2YWwBGm5vZGVCYWxhbmNlTG9ja0ludGVydmFsS0VZAAIaYmFsYW5jZV9ub2RlX2xvY2tfaW50ZXJ2YWwBDW91dEZlZVBhcnRLRVkBCHN3YXBUeXBlCQCsAgIFCHN3YXBUeXBlAhBPdXRfc3dhcF9mZWVQYXJ0ARFzd2Fwc1RpbWVmcmFtZUtFWQACD3N3YXBzX3RpbWVmcmFtZQEOYnJQcm90ZWN0ZWRLRVkAAhdtaW5fQlJfcHJvdGVjdGlvbl9sZXZlbAERbWluU3dhcEFtb3VudFJFQUQBCHN3YXBUeXBlCQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJARBtaW5Td2FwQW1vdW50S0VZAQUIc3dhcFR5cGUAAAESc3dhcHNUaW1lZnJhbWVSRUFEAAkBC3ZhbHVlT3JFbHNlAgkAmggCBQR0aGlzCQERc3dhcHNUaW1lZnJhbWVLRVkAAKALAQ90b3RhbExvY2tlZFJFQUQBCHN3YXBUeXBlCQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJAQ50b3RhbExvY2tlZEtFWQEFCHN3YXBUeXBlAAABFXRvdGFsTG9ja2VkQnlVc2VyUkVBRAIIc3dhcFR5cGUFb3duZXIJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwkBFHRvdGFsTG9ja2VkQnlVc2VyS0VZAgUIc3dhcFR5cGUFBW93bmVyAAABF2JhbGFuY2VMb2NrSW50ZXJ2YWxSRUFEAQhzd2FwVHlwZQkBC3ZhbHVlT3JFbHNlAgkAmggCBQR0aGlzCQEWYmFsYW5jZUxvY2tJbnRlcnZhbEtFWQEFCHN3YXBUeXBlAKALARtub2RlQmFsYW5jZUxvY2tJbnRlcnZhbFJFQUQACQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJARpub2RlQmFsYW5jZUxvY2tJbnRlcnZhbEtFWQAAAQEYa2V5U3dhcFVzZXJTcGVudEluUGVyaW9kAQt1c2VyQWRkcmVzcwkAuQkCCQDMCAICBCVzJXMJAMwIAgIVc3dhcFVzZXJTcGVudEluUGVyaW9kCQDMCAIFC3VzZXJBZGRyZXNzBQNuaWwFA1NFUAEVa2V5VXNlckxhc3RTd2FwSGVpZ2h0AQt1c2VyQWRkcmVzcwkAuQkCCQDMCAICBCVzJXMJAMwIAgISdXNlckxhc3RTd2FwSGVpZ2h0CQDMCAIFC3VzZXJBZGRyZXNzBQNuaWwFA1NFUAEWY29udmVydE5ldXRyaW5vVG9XYXZlcwIGYW1vdW50BXByaWNlCQBrAwkAawMFBmFtb3VudAUIUFJJQ0VMRVQFBXByaWNlBQdXQVZFTEVUBQVQQVVMSQEWY29udmVydFdhdmVzVG9OZXV0cmlubwIGYW1vdW50BXByaWNlCQBrAwkAawMFBmFtb3VudAUFcHJpY2UFCFBSSUNFTEVUBQVQQVVMSQUHV0FWRUxFVAESY29udmVydFdhdmVzVG9Cb25kAgZhbW91bnQFcHJpY2UJARZjb252ZXJ0V2F2ZXNUb05ldXRyaW5vAgUGYW1vdW50BQVwcmljZQEWY29udmVydEpzb25BcnJheVRvTGlzdAEJanNvbkFycmF5CQC1CQIFCWpzb25BcnJheQIBLAERbWluU3dhcEFtb3VudEZBSUwCCHN3YXBUeXBlDW1pblN3YXBBbW91bnQJAAIBCQCsAgIJAKwCAgkArAICAhhUaGUgc3BlY2lmaWVkIGFtb3VudCBpbiAFCHN3YXBUeXBlAisgc3dhcCBpcyBsZXNzIHRoYW4gdGhlIHJlcXVpcmVkIG1pbmltdW0gb2YgCQCkAwEFDW1pblN3YXBBbW91bnQBFWVtZXJnZW5jeVNodXRkb3duRkFJTAAJAAIBAlpjb250cmFjdCBpcyBibG9ja2VkIGJ5IEVNRVJHRU5DWSBTSFVURE9XTiBhY3Rpb25zIHVudGlsbCByZWFjdGl2YXRpb24gYnkgZW1lcmdlbmN5IG9yYWNsZXMBDnByaWNlSW5kZXhGQUlMBQVpbmRleApwcmljZUluZGV4C2luZGV4SGVpZ2h0DHVubG9ja0hlaWdodA9wcmV2SW5kZXhIZWlnaHQJAAIBCQCsAgIJAKwCAgkArAICCQCsAgIJAKwCAgkArAICCQCsAgIJAKwCAgkArAICAiNpbnZhbGlkIHByaWNlIGhpc3RvcnkgaW5kZXg6IGluZGV4PQkApAMBBQVpbmRleAIMIHByaWNlSW5kZXg9CQCkAwEFCnByaWNlSW5kZXgCDSBpbmRleEhlaWdodD0JAKQDAQULaW5kZXhIZWlnaHQCDiB1bmxvY2tIZWlnaHQ9CQCkAwEFDHVubG9ja0hlaWdodAIRIHByZXZJbmRleEhlaWdodD0JAKQDAQUPcHJldkluZGV4SGVpZ2h0AA9uZXV0cmlub0Fzc2V0SWQJANkEAQkBDmdldFN0cmluZ0J5S2V5AQUSTmV1dHJpbm9Bc3NldElkS2V5AApwcmljZUluZGV4CQEYZ2V0TnVtYmVyQnlBZGRyZXNzQW5kS2V5AgUPY29udHJvbENvbnRyYWN0BQ1QcmljZUluZGV4S2V5AAlpc0Jsb2NrZWQJARZnZXRCb29sQnlBZGRyZXNzQW5kS2V5AgUPY29udHJvbENvbnRyYWN0BQxJc0Jsb2NrZWRLZXkAGG5vZGVPcmFjbGVQcm92aWRlclB1YktleQkA2QQBCQEOZ2V0U3RyaW5nQnlLZXkBBRtOb2RlT3JhY2xlUHJvdmlkZXJQdWJLZXlLZXkAC2JvbmRBc3NldElkCQDZBAECLDZuU3BWeU5IN3lNNjllZzQ0NndyUVI5NGlwYmJjbVpNVTFFTlB3YW5DOTdnABVkZXByZWNhdGVkQm9uZEFzc2V0SWQJANkEAQIsOTc1YWtaQmZuTWo1MTNVN01aYUhLelFybXNFeDVhRTN3ZFdLVHJIQmhiakYAEG5ldXRyaW5vQ29udHJhY3QFBHRoaXMADGN1cnJlbnRQcmljZQkBGGdldE51bWJlckJ5QWRkcmVzc0FuZEtleQIFD2NvbnRyb2xDb250cmFjdAUIUHJpY2VLZXkBG2NoZWNrSXNWYWxpZE1pblNwb25zb3JlZEZlZQECdHgEDk1JTlRSQU5TRkVSRkVFAKCNBgQWU3BvbnNvcmVkRmVlVXBwZXJCb3VuZADoBwQPcmVhbE5ldXRyaW5vRmVlCQEWY29udmVydFdhdmVzVG9OZXV0cmlubwIFDk1JTlRSQU5TRkVSRkVFBQxjdXJyZW50UHJpY2UEDm1pbk5ldXRyaW5vRmVlCQBoAgUPcmVhbE5ldXRyaW5vRmVlAAIEDm1heE5ldXRyaW5vRmVlCQBrAwUPcmVhbE5ldXRyaW5vRmVlBRZTcG9uc29yZWRGZWVVcHBlckJvdW5kAGQECGlucHV0RmVlCQEFdmFsdWUBCAUCdHgUbWluU3BvbnNvcmVkQXNzZXRGZWUDAwkAZwIFCGlucHV0RmVlBQ5taW5OZXV0cmlub0ZlZQkAZwIFDm1heE5ldXRyaW5vRmVlBQhpbnB1dEZlZQcJAAACCAUCdHgHYXNzZXRJZAUPbmV1dHJpbm9Bc3NldElkBwEPZ2V0UHJpY2VIaXN0b3J5AQVibG9jawkBGGdldE51bWJlckJ5QWRkcmVzc0FuZEtleQIFD2NvbnRyb2xDb250cmFjdAkBEmdldFByaWNlSGlzdG9yeUtleQEFBWJsb2NrARVnZXRIZWlnaHRQcmljZUJ5SW5kZXgBBWluZGV4CQEYZ2V0TnVtYmVyQnlBZGRyZXNzQW5kS2V5AgUPY29udHJvbENvbnRyYWN0CQEYZ2V0SGVpZ2h0UHJpY2VCeUluZGV4S2V5AQUFaW5kZXgBFmtleUxvY2tQYXJhbVVzZXJBbW91bnQBC3VzZXJBZGRyZXNzCQC5CQIJAMwIAgIGJXMlcyVzCQDMCAICC3BhcmFtQnlVc2VyCQDMCAIFC3VzZXJBZGRyZXNzCQDMCAICBmFtb3VudAUDbmlsBQNTRVAADHNJZHhTd2FwVHlwZQABAApzSWR4U3RhdHVzAAIADHNJZHhJbkFtb3VudAADAA9zSWR4U3RhcnRIZWlnaHQABwASc0lkeFN0YXJ0VGltZXN0YW1wAAgAFHNJZHhTZWxmVW5sb2NrSGVpZ2h0AAsAC3NJZHhNaW5SYW5kAA8AC3NJZHhNYXhSYW5kABABB3N3YXBLRVkCC3VzZXJBZGRyZXNzBHR4SWQJALkJAgkAzAgCAgQlcyVzCQDMCAIFC3VzZXJBZGRyZXNzCQDMCAIFBHR4SWQFA25pbAUDU0VQAQtzdHJTd2FwREFUQRIIc3dhcFR5cGUGc3RhdHVzCGluQW1vdW50BXByaWNlDG91dE5ldEFtb3VudAxvdXRGZWVBbW91bnQLc3RhcnRIZWlnaHQOc3RhcnRUaW1lc3RhbXAJZW5kSGVpZ2h0DGVuZFRpbWVzdGFtcBBzZWxmVW5sb2NrSGVpZ2h0EHJhbmRVbmxvY2tIZWlnaHQFaW5kZXgMd2l0aGRyYXdUeElkB3JhbmRNaW4HcmFuZE1heApvdXRTdXJmQW10AmJyCQC5CQIJAMwIAgIkJXMlcyVkJWQlZCVkJWQlZCVkJWQlZCVkJWQlcyVkJWQlZCVkCQDMCAIFCHN3YXBUeXBlCQDMCAIFBnN0YXR1cwkAzAgCBQhpbkFtb3VudAkAzAgCBQVwcmljZQkAzAgCBQxvdXROZXRBbW91bnQJAMwIAgUMb3V0RmVlQW1vdW50CQDMCAIFC3N0YXJ0SGVpZ2h0CQDMCAIFDnN0YXJ0VGltZXN0YW1wCQDMCAIFCWVuZEhlaWdodAkAzAgCBQxlbmRUaW1lc3RhbXAJAMwIAgUQc2VsZlVubG9ja0hlaWdodAkAzAgCBRByYW5kVW5sb2NrSGVpZ2h0CQDMCAIFBWluZGV4CQDMCAIFDHdpdGhkcmF3VHhJZAkAzAgCBQdyYW5kTWluCQDMCAIFB3JhbmRNYXgJAMwIAgUKb3V0U3VyZkFtdAkAzAgCBQJicgUDbmlsBQNTRVABD3BlbmRpbmdTd2FwREFUQQMIc3dhcFR5cGUNaW5Bc3NldEFtb3VudBBzZWxmVW5sb2NrSGVpZ2h0CQELc3RyU3dhcERBVEESBQhzd2FwVHlwZQIHUEVORElORwkApAMBBQ1pbkFzc2V0QW1vdW50AgEwAgEwAgEwCQCkAwEFBmhlaWdodAkApAMBCAUJbGFzdEJsb2NrCXRpbWVzdGFtcAIBMAIBMAkApAMBBRBzZWxmVW5sb2NrSGVpZ2h0AgEwAgEwAgROVUxMAgEwAgEwAgEwAgEwAQ5maW5pc2hTd2FwREFUQQkJZGF0YUFycmF5BXByaWNlDG91dE5ldEFtb3VudAxvdXRGZWVBbW91bnQQcmFuZFVubG9ja0hlaWdodAVpbmRleAx3aXRoZHJhd1R4SWQKb3V0U3VyZkFtdAJicgkBC3N0clN3YXBEQVRBEgkAkQMCBQlkYXRhQXJyYXkFDHNJZHhTd2FwVHlwZQIIRklOSVNIRUQJAJEDAgUJZGF0YUFycmF5BQxzSWR4SW5BbW91bnQJAKQDAQUFcHJpY2UJAKQDAQUMb3V0TmV0QW1vdW50CQCkAwEFDG91dEZlZUFtb3VudAkAkQMCBQlkYXRhQXJyYXkFD3NJZHhTdGFydEhlaWdodAkAkQMCBQlkYXRhQXJyYXkFEnNJZHhTdGFydFRpbWVzdGFtcAkApAMBBQZoZWlnaHQJAKQDAQgFCWxhc3RCbG9jawl0aW1lc3RhbXAJAJEDAgUJZGF0YUFycmF5BRRzSWR4U2VsZlVubG9ja0hlaWdodAkApAMBBRByYW5kVW5sb2NrSGVpZ2h0CQCkAwEFBWluZGV4BQx3aXRoZHJhd1R4SWQJAJEDAgUJZGF0YUFycmF5BQtzSWR4TWluUmFuZAkAkQMCBQlkYXRhQXJyYXkFC3NJZHhNYXhSYW5kCQCkAwEFCm91dFN1cmZBbXQJAKQDAQUCYnIBEnN3YXBEYXRhRmFpbE9yUkVBRAILdXNlckFkZHJlc3MIc3dhcFR4SWQEB3N3YXBLZXkJAQdzd2FwS0VZAgULdXNlckFkZHJlc3MFCHN3YXBUeElkCQC1CQIJARN2YWx1ZU9yRXJyb3JNZXNzYWdlAgkAnQgCBQR0aGlzBQdzd2FwS2V5CQCsAgICEW5vIHN3YXAgZGF0YSBmb3IgBQdzd2FwS2V5BQNTRVABCWFwcGx5RmVlcwMOYW1vdW50T3V0R3Jvc3MLaW5BbXRUb1NVUkYHZmVlUGFydAQJZmVlQW1vdW50CQBrAwUOYW1vdW50T3V0R3Jvc3MFB2ZlZVBhcnQFBVBBVUxJCQDMCAIJAGUCBQ5hbW91bnRPdXRHcm9zcwUJZmVlQW1vdW50CQDMCAIFCWZlZUFtb3VudAUDbmlsAQNhYnMBAXgDCQBmAgAABQF4CQEBLQEFAXgFAXgBCnNlbGVjdE5vZGUBDXVubGVhc2VBbW91bnQEDWFtb3VudFRvTGVhc2UJAGUCCQBlAggJAO8HAQUQbmV1dHJpbm9Db250cmFjdAlhdmFpbGFibGUFDXVubGVhc2VBbW91bnQJAR9nZXRSZXNlcnZlZEFtb3VudEZvclNwb25zb3JzaGlwAAQKb2xkTGVhc2VkMAkBDmdldE51bWJlckJ5S2V5AQkBEWdldExlYXNlQW1vdW50S2V5AQAABApvbGRMZWFzZWQxCQEOZ2V0TnVtYmVyQnlLZXkBCQERZ2V0TGVhc2VBbW91bnRLZXkBAAEECm5ld0xlYXNlZDAJAGQCBQ1hbW91bnRUb0xlYXNlBQpvbGRMZWFzZWQwBApuZXdMZWFzZWQxCQBkAgUNYW1vdW50VG9MZWFzZQUKb2xkTGVhc2VkMQMDCQBmAgUKbmV3TGVhc2VkMAAABgkAZgIFCm5ld0xlYXNlZDEAAAQGZGVsdGEwCQEDYWJzAQkAZQIFCm5ld0xlYXNlZDAFCm9sZExlYXNlZDEEBmRlbHRhMQkBA2FicwEJAGUCBQpuZXdMZWFzZWQxBQpvbGRMZWFzZWQwAwkAZwIFBmRlbHRhMQUGZGVsdGEwCQCUCgIAAAUKbmV3TGVhc2VkMAkAlAoCAAEFCm5ld0xlYXNlZDEJAJQKAgD///////////8BAAABCHRoaXNPbmx5AQFpAwkBAiE9AggFAWkGY2FsbGVyBQR0aGlzCQACAQItUGVybWlzc2lvbiBkZW5pZWQ6IHRoaXMgY29udHJhY3Qgb25seSBhbGxvd2VkBgEWcHJlcGFyZVVubGVhc2VBbmRMZWFzZQENdW5sZWFzZUFtb3VudAQJbm9kZVR1cGxlCQEKc2VsZWN0Tm9kZQEFDXVubGVhc2VBbW91bnQECW5vZGVJbmRleAgFCW5vZGVUdXBsZQJfMQQObmV3TGVhc2VBbW91bnQIBQlub2RlVHVwbGUCXzIDCQBmAgUObmV3TGVhc2VBbW91bnQAAAQKbGVhc2VJZEtleQkBDWdldExlYXNlSWRLZXkBBQlub2RlSW5kZXgECG9sZExlYXNlCQCcCAIFBHRoaXMFCmxlYXNlSWRLZXkEDnVubGVhc2VPckVtcHR5AwkBCWlzRGVmaW5lZAEFCG9sZExlYXNlCQDMCAIJAQtMZWFzZUNhbmNlbAEJAQV2YWx1ZQEFCG9sZExlYXNlBQNuaWwFA25pbAQObGVhc2VBbW91bnRLZXkJARFnZXRMZWFzZUFtb3VudEtleQEFCW5vZGVJbmRleAQFbGVhc2UJAMQIAgkBHGdldFN0YWtpbmdOb2RlQWRkcmVzc0J5SW5kZXgBBQlub2RlSW5kZXgFDm5ld0xlYXNlQW1vdW50CQDOCAIFDnVubGVhc2VPckVtcHR5CQDMCAIFBWxlYXNlCQDMCAIJAQtCaW5hcnlFbnRyeQIFCmxlYXNlSWRLZXkJAQVsY2FsYwEFBWxlYXNlCQDMCAIJAQxJbnRlZ2VyRW50cnkCCQERZ2V0TGVhc2VBbW91bnRLZXkBBQlub2RlSW5kZXgFDm5ld0xlYXNlQW1vdW50BQNuaWwFA25pbAEMcmVhZE5vZGVJbmZvAQdub2RlSWR4BAtub2RlQWRkcmVzcwkBHGdldFN0YWtpbmdOb2RlQWRkcmVzc0J5SW5kZXgBBQdub2RlSWR4BAxsZWFzZWRBbXRLRVkJARFnZXRMZWFzZUFtb3VudEtleQEFB25vZGVJZHgECWxlYXNlZEFtdAkBDmdldE51bWJlckJ5S2V5AQUMbGVhc2VkQW10S0VZBApsZWFzZUlkS0VZCQENZ2V0TGVhc2VJZEtleQEFB25vZGVJZHgEB2xlYXNlSWQJAQV2YWx1ZQEJAJwIAgUEdGhpcwUKbGVhc2VJZEtFWQkAlwoFBQtub2RlQWRkcmVzcwUMbGVhc2VkQW10S0VZBQlsZWFzZWRBbXQFCmxlYXNlSWRLRVkFB2xlYXNlSWQBCmNvbW1vblN3YXAFCHN3YXBUeXBlCXBtdEFtb3VudA51c2VyQWRkcmVzc1N0cgZ0eElkNTgbc3dhcFBhcmFtc0J5VXNlclNZU1JFQURPTkxZBA5zd2FwTGltaXRTcGVudAgFG3N3YXBQYXJhbXNCeVVzZXJTWVNSRUFET05MWQJfMgQOYmxja3MyTG10UmVzZXQIBRtzd2FwUGFyYW1zQnlVc2VyU1lTUkVBRE9OTFkCXzMEEXdhdmVzU3dhcExpbWl0TWF4CAUbc3dhcFBhcmFtc0J5VXNlclNZU1JFQURPTkxZAl82BBB1c2RuU3dhcExpbWl0TWF4CAUbc3dhcFBhcmFtc0J5VXNlclNZU1JFQURPTkxZAl83BA1taW5Td2FwQW1vdW50CQERbWluU3dhcEFtb3VudFJFQUQBBQhzd2FwVHlwZQQLdG90YWxMb2NrZWQJAQ90b3RhbExvY2tlZFJFQUQBBQhzd2FwVHlwZQQRdG90YWxMb2NrZWRCeVVzZXIJARV0b3RhbExvY2tlZEJ5VXNlclJFQUQCBQhzd2FwVHlwZQUOdXNlckFkZHJlc3NTdHIEC25vZGVBZGRyZXNzCQEVZ2V0U3Rha2luZ05vZGVCeUluZGV4AQAABAxwcmljZUJ5SW5kZXgJAQ9nZXRQcmljZUhpc3RvcnkBCQEVZ2V0SGVpZ2h0UHJpY2VCeUluZGV4AQUKcHJpY2VJbmRleAQMaXNTd2FwQnlOb2RlCQAAAgULbm9kZUFkZHJlc3MFDnVzZXJBZGRyZXNzU3RyBBZiYWxhbmNlTG9ja01heEludGVydmFsAwUMaXNTd2FwQnlOb2RlCQEbbm9kZUJhbGFuY2VMb2NrSW50ZXJ2YWxSRUFEAAkBF2JhbGFuY2VMb2NrSW50ZXJ2YWxSRUFEAQUIc3dhcFR5cGUEEHNlbGZVbmxvY2tIZWlnaHQJAGQCBQZoZWlnaHQFFmJhbGFuY2VMb2NrTWF4SW50ZXJ2YWwEDnN3YXBVc2RuVm9sdW1lAwkAAAIFCHN3YXBUeXBlAghuZXV0cmlubwUJcG10QW1vdW50CQEWY29udmVydFdhdmVzVG9OZXV0cmlubwIFCXBtdEFtb3VudAUMcHJpY2VCeUluZGV4BAxzd2FwTGltaXRNYXgDCQAAAgUIc3dhcFR5cGUCCG5ldXRyaW5vBRB1c2RuU3dhcExpbWl0TWF4CQEWY29udmVydFdhdmVzVG9OZXV0cmlubwIFEXdhdmVzU3dhcExpbWl0TWF4BQxwcmljZUJ5SW5kZXgDCQBmAgUNbWluU3dhcEFtb3VudAUJcG10QW1vdW50CQERbWluU3dhcEFtb3VudEZBSUwCBQhzd2FwVHlwZQUNbWluU3dhcEFtb3VudAMDCQEBIQEFDGlzU3dhcEJ5Tm9kZQkAZgIFDnN3YXBMaW1pdFNwZW50AAAHCQACAQkArAICAjpZb3UgaGF2ZSBleGNlZWRlZCBzd2FwIGxpbWl0ISBOZXh0IGFsbG93ZWQgc3dhcCBoZWlnaHQgaXMgCQCkAwEJAGQCBQZoZWlnaHQFDmJsY2tzMkxtdFJlc2V0AwMJAQEhAQUMaXNTd2FwQnlOb2RlCQBmAgUOc3dhcFVzZG5Wb2x1bWUFDHN3YXBMaW1pdE1heAcJAAIBCQCsAgIJAKwCAgkArAICAi5Zb3UgaGF2ZSBleGNlZWRlZCB5b3VyIHN3YXAgbGltaXQhIFJlcXVlc3RlZDogCQCkAwEFDnN3YXBVc2RuVm9sdW1lAg0sIGF2YWlsYWJsZTogCQCkAwEFDHN3YXBMaW1pdE1heAMFCWlzQmxvY2tlZAkBFWVtZXJnZW5jeVNodXRkb3duRkFJTAAECWxlYXNlUGFydAMJAAACBQhzd2FwVHlwZQIFd2F2ZXMJARZwcmVwYXJlVW5sZWFzZUFuZExlYXNlAQAABQNuaWwJAJQKAgkAzggCCQDMCAIJAQxJbnRlZ2VyRW50cnkCCQEYa2V5U3dhcFVzZXJTcGVudEluUGVyaW9kAQUOdXNlckFkZHJlc3NTdHIFDnN3YXBVc2RuVm9sdW1lCQDMCAIJAQxJbnRlZ2VyRW50cnkCCQEVa2V5VXNlckxhc3RTd2FwSGVpZ2h0AQUOdXNlckFkZHJlc3NTdHIFBmhlaWdodAkAzAgCCQEMSW50ZWdlckVudHJ5AgkBFHRvdGFsTG9ja2VkQnlVc2VyS0VZAgUIc3dhcFR5cGUFDnVzZXJBZGRyZXNzU3RyCQBkAgURdG90YWxMb2NrZWRCeVVzZXIFCXBtdEFtb3VudAkAzAgCCQEMSW50ZWdlckVudHJ5AgkBGGdldEJhbGFuY2VVbmxvY2tCbG9ja0tleQEFDnVzZXJBZGRyZXNzU3RyBRBzZWxmVW5sb2NrSGVpZ2h0CQDMCAIJAQxJbnRlZ2VyRW50cnkCCQEOdG90YWxMb2NrZWRLRVkBBQhzd2FwVHlwZQkAZAIFC3RvdGFsTG9ja2VkBQlwbXRBbW91bnQJAMwIAgkBC1N0cmluZ0VudHJ5AgkBB3N3YXBLRVkCBQ51c2VyQWRkcmVzc1N0cgUGdHhJZDU4CQEPcGVuZGluZ1N3YXBEQVRBAwUIc3dhcFR5cGUFCXBtdEFtb3VudAUQc2VsZlVubG9ja0hlaWdodAUDbmlsBQlsZWFzZVBhcnQFBHVuaXQAD25NZXRyaWNJZHhQcmljZQAAABtuTWV0cmljSWR4VXNkbkxvY2tlZEJhbGFuY2UAAQAcbk1ldHJpY0lkeFdhdmVzTG9ja2VkQmFsYW5jZQACABFuTWV0cmljSWR4UmVzZXJ2ZQADABduTWV0cmljSWR4UmVzZXJ2ZUluVXNkbgAEABRuTWV0cmljSWR4VXNkblN1cHBseQAFABFuTWV0cmljSWR4U3VycGx1cwAGABhuTWV0cmljSWR4U3VycGx1c1BlcmNlbnQABwAMbk1ldHJpY0lkeEJSAAgAFG5NZXRyaWNJZHhOc2J0U3VwcGx5AAkAF25NZXRyaWNJZHhNYXhOc2J0U3VwcGx5AAoAFG5NZXRyaWNJZHhTdXJmU3VwcGx5AAsADGJGdW5jSWR4U3VyZgAAAA1iRnVuY0lkeFdhdmVzAAEADGJGdW5jSWR4VXNkbgACABRiRnVuY0lkeFJlc2VydmVTdGFydAADABNiRnVuY0lkeFN1cHBseVN0YXJ0AAQAD2JGdW5jSWR4QlJTdGFydAAFABJiRnVuY0lkeFJlc2VydmVFbmQABgARYkZ1bmNJZHhTdXBwbHlFbmQABwANYkZ1bmNJZHhCUkVuZAAIAAxiRnVuY0lkeFJlc3QACQASYkZ1bmNJZHhXYXZlc1ByaWNlAAoBD2NhbGNXaXRoZHJhd1cyVQIHd2F2ZXNJbgVwcmljZQQLb3V0QW10R3Jvc3MJARZjb252ZXJ0V2F2ZXNUb05ldXRyaW5vAgUHd2F2ZXNJbgUFcHJpY2UJAJsKCQULb3V0QW10R3Jvc3MFD25ldXRyaW5vQXNzZXRJZAAABQR1bml0AAAFB3dhdmVzSW4AAAAAAAABD2NhbGNXaXRoZHJhd1UyVwUGdXNkbkluBXByaWNlAmJyDnJlc2VydmVzSW5Vc2RuCnVzZG5TdXBwbHkEC2JyUHJvdGVjdGVkCQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJAQ5iclByb3RlY3RlZEtFWQAFC0JSUFJPVEVDVEVEBBltYXhBbGxvd2VkVXNkbkJlZm9yZU1pbkJyAwkAZwIFC2JyUHJvdGVjdGVkBQJicgAACQBrAwkAZQIFDnJlc2VydmVzSW5Vc2RuCQBrAwULYnJQcm90ZWN0ZWQFCnVzZG5TdXBwbHkFBVBBVUxJBQVQQVVMSQkAZQIFBVBBVUxJBQticlByb3RlY3RlZAQWYWxsb3dlZFVzZG5CZWZvcmVNaW5CcgMJAGYCBQZ1c2RuSW4FGW1heEFsbG93ZWRVc2RuQmVmb3JlTWluQnIFGW1heEFsbG93ZWRVc2RuQmVmb3JlTWluQnIFBnVzZG5JbgQVYWxsb3dlZFVzZG5BZnRlck1pbkJyAwkAZgIFBnVzZG5JbgUZbWF4QWxsb3dlZFVzZG5CZWZvcmVNaW5CcgkAawMJAGUCBQZ1c2RuSW4FGW1heEFsbG93ZWRVc2RuQmVmb3JlTWluQnIFAmJyBQVQQVVMSQAABAthbGxvd2VkVXNkbgkAZAIFFmFsbG93ZWRVc2RuQmVmb3JlTWluQnIFFWFsbG93ZWRVc2RuQWZ0ZXJNaW5CcgQJdXNkbjJTVVJGCQBlAgUGdXNkbkluBQthbGxvd2VkVXNkbgQLb3V0QW10R3Jvc3MJARZjb252ZXJ0TmV1dHJpbm9Ub1dhdmVzAgULYWxsb3dlZFVzZG4FBXByaWNlCQCbCgkFC291dEFtdEdyb3NzBQR1bml0BQl1c2RuMlNVUkYFD25ldXRyaW5vQXNzZXRJZAULb3V0QW10R3Jvc3MFC2FsbG93ZWRVc2RuBRltYXhBbGxvd2VkVXNkbkJlZm9yZU1pbkJyBRZhbGxvd2VkVXNkbkJlZm9yZU1pbkJyBRVhbGxvd2VkVXNkbkFmdGVyTWluQnIBDGNhbGNXaXRoZHJhdwQIc3dhcFR5cGUIaW5BbW91bnQFcHJpY2UPbmV1dHJpbm9NZXRyaWNzBApvdXRGZWVQYXJ0CQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJAQ1vdXRGZWVQYXJ0S0VZAQUIc3dhcFR5cGUFDkRFRkFVTFRTV0FQRkVFAwMJAGYCAAAFCm91dEZlZVBhcnQGCQBnAgUKb3V0RmVlUGFydAUFUEFVTEkJAAIBCQCsAgIJAKwCAgkArAICAh5pbnZhbGlkIG91dEZlZVBhcnQgY29uZmlnIGZvciAFCHN3YXBUeXBlAhIgc3dhcDogb3V0RmVlUGFydD0JAKQDAQUKb3V0RmVlUGFydAQLYnJQcm90ZWN0ZWQJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwkBDmJyUHJvdGVjdGVkS0VZAAULQlJQUk9URUNURUQEAkJSCQEFYXNJbnQBCQCRAwIFD25ldXRyaW5vTWV0cmljcwUMbk1ldHJpY0lkeEJSBA5yZXNlcnZlc0luVXNkbgkBBWFzSW50AQkAkQMCBQ9uZXV0cmlub01ldHJpY3MFF25NZXRyaWNJZHhSZXNlcnZlSW5Vc2RuBAp1c2RuU3VwcGx5CQEFYXNJbnQBCQCRAwIFD25ldXRyaW5vTWV0cmljcwUUbk1ldHJpY0lkeFVzZG5TdXBwbHkEDG91dERhdGFUdXBsZQMJAAACBQhzd2FwVHlwZQIFd2F2ZXMJAQ9jYWxjV2l0aGRyYXdXMlUCBQhpbkFtb3VudAUFcHJpY2UDCQAAAgUIc3dhcFR5cGUCCG5ldXRyaW5vCQEPY2FsY1dpdGhkcmF3VTJXBQUIaW5BbW91bnQFBXByaWNlBQJCUgUOcmVzZXJ2ZXNJblVzZG4FCnVzZG5TdXBwbHkJAAIBCQCsAgICFlVuc3VwcG9ydGVkIHN3YXAgdHlwZSAFCHN3YXBUeXBlBAtvdXRBbXRHcm9zcwgFDG91dERhdGFUdXBsZQJfMQQKb3V0QXNzZXRJZAgFDG91dERhdGFUdXBsZQJfMgQPaW5BbXRUb1N1cmZQYXJ0CAUMb3V0RGF0YVR1cGxlAl8zBAlpbkFzc2V0SWQIBQxvdXREYXRhVHVwbGUCXzQECnVubGVhc2VBbXQIBQxvdXREYXRhVHVwbGUCXzUEDHBheW91dHNBcnJheQkBCWFwcGx5RmVlcwMFC291dEFtdEdyb3NzBQ9pbkFtdFRvU3VyZlBhcnQFCm91dEZlZVBhcnQECW91dE5ldEFtdAkAkQMCBQxwYXlvdXRzQXJyYXkFDElkeE5ldEFtb3VudAQJb3V0RmVlQW10CQCRAwIFDHBheW91dHNBcnJheQUMSWR4RmVlQW1vdW50BApvdXRTdXJmQW10AwkAZwIAAAUPaW5BbXRUb1N1cmZQYXJ0AAAECnN1cmZSZXN1bHQJAQlhc0FueUxpc3QBCQD8BwQFDG1hdGhDb250cmFjdAIUc3VyZkZ1bmN0aW9uUkVBRE9OTFkJAMwIAgUPaW5BbXRUb1N1cmZQYXJ0CQDMCAIFCWluQXNzZXRJZAUDbmlsBQNuaWwJAQVhc0ludAEJAJEDAgUKc3VyZlJlc3VsdAUMYkZ1bmNJZHhTdXJmCQCZCgcFCW91dE5ldEFtdAUKb3V0QXNzZXRJZAUKb3V0U3VyZkFtdAUPaW5BbXRUb1N1cmZQYXJ0BQp1bmxlYXNlQW10BQlvdXRGZWVBbXQFC291dEFtdEdyb3NzAQ5jb21tb25XaXRoZHJhdwUHYWNjb3VudAVpbmRleAhzd2FwVHhJZAx3aXRoZHJhd1R4SWQPbmV1dHJpbm9NZXRyaWNzBAt1c2VyQWRkcmVzcwkBEUBleHRyTmF0aXZlKDEwNjIpAQUHYWNjb3VudAQJZGF0YUFycmF5CQESc3dhcERhdGFGYWlsT3JSRUFEAgUHYWNjb3VudAUIc3dhcFR4SWQEEHNlbGZVbmxvY2tIZWlnaHQJAQ1wYXJzZUludFZhbHVlAQkAkQMCBQlkYXRhQXJyYXkFFHNJZHhTZWxmVW5sb2NrSGVpZ2h0BAhzd2FwVHlwZQkAkQMCBQlkYXRhQXJyYXkFDHNJZHhTd2FwVHlwZQQIaW5BbW91bnQJAQ1wYXJzZUludFZhbHVlAQkAkQMCBQlkYXRhQXJyYXkFDHNJZHhJbkFtb3VudAQKc3dhcFN0YXR1cwkAkQMCBQlkYXRhQXJyYXkFCnNJZHhTdGF0dXMEC3N0YXJ0SGVpZ2h0CQENcGFyc2VJbnRWYWx1ZQEJAJEDAgUJZGF0YUFycmF5BQ9zSWR4U3RhcnRIZWlnaHQECm91dEZlZVBhcnQJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwkBDW91dEZlZVBhcnRLRVkBBQhzd2FwVHlwZQUOREVGQVVMVFNXQVBGRUUEC3RvdGFsTG9ja2VkCQEPdG90YWxMb2NrZWRSRUFEAQUIc3dhcFR5cGUEEXRvdGFsTG9ja2VkQnlVc2VyCQEVdG90YWxMb2NrZWRCeVVzZXJSRUFEAgUIc3dhcFR5cGUFB2FjY291bnQEDHVubG9ja0hlaWdodAUQc2VsZlVubG9ja0hlaWdodAQLaW5kZXhIZWlnaHQJARVnZXRIZWlnaHRQcmljZUJ5SW5kZXgBBQVpbmRleAQPcHJldkluZGV4SGVpZ2h0CQEVZ2V0SGVpZ2h0UHJpY2VCeUluZGV4AQkAZQIFBWluZGV4AAEEDHByaWNlQnlJbmRleAkBD2dldFByaWNlSGlzdG9yeQEFC2luZGV4SGVpZ2h0AwUJaXNCbG9ja2VkCQEVZW1lcmdlbmN5U2h1dGRvd25GQUlMAAMJAQIhPQIFCnN3YXBTdGF0dXMCB1BFTkRJTkcJAAIBAh9zd2FwIGhhcyBiZWVuIGFscmVhZHkgcHJvY2Vzc2VkAwkAZgIFDHVubG9ja0hlaWdodAUGaGVpZ2h0CQACAQkArAICCQCsAgICEXBsZWFzZSB3YWl0IGZvcjogCQCkAwEFDHVubG9ja0hlaWdodAIfIGJsb2NrIGhlaWdodCB0byB3aXRoZHJhdyBmdW5kcwMDAwkAZgIFBWluZGV4BQpwcmljZUluZGV4BgkAZgIFDHVubG9ja0hlaWdodAULaW5kZXhIZWlnaHQGAwkBAiE9AgUPcHJldkluZGV4SGVpZ2h0AAAJAGcCBQ9wcmV2SW5kZXhIZWlnaHQFDHVubG9ja0hlaWdodAcJAQ5wcmljZUluZGV4RkFJTAUFBWluZGV4BQpwcmljZUluZGV4BQtpbmRleEhlaWdodAUMdW5sb2NrSGVpZ2h0BQ9wcmV2SW5kZXhIZWlnaHQEDXdpdGhkcmF3VHVwbGUJAQxjYWxjV2l0aGRyYXcEBQhzd2FwVHlwZQUIaW5BbW91bnQFDHByaWNlQnlJbmRleAUPbmV1dHJpbm9NZXRyaWNzBAxvdXROZXRBbW91bnQIBQ13aXRoZHJhd1R1cGxlAl8xBApvdXRBc3NldElkCAUNd2l0aGRyYXdUdXBsZQJfMgQKb3V0U3VyZkFtdAgFDXdpdGhkcmF3VHVwbGUCXzMECnVubGVhc2VBbXQIBQ13aXRoZHJhd1R1cGxlAl81BAxvdXRGZWVBbW91bnQIBQ13aXRoZHJhd1R1cGxlAl82BAtvdXRBbXRHcm9zcwgFDXdpdGhkcmF3VHVwbGUCXzcDCQBnAgAABQtvdXRBbXRHcm9zcwkAAgECE2JhbGFuY2UgZXF1YWxzIHplcm8EAkJSCQEFYXNJbnQBCQCRAwIFD25ldXRyaW5vTWV0cmljcwUMbk1ldHJpY0lkeEJSBAVzdGF0ZQkAzAgCCQEMSW50ZWdlckVudHJ5AgkBFHRvdGFsTG9ja2VkQnlVc2VyS0VZAgUIc3dhcFR5cGUFB2FjY291bnQJAGUCBRF0b3RhbExvY2tlZEJ5VXNlcgUIaW5BbW91bnQJAMwIAgkBDEludGVnZXJFbnRyeQIJAQ50b3RhbExvY2tlZEtFWQEFCHN3YXBUeXBlCQBlAgULdG90YWxMb2NrZWQFCGluQW1vdW50CQDMCAIJAQ5TY3JpcHRUcmFuc2ZlcgMFC3VzZXJBZGRyZXNzBQxvdXROZXRBbW91bnQFCm91dEFzc2V0SWQJAMwIAgkBC1N0cmluZ0VudHJ5AgkBB3N3YXBLRVkCBQdhY2NvdW50BQhzd2FwVHhJZAkBDmZpbmlzaFN3YXBEQVRBCQUJZGF0YUFycmF5BQxwcmljZUJ5SW5kZXgFDG91dE5ldEFtb3VudAUMb3V0RmVlQW1vdW50BQx1bmxvY2tIZWlnaHQFBWluZGV4BQx3aXRoZHJhd1R4SWQFCm91dFN1cmZBbXQFAkJSBQNuaWwEDXN1cmZDb25kaXRpb24DCQBmAgUKb3V0U3VyZkFtdAAABAtpc3N1ZVJlc3VsdAkA/AcEBQ9hdWN0aW9uQ29udHJhY3QCCWlzc3VlU3VyZgkAzAgCBQpvdXRTdXJmQW10CQDMCAIFB2FjY291bnQFA25pbAUDbmlsAwkAAAIFC2lzc3VlUmVzdWx0BQtpc3N1ZVJlc3VsdAAACQACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAADCQAAAgUNc3VyZkNvbmRpdGlvbgUNc3VyZkNvbmRpdGlvbgkAlQoDBQVzdGF0ZQkBD0F0dGFjaGVkUGF5bWVudAIFCm91dEFzc2V0SWQFDG91dEZlZUFtb3VudAUKdW5sZWFzZUFtdAkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgESa2V5QXBwbHlJblByb2dyZXNzAAITJXNfX2FwcGx5SW5Qcm9ncmVzcwETa2V5UHJvcG9zYWxEYXRhQnlJZAEKcHJvcG9zYWxJZAkArAICAhQlcyVkX19wcm9wb3NhbERhdGFfXwkApAMBBQpwcm9wb3NhbElkAAtnb3ZJZHhUeElkcwAJAQ52YWxpZGF0ZVVwZGF0ZQECdHgEByRtYXRjaDAFAnR4AwkAAQIFByRtYXRjaDACBU9yZGVyBAFvBQckbWF0Y2gwCQACAQIVT3JkZXJzIGFyZW4ndCBhbGxvd2VkAwMJAAECBQckbWF0Y2gwAg9EYXRhVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACFVNwb25zb3JGZWVUcmFuc2FjdGlvbgYDCQABAgUHJG1hdGNoMAIUU2V0U2NyaXB0VHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACFkNyZWF0ZUFsaWFzVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACFkxlYXNlQ2FuY2VsVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACEExlYXNlVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACEElzc3VlVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACG0ludm9rZUV4cHJlc3Npb25UcmFuc2FjdGlvbgYDCQABAgUHJG1hdGNoMAIaVXBkYXRlQXNzZXRJbmZvVHJhbnNhY3Rpb24GAwkAAQIFByRtYXRjaDACF0ludm9rZVNjcmlwdFRyYW5zYWN0aW9uBgMJAAECBQckbWF0Y2gwAhlTZXRBc3NldFNjcmlwdFRyYW5zYWN0aW9uBgMJAAECBQckbWF0Y2gwAhNUcmFuc2ZlclRyYW5zYWN0aW9uBgMJAAECBQckbWF0Y2gwAhNFeGNoYW5nZVRyYW5zYWN0aW9uBgMJAAECBQckbWF0Y2gwAhdNYXNzVHJhbnNmZXJUcmFuc2FjdGlvbgYDCQABAgUHJG1hdGNoMAIPQnVyblRyYW5zYWN0aW9uBgkAAQIFByRtYXRjaDACElJlaXNzdWVUcmFuc2FjdGlvbgQBdAUHJG1hdGNoMAQEdHhJZAkA2AQBCAUBdAJpZAQKcHJvcG9zYWxJZAkBE3ZhbHVlT3JFcnJvck1lc3NhZ2UCCQCaCAIFC2dvdkNvbnRyYWN0CQESa2V5QXBwbHlJblByb2dyZXNzAAIWQXBwbHkgaXMgbm90IGhhcHBlbmluZwQGdHhMaXN0CQC1CQIJAJEDAgkAtQkCCQEPZ2V0U3RyaW5nT3JGYWlsAgULZ292Q29udHJhY3QJARNrZXlQcm9wb3NhbERhdGFCeUlkAQUKcHJvcG9zYWxJZAUDU0VQBQtnb3ZJZHhUeElkcwUHTElTVFNFUAMJAQEhAQkBCWlzRGVmaW5lZAEJAM8IAgUGdHhMaXN0BQR0eElkCQACAQkArAICCQCsAgIJAKwCAgIOVW5rbm93biB0eElkOiAFBHR4SWQCECBmb3IgcHJvcG9zYWxJZD0JAKQDAQUKcHJvcG9zYWxJZAYJAAIBAgtNYXRjaCBlcnJvcg0BaQELY29uc3RydWN0b3IMEm5ldXRyaW5vQXNzZXRJZFBybQ5ib25kQXNzZXRJZFBybRJhdWN0aW9uQ29udHJhY3RQcm0WbGlxdWlkYXRpb25Db250cmFjdFBybQ5ycGRDb250cmFjdFBybRtub2RlT3JhY2xlUHJvdmlkZXJQdWJLZXlQcm0bYmFsYW5jZVdhdmVzTG9ja0ludGVydmFsUHJtHmJhbGFuY2VOZXV0cmlub0xvY2tJbnRlcnZhbFBybRVtaW5XYXZlc1N3YXBBbW91bnRQcm0YbWluTmV1dHJpbm9Td2FwQW1vdW50UHJtFW5ldXRyaW5vT3V0RmVlUGFydFBybRJ3YXZlc091dEZlZVBhcnRQcm0EC2NoZWNrQ2FsbGVyCQEIdGhpc09ubHkBBQFpAwkAAAIFC2NoZWNrQ2FsbGVyBQtjaGVja0NhbGxlcgMJAQIhPQIJAJADAQgFAWkIcGF5bWVudHMAAAkAAgECE25vIHBheW1lbnRzIGFsbG93ZWQJAMwIAgkBC1N0cmluZ0VudHJ5AgUSTmV1dHJpbm9Bc3NldElkS2V5BRJuZXV0cmlub0Fzc2V0SWRQcm0JAMwIAgkBC1N0cmluZ0VudHJ5AgUOQm9uZEFzc2V0SWRLZXkFDmJvbmRBc3NldElkUHJtCQDMCAIJAQtTdHJpbmdFbnRyeQIFEkF1Y3Rpb25Db250cmFjdEtleQUSYXVjdGlvbkNvbnRyYWN0UHJtCQDMCAIJAQtTdHJpbmdFbnRyeQIFFkxpcXVpZGF0aW9uQ29udHJhY3RLZXkFFmxpcXVpZGF0aW9uQ29udHJhY3RQcm0JAMwIAgkBC1N0cmluZ0VudHJ5AgUOUlBEQ29udHJhY3RLZXkFDnJwZENvbnRyYWN0UHJtCQDMCAIJAQtTdHJpbmdFbnRyeQIFG05vZGVPcmFjbGVQcm92aWRlclB1YktleUtleQUbbm9kZU9yYWNsZVByb3ZpZGVyUHViS2V5UHJtCQDMCAIJAQxJbnRlZ2VyRW50cnkCBRtCYWxhbmNlV2F2ZXNMb2NrSW50ZXJ2YWxLZXkFG2JhbGFuY2VXYXZlc0xvY2tJbnRlcnZhbFBybQkAzAgCCQEMSW50ZWdlckVudHJ5AgUeQmFsYW5jZU5ldXRyaW5vTG9ja0ludGVydmFsS2V5BR5iYWxhbmNlTmV1dHJpbm9Mb2NrSW50ZXJ2YWxQcm0JAMwIAgkBDEludGVnZXJFbnRyeQIFFU1pbldhdmVzU3dhcEFtb3VudEtleQUVbWluV2F2ZXNTd2FwQW1vdW50UHJtCQDMCAIJAQxJbnRlZ2VyRW50cnkCBRhNaW5OZXV0cmlub1N3YXBBbW91bnRLZXkFGG1pbk5ldXRyaW5vU3dhcEFtb3VudFBybQkAzAgCCQEMSW50ZWdlckVudHJ5AgUVTmV1dHJpbm9PdXRGZWVQYXJ0S2V5BRVuZXV0cmlub091dEZlZVBhcnRQcm0JAMwIAgkBDEludGVnZXJFbnRyeQIFEldhdmVzT3V0RmVlUGFydEtleQUSd2F2ZXNPdXRGZWVQYXJ0UHJtBQNuaWwJAAIBAiRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4BaQENY29uc3RydWN0b3JWMgMMbWF0aENvbnRyYWN0E25zYnRTdGFraW5nQ29udHJhY3QUc3dhcHNUaW1lZnJhbWVCbG9ja3MEC2NoZWNrQ2FsbGVyCQEIdGhpc09ubHkBBQFpAwkAAAIFC2NoZWNrQ2FsbGVyBQtjaGVja0NhbGxlcgMJAQIhPQIJAJADAQgFAWkIcGF5bWVudHMAAAkAAgECE25vIHBheW1lbnRzIGFsbG93ZWQJAMwIAgkBC1N0cmluZ0VudHJ5AgUPTWF0aENvbnRyYWN0S2V5BQxtYXRoQ29udHJhY3QJAMwIAgkBC1N0cmluZ0VudHJ5AgUWTnNidFN0YWtpbmdDb250cmFjdEtleQUTbnNidFN0YWtpbmdDb250cmFjdAkAzAgCCQEMSW50ZWdlckVudHJ5AgkBEXN3YXBzVGltZWZyYW1lS0VZAAUUc3dhcHNUaW1lZnJhbWVCbG9ja3MFA25pbAkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgFpARNzd2FwV2F2ZXNUb05ldXRyaW5vAAMJAQIhPQIJAJADAQgFAWkIcGF5bWVudHMAAQkAAgECLHN3YXBXYXZlc1RvTmV1dHJpbm8gcmVxdWlyZSBvbmx5IG9uZSBwYXltZW50BANwbXQJAQV2YWx1ZQEJAJEDAggFAWkIcGF5bWVudHMAAAMJAQlpc0RlZmluZWQBCAUDcG10B2Fzc2V0SWQJAAIBAilPbmx5IFdhdmVzIHRva2VuIGlzIGFsbG93ZWQgZm9yIHN3YXBwaW5nLgQLdXNlckFkZHJlc3MJAKUIAQgFAWkGY2FsbGVyBAZ0eElkNTgJANgEAQgFAWkNdHJhbnNhY3Rpb25JZAQQc3dhcFBhcmFtc1NUUlVDVAkBEmFzU3dhcFBhcmFtc1NUUlVDVAEJAPwHBAUEdGhpcwIbc3dhcFBhcmFtc0J5VXNlclNZU1JFQURPTkxZCQDMCAIFC3VzZXJBZGRyZXNzCQDMCAIAAAUDbmlsBQNuaWwEEGNvbW1vblN3YXBSZXN1bHQJAQpjb21tb25Td2FwBQIFd2F2ZXMIBQNwbXQGYW1vdW50BQt1c2VyQWRkcmVzcwUGdHhJZDU4BRBzd2FwUGFyYW1zU1RSVUNUBRBjb21tb25Td2FwUmVzdWx0AWkBE3N3YXBOZXV0cmlub1RvV2F2ZXMAAwkBAiE9AgkAkAMBCAUBaQhwYXltZW50cwABCQACAQIsc3dhcE5ldXRyaW5vVG9XYXZlcyByZXF1aXJlIG9ubHkgb25lIHBheW1lbnQEA3BtdAkBBXZhbHVlAQkAkQMCCAUBaQhwYXltZW50cwAAAwkBAiE9AggFA3BtdAdhc3NldElkBQ9uZXV0cmlub0Fzc2V0SWQJAAIBAjpPbmx5IGFwcHJvcHJpYXRlIE5ldXRyaW5vIHRva2VucyBhcmUgYWxsb3dlZCBmb3Igc3dhcHBpbmcuBAt1c2VyQWRkcmVzcwkApQgBCAUBaQZjYWxsZXIEBnR4SWQ1OAkA2AQBCAUBaQ10cmFuc2FjdGlvbklkBBBzd2FwUGFyYW1zU1RSVUNUCQESYXNTd2FwUGFyYW1zU1RSVUNUAQkA/AcEBQR0aGlzAhtzd2FwUGFyYW1zQnlVc2VyU1lTUkVBRE9OTFkJAMwIAgULdXNlckFkZHJlc3MJAMwIAgAABQNuaWwFA25pbAQQY29tbW9uU3dhcFJlc3VsdAkBCmNvbW1vblN3YXAFAghuZXV0cmlubwgFA3BtdAZhbW91bnQFC3VzZXJBZGRyZXNzBQZ0eElkNTgFEHN3YXBQYXJhbXNTVFJVQ1QFEGNvbW1vblN3YXBSZXN1bHQBaQEId2l0aGRyYXcDB2FjY291bnQFaW5kZXgIc3dhcFR4SWQEBHR4SWQJANgEAQgFAWkNdHJhbnNhY3Rpb25JZAMJAQIhPQIJAJADAQgFAWkIcGF5bWVudHMAAAkAAgECE25vIHBheW1lbnRzIGFsbG93ZWQED25ldXRyaW5vTWV0cmljcwkBCWFzQW55TGlzdAEJAPwHBAUMbWF0aENvbnRyYWN0AhpjYWxjTmV1dGlub01ldHJpY3NSRUFET05MWQUDbmlsBQNuaWwEAkJSCQEFYXNJbnQBCQCRAwIFD25ldXRyaW5vTWV0cmljcwUMbk1ldHJpY0lkeEJSBAtjb21tb25UdXBsZQkBDmNvbW1vbldpdGhkcmF3BQUHYWNjb3VudAUFaW5kZXgFCHN3YXBUeElkBQR0eElkBQ9uZXV0cmlub01ldHJpY3MEBXN0YXRlCAULY29tbW9uVHVwbGUCXzEEA2ZlZQgFC2NvbW1vblR1cGxlAl8yBAp1bmxlYXNlQW10CAULY29tbW9uVHVwbGUCXzMEEXVubGVhc2VJbnZPckVtcHR5CQD8BwQFBHRoaXMCF2ludGVybmFsVW5sZWFzZUFuZExlYXNlCQDMCAIFCnVubGVhc2VBbXQFA25pbAUDbmlsAwkAAAIFEXVubGVhc2VJbnZPckVtcHR5BRF1bmxlYXNlSW52T3JFbXB0eQQJZ25zYnREYXRhCQEJYXNBbnlMaXN0AQkA/AcEBRdnbnNidENvbnRyb2xsZXJDb250cmFjdAIUZ25zYnRJbmZvU1lTUkVBRE9OTFkJAMwIAgIACQDMCAIAAAkAzAgCAAAFA25pbAUDbmlsBA1nbnNidEFtdFRvdGFsCQEFYXNJbnQBCQCRAwIFCWduc2J0RGF0YQABBBVnbnNidEFtdEZyb21TdXJmVG90YWwJAQVhc0ludAEJAJEDAgkBCWFzQW55TGlzdAEJAJEDAgUJZ25zYnREYXRhAAMAAwQLc3VyZkZlZUFtdDEDCQECIT0CBQ1nbnNidEFtdFRvdGFsAAAJAGsDCAUDZmVlBmFtb3VudAUVZ25zYnRBbXRGcm9tU3VyZlRvdGFsBQ1nbnNidEFtdFRvdGFsAAAEC3N1cmZGZWVBbXQyAwkBAiE9AgUNZ25zYnRBbXRUb3RhbAAACQBrAwgFA2ZlZQZhbW91bnQJAGUCBQVQQVVMSQUCQlIFBVBBVUxJAAAECnN1cmZGZWVBbXQJAJYDAQkAzAgCBQtzdXJmRmVlQW10MQkAzAgCBQtzdXJmRmVlQW10MgUDbmlsBApuc2J0RmVlQW10CQBlAggFA2ZlZQZhbW91bnQFCnN1cmZGZWVBbXQEC3N1cmZEZXBvc2l0AwkAZgIFCnN1cmZGZWVBbXQAAAQHc3VyZkludgkA/AcEBRNzdXJmU3Rha2luZ0NvbnRyYWN0AgdkZXBvc2l0BQNuaWwJAMwIAgkBD0F0dGFjaGVkUGF5bWVudAIIBQNmZWUHYXNzZXRJZAUKc3VyZkZlZUFtdAUDbmlsAwkAAAIFB3N1cmZJbnYFB3N1cmZJbnYFA25pbAkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgUDbmlsAwkAAAIFC3N1cmZEZXBvc2l0BQtzdXJmRGVwb3NpdAQLbnNidERlcG9zaXQDCQBmAgUKbnNidEZlZUFtdAAABAduc2J0SW52CQD8BwQFE25zYnRTdGFraW5nQ29udHJhY3QCB2RlcG9zaXQFA25pbAkAzAgCCQEPQXR0YWNoZWRQYXltZW50AggFA2ZlZQdhc3NldElkBQpuc2J0RmVlQW10BQNuaWwDCQAAAgUHbnNidEludgUHbnNidEludgUDbmlsCQACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuBQNuaWwDCQAAAgULbnNidERlcG9zaXQFC25zYnREZXBvc2l0BQVzdGF0ZQkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgFpARdpbnRlcm5hbFVubGVhc2VBbmRMZWFzZQENdW5sZWFzZUFtb3VudAMJAQIhPQIIBQFpBmNhbGxlcgUEdGhpcwkAAgECLGludGVybmFsVW5sZWFzZUFuZExlYXNlIGlzIG5vdCBwdWJsaWMgbWV0aG9kCQEWcHJlcGFyZVVubGVhc2VBbmRMZWFzZQEFDXVubGVhc2VBbW91bnQBaQESdHJhbnNmZXJVc2RuVG9Vc2VyAgZhbW91bnQEYWRkcgMJAQIhPQIIBQFpBmNhbGxlcgUPYXVjdGlvbkNvbnRyYWN0CQACAQIjT25seSBhdWN0aW9uIGNvbnRyYWN0IGlzIGF1dGhvcml6ZWQJAMwIAgkBDlNjcmlwdFRyYW5zZmVyAwkBEUBleHRyTmF0aXZlKDEwNjIpAQUEYWRkcgUGYW1vdW50BQ9uZXV0cmlub0Fzc2V0SWQFA25pbAFpAQthY2NlcHRXYXZlcwADAwkBAiE9AggFAWkGY2FsbGVyBQ9hdWN0aW9uQ29udHJhY3QJAQIhPQIIBQFpBmNhbGxlcgULZ292Q29udHJhY3QHCQACAQJDQ3VycmVudGx5IG9ubHkgYXVjdGlvbiBhbmQgZ292ZXJuYW5jZSBjb250cmFjdHMgYXJlIGFsbG93ZWQgdG8gY2FsbAkAlAoCCQEWcHJlcGFyZVVubGVhc2VBbmRMZWFzZQEAAAIHc3VjY2VzcwFpAQ9hcHByb3ZlTGVhc2luZ3MDBm5MaXN0Uwhncm91cE51bQRsQW10BAVuSWR4cwkAzAgCAAAJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHBQNuaWwEB21uZ1B1YlMJAQt2YWx1ZU9yRWxzZQIJAKIIAQIcJXMlc19fY2ZnX19sZWFzaW5nTWFuYWdlclB1YgIsN0FVTVg1NHVrWU1ZdlBtbWE3eW9GZjVOalpoczRCdTVuejNFejlFVjhzdXIEBm1uZ1B1YgkA2QQBBQdtbmdQdWJTBA5ub2RlUmVnQWRkclN0cgkBC3ZhbHVlT3JFbHNlAgkAoggBAh8lcyVzX19jZmdfX25vZGVzUmVnaXN0cnlBZGRyZXNzAiMzUDl2S3FRS2pVZG1wWEFmaVdhdThrclJFWUFZMVhyNjlwRQQLbm9kZVJlZ0FkZHIJARFAZXh0ck5hdGl2ZSgxMDYyKQEFDm5vZGVSZWdBZGRyU3RyBBFsR3JvdXBOb2RlTGlzdEtFWQkBGGdldExlYXNlR3JvdXBOb2RlTGlzdEtleQEFCGdyb3VwTnVtBApsR3JOb2RlT3B0CQCdCAIFBHRoaXMFEWxHcm91cE5vZGVMaXN0S0VZAwkBCWlzRGVmaW5lZAEFCmxHck5vZGVPcHQJAAIBCQCsAgIJAKwCAgIGZ3JvdXAgCQCkAwEFCGdyb3VwTnVtAhQgYWxyZWFkeSBpbml0aWFsaXplZAQFbkxpc3QJALUJAgUGbkxpc3RTBQNTRVAECGV4cENvdW50CQCQAwEFBW5JZHhzAwkBAiE9AggFAWkPY2FsbGVyUHVibGljS2V5BQZtbmdQdWIJAAIBAh5hcHByb3ZlTGVhc2luZ3Mgbm90IGF1dGhvcml6ZWQEDSR0MDM1MzI5MzUzOTEJAQxyZWFkTm9kZUluZm8BAAAEBm5BZGRyMAgFDSR0MDM1MzI5MzUzOTECXzEECGxBbXRLRVkwCAUNJHQwMzUzMjkzNTM5MQJfMgQFbEFtdDAIBQ0kdDAzNTMyOTM1MzkxAl8zBAdsSWRLRVkwCAUNJHQwMzUzMjkzNTM5MQJfNAQEbElkMAgFDSR0MDM1MzI5MzUzOTECXzUEBW5ld0wwCQDECAIFBm5BZGRyMAkAZQIFBWxBbXQwCQBoAgUEbEFtdAUIZXhwQ291bnQECnZhbGlkYXRpb24JAPwHBAULbm9kZVJlZ0FkZHICGnZhbGlkYXRlQW5kQXBwcm92ZUxlYXNpbmdzCQDMCAIFBm5MaXN0UwUDbmlsBQNuaWwDCQAAAgUKdmFsaWRhdGlvbgUKdmFsaWRhdGlvbgoBI2ZvckVhY2hOb2RlVmFsaWRhdGVBbmRHZW5lcmF0ZUxlYXNlAgFhAWkEBG5vZGUJAJEDAgUFbkxpc3QFAWkEAmxhCQDECAIJARFAZXh0ck5hdGl2ZSgxMDYyKQEFBG5vZGUFBGxBbXQJAM4IAgUBYQkAzAgCBQJsYQkAzAgCCQELQmluYXJ5RW50cnkCCQEWZ2V0TGVhc2VJZEJ5QWRkcmVzc0tleQEFBG5vZGUJAQVsY2FsYwEFAmxhCQDMCAIJAQxJbnRlZ2VyRW50cnkCCQEaZ2V0TGVhc2VBbW91bnRCeUFkZHJlc3NLZXkBBQRub2RlBQRsQW10BQNuaWwJAM4IAgkAzAgCCQELU3RyaW5nRW50cnkCBRFsR3JvdXBOb2RlTGlzdEtFWQUGbkxpc3RTCQDMCAIJAQtCaW5hcnlFbnRyeQIFB2xJZEtFWTAJAQVsY2FsYwEFBW5ld0wwCQDMCAIJAQxJbnRlZ2VyRW50cnkCBQhsQW10S0VZMAgFBW5ld0wwBmFtb3VudAkAzAgCCQELTGVhc2VDYW5jZWwBBQRsSWQwCQDMCAIFBW5ld0wwBQNuaWwKAAIkbAUFbklkeHMKAAIkcwkAkAMBBQIkbAoABSRhY2MwBQNuaWwKAQUkZjBfMQICJGECJGkDCQBnAgUCJGkFAiRzBQIkYQkBI2ZvckVhY2hOb2RlVmFsaWRhdGVBbmRHZW5lcmF0ZUxlYXNlAgUCJGEJAJEDAgUCJGwFAiRpCgEFJGYwXzICAiRhAiRpAwkAZwIFAiRpBQIkcwUCJGEJAAIBAhNMaXN0IHNpemUgZXhjZWVkcyA4CQEFJGYwXzICCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECBQUkYWNjMAAAAAEAAgADAAQABQAGAAcACAkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgFpARFyZWJhbGFuY2VMZWFzaW5ncwIGYW1vdW50CGdyb3VwTnVtBAVuSWR4cwkAzAgCAAAJAMwIAgABCQDMCAIAAgkAzAgCAAMJAMwIAgAECQDMCAIABQkAzAgCAAYJAMwIAgAHBQNuaWwEB21uZ1B1YlMJAQt2YWx1ZU9yRWxzZQIJAKIIAQIcJXMlc19fY2ZnX19sZWFzaW5nTWFuYWdlclB1YgIsN0FVTVg1NHVrWU1ZdlBtbWE3eW9GZjVOalpoczRCdTVuejNFejlFVjhzdXIEBm1uZ1B1YgkA2QQBBQdtbmdQdWJTBBFsR3JvdXBOb2RlTGlzdEtFWQkBGGdldExlYXNlR3JvdXBOb2RlTGlzdEtleQEFCGdyb3VwTnVtBAZuTGlzdFMJAQ9nZXRTdHJpbmdPckZhaWwCBQR0aGlzBRFsR3JvdXBOb2RlTGlzdEtFWQQFbkxpc3QJALUJAgUGbkxpc3RTBQNTRVADCQECIT0CCAUBaQ9jYWxsZXJQdWJsaWNLZXkFBm1uZ1B1YgkAAgECIHJlYmFsYW5jZUxlYXNpbmdzIG5vdCBhdXRob3JpemVkBAp1bmxlYXNlQW10CQBkAgkAaQIFBmFtb3VudAkAkAMBBQVuTGlzdAABBA0kdDAzNjYzMTM2NjkzCQEMcmVhZE5vZGVJbmZvAQAABAZuQWRkcjAIBQ0kdDAzNjYzMTM2NjkzAl8xBAhsQW10S0VZMAgFDSR0MDM2NjMxMzY2OTMCXzIEBWxBbXQwCAUNJHQwMzY2MzEzNjY5MwJfMwQHbElkS0VZMAgFDSR0MDM2NjMxMzY2OTMCXzQEBGxJZDAIBQ0kdDAzNjYzMTM2NjkzAl81BAVuZXdMMAkAxAgCBQZuQWRkcjAJAGQCBQVsQW10MAkAaAIFCnVubGVhc2VBbXQJAJADAQUFbkxpc3QKARRmb3JFYWNoTm9kZURvVW5sZWFzZQIBYQFpBARub2RlCQCRAwIFBW5MaXN0BQFpBAZsSWRLRVkJARZnZXRMZWFzZUlkQnlBZGRyZXNzS2V5AQUEbm9kZQQDbElkCQERQGV4dHJOYXRpdmUoMTA1MikCBQR0aGlzBQZsSWRLRVkEB2xBbXRLRVkJARpnZXRMZWFzZUFtb3VudEJ5QWRkcmVzc0tleQEFBG5vZGUEBGxBbXQJARFAZXh0ck5hdGl2ZSgxMDUwKQIFBHRoaXMFB2xBbXRLRVkEA3VsYQkBC0xlYXNlQ2FuY2VsAQkBBXZhbHVlAQUDbElkBAJsYQkAxAgCCQERQGV4dHJOYXRpdmUoMTA2MikBBQRub2RlCQBlAgUEbEFtdAUKdW5sZWFzZUFtdAkAzggCBQFhCQDMCAIJAQtMZWFzZUNhbmNlbAEJAQV2YWx1ZQEFA2xJZAkAzAgCBQJsYQkAzAgCCQELQmluYXJ5RW50cnkCBQZsSWRLRVkJAQVsY2FsYwEFAmxhCQDMCAIJAQxJbnRlZ2VyRW50cnkCBQdsQW10S0VZCAUCbGEGYW1vdW50BQNuaWwJAM4IAgoAAiRsBQVuSWR4cwoAAiRzCQCQAwEFAiRsCgAFJGFjYzAFA25pbAoBBSRmMF8xAgIkYQIkaQMJAGcCBQIkaQUCJHMFAiRhCQEUZm9yRWFjaE5vZGVEb1VubGVhc2UCBQIkYQkAkQMCBQIkbAUCJGkKAQUkZjBfMgICJGECJGkDCQBnAgUCJGkFAiRzBQIkYQkAAgECE0xpc3Qgc2l6ZSBleGNlZWRzIDgJAQUkZjBfMgIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIJAQUkZjBfMQIFBSRhY2MwAAAAAQACAAMABAAFAAYABwAICQDMCAIJAQtCaW5hcnlFbnRyeQIFB2xJZEtFWTAJAQVsY2FsYwEFBW5ld0wwCQDMCAIJAQxJbnRlZ2VyRW50cnkCBQhsQW10S0VZMAgFBW5ld0wwBmFtb3VudAkAzAgCCQELTGVhc2VDYW5jZWwBBQRsSWQwCQDMCAIFBW5ld0wwBQNuaWwBaQEbc3dhcFBhcmFtc0J5VXNlclNZU1JFQURPTkxZAg51c2VyQWRkcmVzc1N0cglnbnNidERpZmYECWduc2J0RGF0YQkBCWFzQW55TGlzdAEJAPwHBAUXZ25zYnRDb250cm9sbGVyQ29udHJhY3QCFGduc2J0SW5mb1NZU1JFQURPTkxZCQDMCAIFDnVzZXJBZGRyZXNzU3RyCQDMCAIAAAkAzAgCAAAFA25pbAUDbmlsBAhnbnNidEFtdAkAZAIJAQVhc0ludAEJAJEDAgUJZ25zYnREYXRhAAAFCWduc2J0RGlmZgQNZ25zYnRBbXRUb3RhbAkAZAIJAQVhc0ludAEJAJEDAgUJZ25zYnREYXRhAAEFCWduc2J0RGlmZgQNc3dhcExpbWl0RGF0YQkBCWFzQW55TGlzdAEJAPwHBAUMbWF0aENvbnRyYWN0AhVjYWxjU3dhcExpbWl0UkVBRE9OTFkJAMwIAgUIZ25zYnRBbXQFA25pbAUDbmlsBBd3YXZlc1N3YXBMaW1pdEluVXNkbk1heAkBBWFzSW50AQkAkQMCBQ1zd2FwTGltaXREYXRhAAAEEXdhdmVzU3dhcExpbWl0TWF4CQEFYXNJbnQBCQCRAwIFDXN3YXBMaW1pdERhdGEAAQQQdXNkblN3YXBMaW1pdE1heAkBBWFzSW50AQkAkQMCBQ1zd2FwTGltaXREYXRhAAIEDmxhc3RTd2FwSGVpZ2h0CQELdmFsdWVPckVsc2UCCQCaCAIFBHRoaXMJARVrZXlVc2VyTGFzdFN3YXBIZWlnaHQBBQ51c2VyQWRkcmVzc1N0cgAABBdzd2FwTGltaXRUaW1lbGlmZUJsb2NrcwkBEnN3YXBzVGltZWZyYW1lUkVBRAAEGXBhc3NlZEJsb2Nrc0FmdGVyTGFzdFN3YXAJAGUCBQZoZWlnaHQFDmxhc3RTd2FwSGVpZ2h0BBFpc1N3YXBUaW1lbGlmZU5ldwkAZwIFGXBhc3NlZEJsb2Nrc0FmdGVyTGFzdFN3YXAFF3N3YXBMaW1pdFRpbWVsaWZlQmxvY2tzBBRzd2FwTGltaXRTcGVudEluVXNkbgMFEWlzU3dhcFRpbWVsaWZlTmV3AAAJAQt2YWx1ZU9yRWxzZQIJAJoIAgUEdGhpcwkBGGtleVN3YXBVc2VyU3BlbnRJblBlcmlvZAEFDnVzZXJBZGRyZXNzU3RyAAAEDmJsY2tzMkxtdFJlc2V0AwURaXNTd2FwVGltZWxpZmVOZXcAAAkAZQIFF3N3YXBMaW1pdFRpbWVsaWZlQmxvY2tzBRlwYXNzZWRCbG9ja3NBZnRlckxhc3RTd2FwCQCUCgIFA25pbAkAmQoHBRd3YXZlc1N3YXBMaW1pdEluVXNkbk1heAUUc3dhcExpbWl0U3BlbnRJblVzZG4FDmJsY2tzMkxtdFJlc2V0BQhnbnNidEFtdAUNZ25zYnRBbXRUb3RhbAURd2F2ZXNTd2FwTGltaXRNYXgFEHVzZG5Td2FwTGltaXRNYXgBaQEdY2FsY1dpdGhkcmF3UmVzdWx0U1lTUkVBRE9OTFkDCHN3YXBUeXBlCGluQW1vdW50BXByaWNlBA9uZXV0cmlub01ldHJpY3MJAQlhc0FueUxpc3QBCQD8BwQFDG1hdGhDb250cmFjdAIaY2FsY05ldXRpbm9NZXRyaWNzUkVBRE9OTFkFA25pbAUDbmlsCQCUCgIFA25pbAkBDGNhbGNXaXRoZHJhdwQFCHN3YXBUeXBlBQhpbkFtb3VudAUFcHJpY2UFD25ldXRyaW5vTWV0cmljcwFpARRyZXBsYWNlQ29tbXVuaXR5Tm9kZQQKb2xkQWRkclN0cgpuZXdBZGRyU3RyCGdyb3VwTnVtDXBlbmFsdHlBbW91bnQEB21uZ1B1YlMJAQt2YWx1ZU9yRWxzZQIJAKIIAQIcJXMlc19fY2ZnX19sZWFzaW5nTWFuYWdlclB1YgIsN0FVTVg1NHVrWU1ZdlBtbWE3eW9GZjVOalpoczRCdTVuejNFejlFVjhzdXIEBm1uZ1B1YgkA2QQBBQdtbmdQdWJTAwkBAiE9AggFAWkPY2FsbGVyUHVibGljS2V5BQZtbmdQdWIJAAIBAiNyZXBsYWNlQ29tbXVuaXR5Tm9kZSBub3QgYXV0aG9yaXplZAQIZ3JvdXBLZXkJARhnZXRMZWFzZUdyb3VwTm9kZUxpc3RLZXkBBQhncm91cE51bQQOZ3JvdXBOb2RlTGlzdFMJAQ9nZXRTdHJpbmdPckZhaWwCBQR0aGlzBQhncm91cEtleQMJAQEhAQkBCGNvbnRhaW5zAgUOZ3JvdXBOb2RlTGlzdFMFCm9sZEFkZHJTdHIJAAIBCQCsAgIJAKwCAgkArAICAgZHcm91cCAJAKQDAQUIZ3JvdXBOdW0CGiBkb2VzIG5vdCBjb250YWluIGFkZHJlc3MgBQpvbGRBZGRyU3RyBAlkb1JlcGxhY2UJAPwHBAUUbm9kZVJlZ2lzdHJ5Q29udHJhY3QCE3JlcGxhY2VBcHByb3ZlZE5vZGUJAMwIAgUKb2xkQWRkclN0cgkAzAgCBQpuZXdBZGRyU3RyCQDMCAIFCGdyb3VwTnVtCQDMCAIFDXBlbmFsdHlBbW91bnQFA25pbAUDbmlsAwkAAAIFCWRvUmVwbGFjZQUJZG9SZXBsYWNlBA1vbGRMZWFzZUlkS2V5CQEWZ2V0TGVhc2VJZEJ5QWRkcmVzc0tleQEFCm9sZEFkZHJTdHIEDm9sZExlYXNlQW10S2V5CQEaZ2V0TGVhc2VBbW91bnRCeUFkZHJlc3NLZXkBBQpvbGRBZGRyU3RyBAhsZWFzZUFtdAkBEUBleHRyTmF0aXZlKDEwNTUpAQUOb2xkTGVhc2VBbXRLZXkEDW5ld0xlYXNlSWRLZXkJARZnZXRMZWFzZUlkQnlBZGRyZXNzS2V5AQUKb2xkQWRkclN0cgQObmV3TGVhc2VBbXRLZXkJARpnZXRMZWFzZUFtb3VudEJ5QWRkcmVzc0tleQEFCm9sZEFkZHJTdHIECG5ld0xlYXNlCQDECAIJARFAZXh0ck5hdGl2ZSgxMDYyKQEFCm5ld0FkZHJTdHIFCGxlYXNlQW10BBV1cGRhdGVkR3JvdXBOb2RlTGlzdFMJALkJAgkAtQkCBQ5ncm91cE5vZGVMaXN0UwUKb2xkQWRkclN0cgUKbmV3QWRkclN0cgkAlAoCCQDMCAIJAQtMZWFzZUNhbmNlbAEJARFAZXh0ck5hdGl2ZSgxMDU3KQEFDW9sZExlYXNlSWRLZXkJAMwIAgkBC0RlbGV0ZUVudHJ5AQUNb2xkTGVhc2VJZEtleQkAzAgCCQELRGVsZXRlRW50cnkBBQ5vbGRMZWFzZUFtdEtleQkAzAgCCQELU3RyaW5nRW50cnkCBQhncm91cEtleQUVdXBkYXRlZEdyb3VwTm9kZUxpc3RTCQDMCAIFCG5ld0xlYXNlCQDMCAIJAQtCaW5hcnlFbnRyeQIFDW5ld0xlYXNlSWRLZXkJAQVsY2FsYwEFCG5ld0xlYXNlCQDMCAIJAQxJbnRlZ2VyRW50cnkCBQ5uZXdMZWFzZUFtdEtleQUIbGVhc2VBbXQFA25pbAUEdW5pdAkAAgECJFN0cmljdCB2YWx1ZSBpcyBub3QgZXF1YWwgdG8gaXRzZWxmLgECdHgBBnZlcmlmeQAEE3B1YktleUFkbWluc0xpc3RTdHIJALkJAgkAzAgCAixHSmRMU2FMaXY1Szd4dWVqYWM4bWNSY0hveW8zZFByRVNydmt0RzNhNk1BUgkAzAgCAixFWXdabVVSZDVLS2FRUkJqc1ZhNmc4RFBpc0ZvUzZTb3ZSSnRGaUw1Z01IVQkAzAgCAixEdG1BZnVEZENySEs4c3BkQWVBWXpxNk1zWmVnZUQ5Z25zcnB1VFJrQ2JWQQkAzAgCAiw1V1JYRlNqd2NUYk5mS2NKczhacVhtU1NXWXNTVkpVdE12TXFaajVoSDROYwUDbmlsBQNTRVAEEHB1YktleUFkbWluc0xpc3QJALUJAgkBC3ZhbHVlT3JFbHNlAgkAnQgCBQ9jb250cm9sQ29udHJhY3QCDCVzX19tdWx0aXNpZwUTcHViS2V5QWRtaW5zTGlzdFN0cgUDU0VQBAVjb3VudAkAZAIJAGQCCQBkAgMJAPQDAwgFAnR4CWJvZHlCeXRlcwkAkQMCCAUCdHgGcHJvb2ZzAAAJANkEAQkAkQMCBRBwdWJLZXlBZG1pbnNMaXN0AAAAAQAAAwkA9AMDCAUCdHgJYm9keUJ5dGVzCQCRAwIIBQJ0eAZwcm9vZnMAAQkA2QQBCQCRAwIFEHB1YktleUFkbWluc0xpc3QAAQABAAADCQD0AwMIBQJ0eAlib2R5Qnl0ZXMJAJEDAggFAnR4BnByb29mcwACCQDZBAEJAJEDAgUQcHViS2V5QWRtaW5zTGlzdAACAAEAAAMJAPQDAwgFAnR4CWJvZHlCeXRlcwkAkQMCCAUCdHgGcHJvb2ZzAAMJANkEAQkAkQMCBRBwdWJLZXlBZG1pbnNMaXN0AAMAAgAAAwMFCWlzQmxvY2tlZAkAAAIJARFAZXh0ck5hdGl2ZSgxMDUzKQIFD2NvbnRyb2xDb250cmFjdAIRaXNfYmxvY2tlZF9jYWxsZXIJAKUIAQULZ292Q29udHJhY3QHCQEOdmFsaWRhdGVVcGRhdGUBBQJ0eAQHJG1hdGNoMAUCdHgDCQABAgUHJG1hdGNoMAIVU3BvbnNvckZlZVRyYW5zYWN0aW9uBAlzcG9uc29yVHgFByRtYXRjaDADCQEbY2hlY2tJc1ZhbGlkTWluU3BvbnNvcmVkRmVlAQUJc3BvbnNvclR4CQBnAgUFY291bnQAAwcJAGcCBQVjb3VudAADVbh7Ug=="},
	} {
		rawAST, buf, err := buildAST(t, test.code, false)
		require.NoError(t, err)
		astParser := NewASTParser(rawAST, buf)
		astParser.Parse()
		if !test.fail {
			_, tree := parseBase64Script(t, test.expected)
			if diff := deep.Equal(tree.Declarations, astParser.Tree.Declarations); diff != nil {
				t.Errorf("Declaration mismatch:\n%s", strings.Join(diff, "\n"))
			}
			if diff := deep.Equal(tree.Functions, astParser.Tree.Functions); diff != nil {
				t.Errorf("Functions mismatch:\n%s", strings.Join(diff, "\n"))
			}
			if diff := deep.Equal(tree.Verifier, astParser.Tree.Verifier); diff != nil {
				t.Errorf("Verifier mismatch:\n%s", strings.Join(diff, "\n"))
			}
			if diff := deep.Equal(tree.Meta, astParser.Tree.Meta); diff != nil {
				t.Errorf("Meta mismatch:\n%s", strings.Join(diff, "\n"))
			}
		} else {
			assert.Len(t, astParser.ErrorsList, 1)
			assert.Equal(t, astParser.ErrorsList[0].Error(), test.expected)
		}
	}
}
