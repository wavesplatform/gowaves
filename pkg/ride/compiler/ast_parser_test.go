package compiler

import (
	"context"
	"embed"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

func parseBase64Script(t *testing.T, src string) *ast.Tree {
	script, err := base64.StdEncoding.DecodeString(src)
	require.NoError(t, err)
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	require.NotNil(t, tree)
	return tree
}

func compareScriptsOrError(t *testing.T, code string, fail bool, expected string, compact bool, removeUnused bool) {
	tree, err := CompileToTree(code)
	if !fail {
		require.Empty(t, err)
		if removeUnused && tree.IsDApp() {
			removeUnusedCode(tree)
		}
		if compact && tree.IsDApp() {
			comp := NewCompaction(tree)
			comp.Compact()
		}
		expectedTree := parseBase64Script(t, expected)
		assert.Equal(t, expectedTree.ContentType, tree.ContentType)
		assert.Equal(t, expectedTree.LibVersion, tree.LibVersion)
		if diff := deep.Equal(expectedTree.Declarations, tree.Declarations); diff != nil {
			t.Errorf("Declaration mismatch:\n%s", strings.Join(diff, "\n"))
		}
		if diff := deep.Equal(expectedTree.Functions, tree.Functions); diff != nil {
			t.Errorf("Functions mismatch:\n%s", strings.Join(diff, "\n"))
		}
		if diff := deep.Equal(expectedTree.Verifier, tree.Verifier); diff != nil {
			t.Errorf("Verifier mismatch:\n%s", strings.Join(diff, "\n"))
		}
		if diff := deep.Equal(expectedTree.Meta, tree.Meta); diff != nil {
			t.Errorf("Meta mismatch:\n%s", strings.Join(diff, "\n"))
		}
	} else {
		require.NotEmpty(t, err, "Expected error, but errors list is empty")
		require.Equal(t, expected, err[0].Error())
	}
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
{-# STDLIB_VERSION 8 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:21): Invalid directive 'STDLIB_VERSION': unsupported library version '8'"}},
		{`
{-# STDLIB_VERSION 0 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:21): Invalid directive 'STDLIB_VERSION': unsupported library version '0'"}},
		{`
{-# STDLIB_VERSION XXX #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(2:20, 2:23): Failed to parse version 'XXX': strconv.ParseInt: parsing \"XXX\": invalid syntax"}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE XXX #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:5, 3:17): Illegal value 'XXX' of directive 'CONTENT_TYPE'"}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# XXX XXX #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:5, 3:8): Illegal directive 'XXX'"}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE XXX #-}`, []string{"(4:5, 4:16): Illegal value 'XXX' of directive 'SCRIPT_TYPE'"}},
		{`
{-# STDLIB_VERSION 6 #-}
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}`, []string{"(3:1, 4:0): Directive 'STDLIB_VERSION' is used more than once"}},
	} {
		code := test.code
		rawAST, buf, err := buildAST(t, code, false)
		assert.NoError(t, err)
		ap := newASTParser(rawAST, buf)
		ap.parse()
		assert.Equal(t, len(ap.errorsList), len(test.errorMsg))
		for i, err := range ap.errorsList {
			assert.Equal(t, test.errorMsg[i], err.Error())
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
		ap := newASTParser(rawAST, buf)
		ap.parse()
		assert.Equal(t, ap.tree.LibVersion, test.expected.LibVersion)
		assert.Equal(t, ap.tree.ContentType, test.expected.ContentType)
	}
}

func TestConstDeclaration(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
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
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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
		{`let a = "\a"`, true, "(4:10, 4:12): Unknown escaped symbol: '\\a'. The valid are \\b, \\f, \\n, \\r, \\t, \\\""},
		{`let a = "\u1234"`, false, "BgICCAIBAAFhAgPhiLQAAKUbIjo="},
		{`let a = "\u1234a\t"`, false, "BgICCAIBAAFhAgXhiLRhCQAADF+pNw=="},
	} {
		code := DappV6Directive + test.code
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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
		{`let a = (1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23)`, true, "(4:9, 4:92): Invalid Tuple length 23 (allowed 2 to 22)"},
		{`let (a, b, c) = (1, 2, 3)`, false, "BgICCAIEAAgkdDA3OTEwNAkAlQoDAAEAAgADAAFhCAUIJHQwNzkxMDQCXzEAAWIIBQgkdDA3OTEwNAJfMgABYwgFCCR0MDc5MTA0Al8zAAB8+W2s"},
		{`let a = (1, 2, 3)
let (b, c, d) = a`, false, "BgICCAIFAAFhCQCVCgMAAQACAAMACCR0MDk3MTE0BQFhAAFiCAUIJHQwOTcxMTQCXzEAAWMIBQgkdDA5NzExNAJfMgABZAgFCCR0MDk3MTE0Al8zAAAU7y0b"},
		{`let (a, b) = (1, "2", true)`, false, "BgICCAIDAAgkdDA3OTEwNgkAlQoDAAECATIGAAFhCAUIJHQwNzkxMDYCXzEAAWIIBQgkdDA3OTEwNgJfMgAAdj+WZg=="},
		{`let (a, b, c, d) = (1, "2", true)`, true, "(4:1, 4:34): Number of Identifiers should be less or equal than Tuple length"},
		{`
let a = if true then (1, 2, "a") else ("a", 1, 3)
let (b, c, d) = a
`, false, "BgICCAIFAAFhAwYJAJUKAwABAAICAWEJAJUKAwIBYQABAAMACSR0MDEzMDE0NwUBYQABYggFCSR0MDEzMDE0NwJfMQABYwgFCSR0MDEzMDE0NwJfMgABZAgFCSR0MDEzMDE0NwJfMwAAbYyzBw=="},
	} {
		code := DappV6Directive + test.code
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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
		{`let a = "a" + 1`, true, "(4:9, 4:16): Unexpected types for '+' operator 'String' and 'Int'"},
		{`let a = 1 > 2`, false, "BgICCAIBAAFhCQBmAgABAAIAAKf+6ug="},
		{`let a = 1 < 2`, false, "BgICCAIBAAFhCQBmAgACAAEAAAO8zuo="},
		{`let a = 1 <= 2`, false, "BgICCAIBAAFhCQBnAgACAAEAAJShBI8="},
		{`let a = 1 >= 2`, false, "BgICCAIBAAFhCQBnAgABAAIAAPdIIeU="},
		{`let a = 1 >= "a"`, true, "(4:14, 4:17): Unexpected type, required 'Int', but 'String' found"},
		{`let a = 1 == "a"`, true, "(4:14, 4:17): Unexpected type, required 'Int', but 'String' found"},
		{`
let a = if true then 1 else unit
let b = a == 10`,
			false, "BgICCAICAAFhAwYAAQUEdW5pdAABYgkAAAIFAWEACgAAXA5B8A=="},
		{`
let a = if true then 1 else unit
let b = a >= 10`,
			true, "(6:9, 6:10): Unexpected type, required 'BigInt' or 'Int', but 'Int|Unit' found"},
		{`let a = [1, 2] :+ "a"`, false, "BgICCAIBAAFhCQDNCAIJAMwIAgABCQDMCAIAAgUDbmlsAgFhAAAmqjlN"},
		{`let a = [1, 2] ++ nil`, false, "BgICCAIBAAFhCQDOCAIJAMwIAgABCQDMCAIAAgUDbmlsBQNuaWwAAOmqp9I="},
		{`let a = "a" :: [1, 2]`, false, "BgICCAIBAAFhCQDMCAICAWEJAMwIAgABCQDMCAIAAgUDbmlsAADcsh9u"},
	} {
		code := DappV6Directive + test.code
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
	}
}

func TestFOLD(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
func sum(accum: Int, next: Int) = accum + next
let arr = [1,2,3,4,5]
let a = FOLD<5>(arr, 0, sum)
`, false, "BgICCAIDAQNzdW0CBWFjY3VtBG5leHQJAGQCBQVhY2N1bQUEbmV4dAADYXJyCQDMCAIAAQkAzAgCAAIJAMwIAgADCQDMCAIABAkAzAgCAAUFA25pbAABYQoAAiRsBQNhcnIKAAIkcwkAkAMBBQIkbAoABSRhY2MwAAAKAQUkZjBfMQICJGECJGkDCQBnAgUCJGkFAiRzBQIkYQkBA3N1bQIFAiRhCQCRAwIFAiRsBQIkaQoBBSRmMF8yAgIkYQIkaQMJAGcCBQIkaQUCJHMFAiRhCQACAQITTGlzdCBzaXplIGV4Y2VlZHMgNQkBBSRmMF8yAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgkBBSRmMF8xAgUFJGFjYzAAAAABAAIAAwAEAAUAABK5ZXo="},
		{`
func filterEven(accum: List[Int], next: Int) =
if (next % 2 == 0) then accum :+ next else accum
let arr = [1,2,3,4,5]
let a = FOLD<5>(arr, [], filterEven)
`, false, "BgICCAIDAQpmaWx0ZXJFdmVuAgVhY2N1bQRuZXh0AwkAAAIJAGoCBQRuZXh0AAIAAAkAzQgCBQVhY2N1bQUEbmV4dAUFYWNjdW0AA2FycgkAzAgCAAEJAMwIAgACCQDMCAIAAwkAzAgCAAQJAMwIAgAFBQNuaWwAAWEKAAIkbAUDYXJyCgACJHMJAJADAQUCJGwKAAUkYWNjMAUDbmlsCgEFJGYwXzECAiRhAiRpAwkAZwIFAiRpBQIkcwUCJGEJAQpmaWx0ZXJFdmVuAgUCJGEJAJEDAgUCJGwFAiRpCgEFJGYwXzICAiRhAiRpAwkAZwIFAiRpBQIkcwUCJGEJAAIBAhNMaXN0IHNpemUgZXhjZWVkcyA1CQEFJGYwXzICCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECCQEFJGYwXzECBQUkYWNjMAAAAAEAAgADAAQABQAAWwkCmw=="},
		{`
func sum(a:Int, b:Int) = a + b
let a = FOLD<5>(1, 9, sum)
`, true, "(6:17, 6:18): First argument of fold must be List, but 'Int' found"},
		{`
func sum(a:Int, b:String) = a
let b = FOLD<5>([1], 0, sum)
`, true, "(6:25, 6:28): Can't find suitable function 'sum(Int, Int)'"},
		{`
func sum(a:Int) = a
let b = FOLD<5>([1], 0, sum)
`, true, "(6:25, 6:28): Function 'sum' must have 2 arguments"},
	} {
		code := DappV6Directive + test.code
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
	}
}

func TestExprSimple(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ASSET #-}
		
1 == 1
`, false, "BgEJAAACAAEAAb+26yY="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
func x() = true
x()
`, false, "BgEKAQF4AAYJAQF4AIghWZw="},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
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
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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

func test(a : BigInt) = true`, true, "(6:15, 6:21): Undefined type 'BigInt'"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func test(a : BigInt) = true`, false, "BgICCAIBAQR0ZXN0AQFhBgAA2dZM5Q=="},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
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
		let b = a.amountAsset`, false, "BgICCAICAAFhCQEJQXNzZXRQYWlyAgEAAQAAAWIIBQFhC2Ftb3VudEFzc2V0AADmGKl+"},
	} {

		code := DappV6Directive + test.code
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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
			true, "(9:10, 9:17): Matching not exhaustive: possible Types are 'Int|String', while matched are 'Boolean'"},
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
			true, "(8:17, 9:0): Variable 'y' doesn't exist"},
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
			true, "(8:9, 8:20): Matching not exhaustive: possible Types are 'Int|String', while matched are 'Boolean|Int'"},
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
			true, "(8:12, 8:20): Matching not exhaustive: possible Types are '(Int, String)', while matched are '(Int, ByteVector)'"},
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
		compareScriptsOrError(t, code, test.fail, test.expected, false, false)
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
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
	}
}

func TestListOp(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = if true then ["string"] else [1]
let b = a :+ "a"
`, false, "BgICCAICAAFhAwYJAMwIAgIGc3RyaW5nBQNuaWwJAMwIAgABBQNuaWwAAWIJAM0IAgUBYQIBYQAAmbXS3g=="},
		{
			`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = if true then ["string"] else [1]
let b = a :+ "a" 
let c = b :+ nil
let d = c :+ true
`, false, "AAIFAAAAAAAAAAIIAgAAAAQAAAAAAWEDBgkABEwAAAACAgAAAAZzdHJpbmcFAAAAA25pbAkABEwAAAACAAAAAAAAAAABBQAAAANuaWwAAAAAAWIJAARNAAAAAgUAAAABYQIAAAABYQAAAAABYwkABE0AAAACBQAAAAFiBQAAAANuaWwAAAAAAWQJAARNAAAAAgUAAAABYwYAAAAAAAAAAMlcQqo="},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
	}
}

func TestAsAndExactAs(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func foo(name: Any) = {
  let c = name.as[String]
  10
}
`, false, "BgICCAIBAQNmb28BBG5hbWUEAWMKAAFABQRuYW1lAwkAAQIFAUACBlN0cmluZwUBQAUEdW5pdAAKAADfHxhZ"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func foo(name: Any) = {
  let c = name.exactAs[String]
  10
}
`, false, "BgICCAIBAQNmb28BBG5hbWUEAWMKAAFABQRuYW1lAwkAAQIFAUACBlN0cmluZwUBQAkAAgEJAKwCAgkAAwEFAUACGyBjb3VsZG4ndCBiZSBjYXN0IHRvIFN0cmluZwAKAAB5dMaL"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func foo(a : Any) = {
    match a {
    case struct: (Int,ByteVector|Unit,Int|Unit,Int|Unit,Int|Unit,Int|Unit,Int|Unit,Int) => struct
    case _ => throw("fail to cast into WithdrawResult")
  }
}
`, false, "BgICCAIBAQNmb28BAWEEByRtYXRjaDAFAWEDAwkAAQIFByRtYXRjaDACLihJbnQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACLShJbnQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAItKEludCwgVW5pdCwgVW5pdCwgVW5pdCwgVW5pdCwgSW50LCBVbml0LCBJbnQpBgMJAAECBQckbWF0Y2gwAiwoSW50LCBVbml0LCBVbml0LCBVbml0LCBVbml0LCBJbnQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAItKEludCwgVW5pdCwgVW5pdCwgVW5pdCwgSW50LCBVbml0LCBVbml0LCBJbnQpBgMJAAECBQckbWF0Y2gwAiwoSW50LCBVbml0LCBVbml0LCBVbml0LCBJbnQsIFVuaXQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAIsKEludCwgVW5pdCwgVW5pdCwgVW5pdCwgSW50LCBJbnQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACKyhJbnQsIFVuaXQsIFVuaXQsIFVuaXQsIEludCwgSW50LCBJbnQsIEludCkGAwkAAQIFByRtYXRjaDACLShJbnQsIFVuaXQsIFVuaXQsIEludCwgVW5pdCwgVW5pdCwgVW5pdCwgSW50KQYDCQABAgUHJG1hdGNoMAIsKEludCwgVW5pdCwgVW5pdCwgSW50LCBVbml0LCBVbml0LCBJbnQsIEludCkGAwkAAQIFByRtYXRjaDACLChJbnQsIFVuaXQsIFVuaXQsIEludCwgVW5pdCwgSW50LCBVbml0LCBJbnQpBgMJAAECBQckbWF0Y2gwAisoSW50LCBVbml0LCBVbml0LCBJbnQsIFVuaXQsIEludCwgSW50LCBJbnQpBgMJAAECBQckbWF0Y2gwAiwoSW50LCBVbml0LCBVbml0LCBJbnQsIEludCwgVW5pdCwgVW5pdCwgSW50KQYDCQABAgUHJG1hdGNoMAIrKEludCwgVW5pdCwgVW5pdCwgSW50LCBJbnQsIFVuaXQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAIrKEludCwgVW5pdCwgVW5pdCwgSW50LCBJbnQsIEludCwgVW5pdCwgSW50KQYDCQABAgUHJG1hdGNoMAIqKEludCwgVW5pdCwgVW5pdCwgSW50LCBJbnQsIEludCwgSW50LCBJbnQpBgMJAAECBQckbWF0Y2gwAi0oSW50LCBVbml0LCBJbnQsIFVuaXQsIFVuaXQsIFVuaXQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACLChJbnQsIFVuaXQsIEludCwgVW5pdCwgVW5pdCwgVW5pdCwgSW50LCBJbnQpBgMJAAECBQckbWF0Y2gwAiwoSW50LCBVbml0LCBJbnQsIFVuaXQsIFVuaXQsIEludCwgVW5pdCwgSW50KQYDCQABAgUHJG1hdGNoMAIrKEludCwgVW5pdCwgSW50LCBVbml0LCBVbml0LCBJbnQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAIsKEludCwgVW5pdCwgSW50LCBVbml0LCBJbnQsIFVuaXQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACKyhJbnQsIFVuaXQsIEludCwgVW5pdCwgSW50LCBVbml0LCBJbnQsIEludCkGAwkAAQIFByRtYXRjaDACKyhJbnQsIFVuaXQsIEludCwgVW5pdCwgSW50LCBJbnQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACKihJbnQsIFVuaXQsIEludCwgVW5pdCwgSW50LCBJbnQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAIsKEludCwgVW5pdCwgSW50LCBJbnQsIFVuaXQsIFVuaXQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACKyhJbnQsIFVuaXQsIEludCwgSW50LCBVbml0LCBVbml0LCBJbnQsIEludCkGAwkAAQIFByRtYXRjaDACKyhJbnQsIFVuaXQsIEludCwgSW50LCBVbml0LCBJbnQsIFVuaXQsIEludCkGAwkAAQIFByRtYXRjaDACKihJbnQsIFVuaXQsIEludCwgSW50LCBVbml0LCBJbnQsIEludCwgSW50KQYDCQABAgUHJG1hdGNoMAIrKEludCwgVW5pdCwgSW50LCBJbnQsIEludCwgVW5pdCwgVW5pdCwgSW50KQYDCQABAgUHJG1hdGNo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"},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
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
			true, "(7:1, 10:0): Unexpected type in callable args 'Int|String'"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: List[Int|String]) = {
	([StringEntry("a", "a")], unit)
}
`,
			true, "(7:1, 10:0): Unexpected type in callable args 'List[Int|String]'"},
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
			true, "(7:1, 10:0): Unexpected type in callable args 'List[Int]'"},
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
		{`
{-# STDLIB_VERSION 4 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func test(a: List[Int|String]) = {
	[StringEntry("a", "a")]
}
`,
			false, "AAIEAAAAAAAAAAcIAhIDCgEZAAAAAAAAAAEAAAABaQEAAAAEdGVzdAAAAAEAAAABYQkABEwAAAACCQEAAAALU3RyaW5nRW50cnkAAAACAgAAAAFhAgAAAAFhBQAAAANuaWwAAAAAFGRLyg=="},
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func fn() = nil
`,
			false, "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAACZm4AAAAABQAAAANuaWwAAAAA4fhNCw=="},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
	}
}

func TestStrict(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = {
 strict (b, c) = (1, 2)
 10
}
`, false, "BgICCAIBAAFhBAgkdDA5MjExNAkAlAoCAAEAAgMJAAACBQgkdDA5MjExNAUIJHQwOTIxMTQEAWMIBQgkdDA5MjExNAJfMgQBYggFCCR0MDkyMTE0Al8xAAoJAAIBAiRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4AAC5dRqI="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = {
 strict b = 20
 10
}
`, false, "BgICCAIBAAFhBAFiABQDCQAAAgUBYgUBYgAKCQACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuAAALAtz6"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}

strict (str, i, cond, bytes) = ("12345", 12345, true, base58'')
str.parseInt() == i && cond && bytes.size() == 0
`, false, "BgEECCR0MDg3MTUwCQCWCgQCBTEyMzQ1ALlgBgEAAwkAAAIFCCR0MDg3MTUwBQgkdDA4NzE1MAQFYnl0ZXMIBQgkdDA4NzE1MAJfNAQEY29uZAgFCCR0MDg3MTUwAl8zBAFpCAUIJHQwODcxNTACXzIEA3N0cggFCCR0MDg3MTUwAl8xAwMJAAACCQC2CQEFA3N0cgUBaQUEY29uZAcJAAACCQDIAQEFBWJ5dGVzAAAHCQACAQIkU3RyaWN0IHZhbHVlIGlzIG5vdCBlcXVhbCB0byBpdHNlbGYuaoJOpg=="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func f() = if (true) then throw("exception") else 4
let (a, b, c, d) = (1, 2, 3, f())
true
`, false, "BgEKAQFmAAMGCQACAQIJZXhjZXB0aW9uAAQECSR0MDEzOTE3MgkAlgoEAAEAAgADCQEBZgAEAWEIBQkkdDAxMzkxNzICXzEEAWIIBQkkdDAxMzkxNzICXzIEAWMIBQkkdDAxMzkxNzICXzMEAWQIBQkkdDAxMzkxNzICXzQGe+5PRg=="},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
	}
}

func TestLib(t *testing.T) {
	for _, test := range []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# IMPORT lib_test_scripts/lib-foo-1.ride #-}

let a = {
  strict b = foo(10)
  10
}
`, false, "BgICCAICAQNmb28BAWEACgABYQQBYgkBA2ZvbwEACgMJAAACBQFiBQFiAAoJAAIBAiRTdHJpY3QgdmFsdWUgaXMgbm90IGVxdWFsIHRvIGl0c2VsZi4AAKF5kQ8="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# IMPORT lib_test_scripts/lib-foo-1.ride #-}

func bar() = foo(10)
bar() == 10
`, false, "BgEKAQNiYXIACQEDZm9vAQAKCQAAAgkBA2JhcgAACpAHkqA="},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# IMPORT lib_test_scripts/lib_failed.ride #-}

let a = {
    strict b = foo(10)
    10
}
`, true, "lib_test_scripts/lib_failed.ride(4:14, 4:17): Undefined type 'AST'"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# IMPORT lib_test_scripts/lib-foo-1.ride, lib_test_scripts/lib-foo-2.ride #-}

foo(10) == 10
`, true, "lib_test_scripts/lib-foo-2.ride(4:6, 4:9): Function 'foo' already exists"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ACCOUNT #-}
{-# IMPORT lib_test_scripts/lib-baz-1.ride, lib_test_scripts/lib-baz-2.ride #-}

baz != 10
`, true, "lib_test_scripts/lib-baz-2.ride(4:11, 4:12): Variable 'baz' already declared"},
	} {
		compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
	}
}

func TestAnyAndThrowTypes(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let a = if (true) then this.invoke("", [], []) else throw()
func f(b: Any) = b

@Callable(i)
func g() = f(a)`,
			true, "(10:1, 10:16): CallableFunc must return (List[BinaryEntry|BooleanEntry|Burn|DeleteEntry|IntegerEntry|Issue|Lease|LeaseCancel|Reissue|ScriptTransfer|SponsorFee|StringEntry], Any)|List[BinaryEntry|BooleanEntry|Burn|DeleteEntry|IntegerEntry|Issue|Lease|LeaseCancel|Reissue|ScriptTransfer|SponsorFee|StringEntry], but return Any"}, // https://waves-ide.com/s/641c6268c4784c002a8e8408
		{`
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

@Callable(i)
func cursed() = [][0]`,
			false, "AAIFAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAGY3Vyc2VkAAAAAAkAAZEAAAACBQAAAANuaWwAAAAAAAAAAAAAAAAAGWOB3Q=="}, // https://waves-ide.com/s/641c670fc4784c002a8e840a
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, false, false)
		})
	}
}

func TestBuiltInVarsWithCompaction(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func f(height: Any) = height

let a = height.f()`,
			false, "BgIQCAIiAWYiBmhlaWdodCIBYQIBAWEBAWIFAWIAAWMJAQFhAQUGaGVpZ2h0AAAPvLXS"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func f(unit: Any) = unit

let a = unit.f()`,
			false, "BgIOCAIiAWYiBHVuaXQiAWECAQFhAQFiBQFiAAFjCQEBYQEFBHVuaXQAALdQWqE="},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, true, false)
		})
	}
}

func TestBuiltInFuncsWithCompaction(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func f(sha256: Any) = sha256

let a = sha256(base16'')`,
			false, "BgIQCAIiAWYiBnNoYTI1NiIBYQIBAWEBAWIFAWIAAWMJAPcDAQEAAAAs+gZs"},
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func f(addressFromPublicKey: Any) = addressFromPublicKey

let a = addressFromPublicKey(base16'')`,
			false, "BgIeCAIiAWYiFGFkZHJlc3NGcm9tUHVibGljS2V5IgFhAgEBYQEBYgUBYgABYwkApwgBAQAAAHqqKaU="},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, true, false)
		})
	}
}

func TestRemoveUnusedCode(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}

let varX = 111
let varY = 222
let varZ = 333

func func3() = varZ * 444
func func2() = 100500 - varY
func func1() = func2() + 42

@Callable(i)
func call() = {
  let tmp1 = func1() + varX
  [IntegerEntry("somekey", tmp1)]
}

@Verifier(tx)
func verify() = {
  func2() != varX
}`,
			false, "BgIzCAISACIEdmFyWCIEdmFyWSIFZnVuYzIiBWZ1bmMxIgFpIgR0bXAxIgJ0eCIGdmVyaWZ5BAABYQBvAAFiAN4BAQFjAAkAZQIAlJEGBQFiAQFkAAkAZAIJAQFjAAAqAQFlAQRjYWxsAAQBZgkAZAIJAQFkAAUBYQkAzAgCCQEMSW50ZWdlckVudHJ5AgIHc29tZWtleQUBZgUDbmlsAQFnAQFoAAkBAiE9AgkBAWMABQFhYjSbXw=="},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, true, true)
		})
	}
}

func TestRemoveUnusedCodeWithLibs(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`
{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# IMPORT lib_test_scripts/lib1_unused.ride,lib_test_scripts/lib2_unused.ride #-}

@Verifier(tx)
func verify() = {
  foo() + bar() == 42
}`,
			false, "BgICCAICAQNmb28AACgBA2JhcgAAAgABAnR4AQZ2ZXJpZnkACQAAAgkAZAIJAQNmb28ACQEDYmFyAAAqeoOlqQ=="},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, false, true)
		})
	}
}

func TestStrangeComment(t *testing.T) {
	tests := []struct {
		code     string
		fail     bool
		expected string
	}{
		{`{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

func foo(a: (Int,
 Int # comment
 )) = {
  a._1 + a._2
}

@Callable(i)
func call() = {
  let a = foo((1, 2))
  ([], a)
}`,
			false, "AAIFAAAAAAAAAAQIAhIAAAAAAQEAAAADZm9vAAAAAQAAAAFhCQAAZAAAAAIIBQAAAAFhAAAAAl8xCAUAAAABYQAAAAJfMgAAAAEAAAABaQEAAAAEY2FsbAAAAAAEAAAAAWEJAQAAAANmb28AAAABCQAFFAAAAAIAAAAAAAAAAAEAAAAAAAAAAAIJAAUUAAAAAgUAAAADbmlsBQAAAAFhAAAAAOHevw4="},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			compareScriptsOrError(t, test.code, test.fail, test.expected, false, true)
		})
	}
}

//go:embed testdata
var embedScripts embed.FS

func TestCompilationWithScalaNode(t *testing.T) {
	if ok, err := strconv.ParseBool(os.Getenv("REAL_NODE")); err != nil || !ok {
		t.Skip("Skipping testing the compilation of scripts with comparison compilation from scala node")
	}
	cli, err := client.NewClient(client.Options{
		BaseUrl: "https://nodes.wavesnodes.com",
		Client:  &http.Client{Timeout: 10 * time.Second},
	})
	require.NoError(t, err)
	files, err := embedScripts.ReadDir("testdata")
	require.NoError(t, err)
	for _, file := range files {
		t.Logf("Test %s", file.Name())
		code, err := embedScripts.ReadFile("testdata/" + file.Name())
		require.NoError(t, err)
		res, _, err := cli.Utils.ScriptCompileCode(context.Background(), string(code), false)
		require.NoError(t, err)
		compareScriptsOrError(t, string(code), false, strings.TrimPrefix(res.Script, "base64:"), false, false)
	}
}

func TestCompilationWithScalaNodeWithCompaction(t *testing.T) {
	if ok, err := strconv.ParseBool(os.Getenv("REAL_NODE")); err != nil || !ok {
		t.Skip("Skipping testing the compilation of scripts with comparison compilation from scala node")
	}
	cli, err := client.NewClient(client.Options{
		BaseUrl: "https://nodes.wavesnodes.com",
		Client:  &http.Client{Timeout: 10 * time.Second},
	})
	require.NoError(t, err)
	files, err := embedScripts.ReadDir("testdata")
	require.NoError(t, err)
	for _, file := range files {
		t.Logf("Test %s", file.Name())
		code, err := embedScripts.ReadFile("testdata/" + file.Name())
		require.NoError(t, err)
		res, _, err := cli.Utils.ScriptCompileCode(context.Background(), string(code), true)
		require.NoError(t, err)
		compareScriptsOrError(t, string(code), false, strings.TrimPrefix(res.Script, "base64:"), true, false)
	}
}
