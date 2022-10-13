package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildAST(t *testing.T, src string, pretty bool) (*node32, []rune, error) {
	p := Parser{Buffer: src}
	p.Pretty = pretty
	err := p.Init()
	require.NoError(t, err)
	err = p.Parse()
	if err != nil {
		return nil, nil, err
	}
	if pretty {
		p.PrintSyntaxTree()
	}
	return p.AST(), p.buffer, nil
}

type nodeExpectation struct {
	name  string
	value string
}

func newNodeExpectation(t *testing.T, s string) nodeExpectation {
	valueBegin := strings.Index(s, "<")
	valueEnd := strings.LastIndex(s, ">")
	if valueBegin == -1 || valueEnd == -1 {
		assert.Fail(t, "invalid node expectation %q", s)
	}
	name := strings.TrimSpace(s[:valueBegin])
	value := strings.TrimSpace(s[valueBegin+1 : valueEnd])
	if name == "" || value == "" {
		assert.Fail(t, "invalid node expectation %q", s)
	}
	return nodeExpectation{name: name, value: value}
}

func checkAST(t *testing.T, expected string, ast *node32, buffer string) {
	exps := make([]nodeExpectation, 0)
	for _, s := range strings.Split(expected, ";") {
		exps = append(exps, newNodeExpectation(t, s))
	}
	i := 0

	discovered := map[*node32]struct{}{}
	stack := make([]*node32, 0)
	stack = append(stack, ast)
	for len(stack) > 0 {
		var n *node32
		n, stack = stack[0], stack[1:]
		if _, ok := discovered[n]; !ok {
			discovered[n] = struct{}{}
			rs := rul3s[n.pegRule]
			exp := exps[i]
			if rs == exp.name {
				quote := string([]rune(buffer)[n.begin:n.end])
				if exp.value != "*" {
					require.Equal(t, exp.value, quote, buffer)
				}
				i++
				if i == len(exps) {
					return
				}
			}
			if n.next != nil {
				stack = append([]*node32{n.next}, stack...)
			}
			if n.up != nil {
				stack = append([]*node32{n.up}, stack...)
			}
		}
	}
	if len(exps[i:]) > 0 {
		assert.Fail(t, fmt.Sprintf("unmet expectations: %v", exps[i:]))
	}
}

func TestDirectives(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`{-# STDLIB_VERSION 6 #-}`, false, "Directive<*>;DirectiveName<STDLIB_VERSION>;IntString<6>"},
		{`{-# STDLIB_VERSION XXX #-}`, false, "Directive<*>;DirectiveName<STDLIB_VERSION>;UpperCaseString<XXX>"},
		{`{-# CONTENT_TYPE DAPP #-}`, false, "Directive<*>;DirectiveName<CONTENT_TYPE>;UpperCaseString<DAPP>"},
		{`{-# SCRIPT_TYPE ACCOUNT #-}`, false, "Directive<*>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<ACCOUNT>"},
		{`{-#	SCRIPT_TYPE 	 ACCOUNT      #-}`, false, "Directive<*>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<ACCOUNT>"},
		{`{-# IMPORT lib1 #-}`, false, "Directive<*>;DirectiveName<IMPORT>;PathString<lib1>"},
		{`{-# IMPORT lib1,my_lib2 #-}`, false, "Directive<*>;DirectiveName<IMPORT>;PathString<lib1,my_lib2>"},
		{`{-# IMPORT lib3.ride,dir/lib4.ride #-}`, false, "Directive<*>;DirectiveName<IMPORT>;PathString<lib3.ride,dir/lib4.ride>"},
		{`{-# STDLIB_version 123 #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"STDLIB_\"\n"},
		{`{-# NAME #-}`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`{-# 123 #-}`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`{-# CONTENT_TYPE account #-}`, false, "Directive<*>;DirectiveName<CONTENT_TYPE>;PathString<account>"},
		{`{-# CONTENT-TYPE ACCOUNT #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"CONTENT\"\n"},
		{`{-# IMPORT lib3.ride,dir\lib4.ride #-}`, true, "\nparse error near PathString (line 1 symbol 12 - line 1 symbol 25):\n\"lib3.ride,dir\"\n"},
		{`{-# IMPORT lib3.ride #-} # comment`, false, "Directive<*>;DirectiveName<IMPORT>;PathString<lib3.ride>"},
		{`	{-# STDLIB_VERSION 6 #-}
				{-# IMPORT lib3.ride,lib4.ride #-} # comment
				{-# CONTENT_TYPE ACCOUNT #-}
				{-# SCRIPT_TYPE DAPP #-}
			`, false,
			"Directive<*>;DirectiveName<STDLIB_VERSION>;IntString<6>;" +
				"Directive<*>;DirectiveName<IMPORT>;PathString<lib3.ride,lib4.ride>;" +
				"Directive<*>;DirectiveName<CONTENT_TYPE>;UpperCaseString<ACCOUNT>;" +
				"Directive<*>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<DAPP>"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestByteVector(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`base16''`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base16<base16''>"},
		{`base58''`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base58<base58''>"},
		{`base64''`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base64<base64''>"},
		{`base16'cafeBEBE12345'`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base16<base16'cafeBEBE12345'>"},
		{`base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base58<base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'>"},
		{`base64'SGVsbG8gd29ybGQhISE='`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base64<base64'SGVsbG8gd29ybGQhISE='>"},
		{`base64'SGVsbG8gd29ybGQhIQ=='`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base64<base64'SGVsbG8gd29ybGQhIQ=='>"},
		{`base16'cafeBEBE12345'`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base16<base16'cafeBEBE12345'>"},
		{`base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base58<base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'>"},
		{`base64'SGVsbG8gd29ybGQhISE='`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base64<base64'SGVsbG8gd29ybGQhISE='>"},
		{`base64'SGVsbG8gd29ybGQhIQ=='`, false, "ConstAtom<*>;ByteVectorAtom<*>;Base64<base64'SGVsbG8gd29ybGQhIQ=='>"},
		{`base16'JFK'`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 7):\n\"base16\"\n"},
		{`base58'IO0'`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 7):\n\"base58\"\n"},
		{`base64'BASE64_-`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 7):\n\"base64\"\n"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestString(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`"some string"`, false, "ConstAtom<*>;StringAtom<\"some string\">"},
		{`"this is \u01F4A9"`, false, "ConstAtom<*>;StringAtom<\"this is \\u01F4A9\">;UnicodeCharAtom<\\u01F4>;CharAtom<A>;CharAtom<9>"},
		{`"esc\t\"x\"\n"`, false, "ConstAtom<*>;StringAtom<\"esc\\t\\\"x\\\"\\n\">;CharAtom<e>;CharAtom<s>;CharAtom<c>;EscapedCharAtom<\\t>;EscapedCharAtom<\\\">;CharAtom<x>;EscapedCharAtom<\\\">;EscapedCharAtom<\\n>"},
		{`"Hello, 世界! Привет!"`, false, "ConstAtom<*>;StringAtom<\"Hello, 世界! Привет!\">"},
		{`"some string`, true, "\nparse error near CharAtom (line 1 symbol 12 - line 1 symbol 13):\n\"g\"\n"},
		{`"Hello, 世界! Привет!"`, false, "ConstAtom<*>;StringAtom<\"Hello, 世界! Привет!\">"},
		{`Hello, 世界! Привет!"`, true, "\nparse error near IdentifierAtom (line 1 symbol 1 - line 1 symbol 6):\n\"Hello\"\n"},
		{`"Hello, 世界! Привет!`, true, "\nparse error near CharAtom (line 1 symbol 19 - line 1 symbol 20):\n\"!\"\n"},
		{`"Hello, 世界!" Привет!"`, true, "\nparse error near WS (line 1 symbol 13 - line 1 symbol 14):\n\" \"\n"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestInt(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`12345`, false, "ConstAtom<*>;IntegerAtom<12345>"},
		{`00000`, false, "ConstAtom<*>;IntegerAtom<00000>"},
		{`01abc`, true, "\nparse error near IntegerAtom (line 1 symbol 1 - line 1 symbol 3):\n\"01\"\n"},
		{`123!@#`, true, "\nparse error near IntegerAtom (line 1 symbol 1 - line 1 symbol 4):\n\"123\"\n"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestBoolean(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`true`, false, "ConstAtom<*>;BooleanAtom<true>"},
		{`false`, false, "ConstAtom<*>;BooleanAtom<false>"},
		{`trueFalse123`, false, "GettableExpr<*>;IdentifierAtom<trueFalse123>;ReservedWords<true>"},
		{`false&^(*`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 6):\n\"false\"\n"},
		{`true!@#`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 5):\n\"true\"\n"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestList(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`[]`, false, "ConstAtom<*>;ListAtom<[]>"},
		{`[[1], [[2], [3]]]`, false, "ConstAtom<*>;ListAtom<*>;ListAtom<*>;IntegerAtom<1>;ListAtom<*>;ListAtom<*>;IntegerAtom<2>;ListAtom<*>;IntegerAtom<3>"},
		{`[12]`, false, "ConstAtom<*>;ListAtom<*>;ConstAtom<*>;IntegerAtom<12>"},
		{`[12, true, "xxx"]`, false, "ConstAtom<*>;ListAtom<*>;ConstAtom<*>;IntegerAtom<12>;ConstAtom<*>;BooleanAtom<true>;ConstAtom<*>;StringAtom<\"xxx\">"},
		{`[12, [true, "xxx"]]`, false, "ConstAtom<*>;ListAtom<*>;ConstAtom<*>;IntegerAtom<12>;ListAtom<*>;ConstAtom<*>;BooleanAtom<true>;ConstAtom<*>;StringAtom<\"xxx\">"},
		{`[12, true "xxx"]`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`[12 true, "xxx"]`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`[12, true, "xxx"`, true, "\nparse error near StringAtom (line 1 symbol 12 - line 1 symbol 17):\n\"\\\"xxx\\\"\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near IntegerAtom (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near IntegerAtom (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestDefinitions(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`let x = 1`, false, "Declaration<*>;Variable<*>;IdentifierAtom<x>;ConstAtom<*>;IntegerAtom<1>"},
		{`let x = "xxx"`, false, "Declaration<*>;Variable<*>;IdentifierAtom<x>;ConstAtom<*>;StringAtom<\"xxx\">"},
		{`let x = y`, false, "Declaration<*>;Variable<*>;IdentifierAtom<x>;GettableExpr<*>;IdentifierAtom<y>"},
		{`let x = `, true, "\nparse error near WS (line 1 symbol 8 - line 1 symbol 9):\n\" \"\n"},
		{`let x y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let x == y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let x != y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let func = true `, true, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"func\"\n"},
		{`strict x = true `, false, "Declaration<*>;StrictVariable<*>;IdentifierAtom<x>;ConstAtom<*>;BooleanAtom<true>"},
		{`let strict = true `, true, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 11):\n\"strict\"\n"},
		{`func a() = 1`, false, "Declaration<*>;Func<*>;IdentifierAtom<a>;Expr<*>"},
		{`func aaa(a: Int, b: String, c: Boolean) = 1`, false, "Declaration<*>;Func<*>;IdentifierAtom<aaa>;FuncArgSeq<*>;FuncArg<*>;IdentifierAtom<a>;OneGenericTypeAtom<Int>;FuncArgSeq<*>;FuncArg<*>;IdentifierAtom<b>;OneGenericTypeAtom<String>;FuncArgSeq<*>;FuncArg<*>;IdentifierAtom<c>;OneGenericTypeAtom<Boolean>;Expr<*>"},
		{`func f a: Int) = a`, true, "\nparse error near WS (line 1 symbol 7 - line 1 symbol 8):\n\" \"\n"},
		{`func f(a: Int = a`, true, "\nparse error near WS (line 1 symbol 14 - line 1 symbol 15):\n\" \"\n"},
		{`func f(a: Int) a`, true, "\nparse error near WS (line 1 symbol 15 - line 1 symbol 16):\n\" \"\n"},
		{`func f(a Int) = a`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`func f(a, b, c) = a`, true, "\nparse error near IdentifierAtom (line 1 symbol 8 - line 1 symbol 9):\n\"a\"\n"},
		{`func f(a Int, b: String, c) a`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`func let() = true`, true, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 9):\n\"let\"\n"},
		{`let x = # xxx
true
`, false, "Declaration<*>;Variable<*>;IdentifierAtom<x>;Comment<*>;Expr<*>;BooleanAtom<true>"},
		{`func xxx 
					(a: Int, b: Int) = # xxx
true
`, false, "Declaration<*>;Func<*>;IdentifierAtom<xxx>;FuncArgSeq<*>;FuncArg<*>;IdentifierAtom<a>;OneGenericTypeAtom<Int>;FuncArgSeq<*>;FuncArg<*>;IdentifierAtom<b>;OneGenericTypeAtom<Int>;Comment<*>;Expr<*>;BooleanAtom<true>"},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
			checkAST(t, test.expected, ast, test.src)
		}
	}
}

func TestReservedWords(t *testing.T) {
	for _, test := range []struct {
		src      string
		expected string
	}{
		{`func let() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 9):\n\"let\"\n"},
		{`func strict() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 12):\n\"strict\"\n"},
		{`func base16() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 12):\n\"base16\"\n"},
		{`func base58() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 12):\n\"base58\"\n"},
		{`func base64() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 12):\n\"base64\"\n"},
		{`func true() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"true\"\n"},
		{`func false() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 11):\n\"false\"\n"},
		{`func if() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 8):\n\"if\"\n"},
		{`func then() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"then\"\n"},
		{`func else() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"else\"\n"},
		{`func match() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 11):\n\"match\"\n"},
		{`func case() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"case\"\n"},
		{`func func() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"func\"\n"},
		{`func FOLD() = true`, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 10):\n\"FOLD\"\n"},
		{`let let = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 8):\n\"let\"\n"},
		{`strict strict = true`, "\nparse error near ReservedWords (line 1 symbol 8 - line 1 symbol 14):\n\"strict\"\n"},
		{`let base16 = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 11):\n\"base16\"\n"},
		{`let base58 = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 11):\n\"base58\"\n"},
		{`let base64 = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 11):\n\"base64\"\n"},
		{`let true = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"true\"\n"},
		{`let false = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 10):\n\"false\"\n"},
		{`let if = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 7):\n\"if\"\n"},
		{`let then = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"then\"\n"},
		{`let else = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"else\"\n"},
		{`let match = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 10):\n\"match\"\n"},
		{`let case = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"case\"\n"},
		{`let func = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"func\"\n"},
		{`let FOLD = true`, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"FOLD\"\n"},
	} {
		_, _, err := buildAST(t, test.src, false)
		assert.EqualError(t, err, test.expected, test.src)
	}
}
