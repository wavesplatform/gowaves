package compiler

import (
	"encoding/base64"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"testing"

	"github.com/stretchr/testify/require"
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
		{`let a = "test"`, false, "BgICCAIBAAFhAgR0ZXN0AABM5UxM"},
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

func TestSimpleAST(t *testing.T) {
	src := `{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
let a = if true then 10 else "a"
let block = 
	match a {
		case x: Int => true
		case y: String => false
		case _ => false
	}
`
	ast, buf, err := buildAST(t, src, false)

	require.NoError(t, err)
	astParser := NewASTParser(ast, buf)
	astParser.Parse()
	for _, err := range astParser.ErrorsList {
		fmt.Println(err.Error())
	}
}
