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
				if exp.value != "." {
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
		{`{-# STDLIB_VERSION 6 #-}`, false, "Directive<.>;DirectiveName<STDLIB_VERSION>;IntString<6>"},
		{`{-# STDLIB_VERSION XXX #-}`, false, "Directive<.>;DirectiveName<STDLIB_VERSION>;UpperCaseString<XXX>"},
		{`{-# CONTENT_TYPE DAPP #-}`, false, "Directive<.>;DirectiveName<CONTENT_TYPE>;UpperCaseString<DAPP>"},
		{`{-# SCRIPT_TYPE ACCOUNT #-}`, false, "Directive<.>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<ACCOUNT>"},
		{`{-#	SCRIPT_TYPE 	 ACCOUNT      #-}`, false, "Directive<.>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<ACCOUNT>"},
		{`{-# IMPORT lib1 #-}`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib1>"},
		{`{-# IMPORT lib1,my_lib2 #-}`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib1,my_lib2>"},
		{`{-# IMPORT lib3.ride,dir/lib4.ride #-}`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride,dir/lib4.ride>"},
		{`{-# STDLIB_version 123 #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"STDLIB_\"\n"},
		{`{-# NAME #-}`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`{-# 123 #-}`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`{-# CONTENT_TYPE account #-}`, false, "Directive<.>;DirectiveName<CONTENT_TYPE>;PathString<account>"},
		{`{-# CONTENT-TYPE ACCOUNT #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"CONTENT\"\n"},
		{`{-# IMPORT lib3.ride,dir\lib4.ride #-}`, true, "\nparse error near PathString (line 1 symbol 12 - line 1 symbol 25):\n\"lib3.ride,dir\"\n"},
		{`{-# IMPORT lib3.ride #-} # comment`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride>"},
		{`	{-# STDLIB_VERSION 6 #-}
				{-# IMPORT lib3.ride,lib4.ride #-} # comment
				{-# CONTENT_TYPE ACCOUNT #-}
				{-# SCRIPT_TYPE DAPP #-}
			`, false,
			"Directive<.>;DirectiveName<STDLIB_VERSION>;IntString<6>;" +
				"Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride,lib4.ride>;" +
				"Directive<.>;DirectiveName<CONTENT_TYPE>;UpperCaseString<ACCOUNT>;" +
				"Directive<.>;DirectiveName<SCRIPT_TYPE>;UpperCaseString<DAPP>"},
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
		{`base16''`, false, "Const<.>;ByteVector<.>;Base16<base16''>"},
		{`base58''`, false, "Const<.>;ByteVector<.>;Base58<base58''>"},
		{`base64''`, false, "Const<.>;ByteVector<.>;Base64<base64''>"},
		{`base16'cafeBEBE12345'`, false, "Const<.>;ByteVector<.>;Base16<base16'cafeBEBE12345'>"},
		{`base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'`, false, "Const<.>;ByteVector<.>;Base58<base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'>"},
		{`base64'SGVsbG8gd29ybGQhISE='`, false, "Const<.>;ByteVector<.>;Base64<base64'SGVsbG8gd29ybGQhISE='>"},
		{`base64'SGVsbG8gd29ybGQhIQ=='`, false, "Const<.>;ByteVector<.>;Base64<base64'SGVsbG8gd29ybGQhIQ=='>"},
		{`base16'cafeBEBE12345'`, false, "Const<.>;ByteVector<.>;Base16<base16'cafeBEBE12345'>"},
		{`base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'`, false, "Const<.>;ByteVector<.>;Base58<base58'3aU8VJHZeWTaNLXCDwaDuqairhwih1Vf3PKgn3H98xXcTxM3Y9ePxbpX4f3ByhatR2Z8ouRgagiMNAEgzavbbG3m'>"},
		{`base64'SGVsbG8gd29ybGQhISE='`, false, "Const<.>;ByteVector<.>;Base64<base64'SGVsbG8gd29ybGQhISE='>"},
		{`base64'SGVsbG8gd29ybGQhIQ=='`, false, "Const<.>;ByteVector<.>;Base64<base64'SGVsbG8gd29ybGQhIQ=='>"},
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
		{`"some string"`, false, "Const<.>;String<\"some string\">"},
		{`"this is \u01F4A9"`, false, "Const<.>;String<\"this is \\u01F4A9\">;UnicodeChar<\\u01F4>;Char<A>;Char<9>"},
		{`"esc\t\"x\"\n"`, false, "Const<.>;String<\"esc\\t\\\"x\\\"\\n\">;Char<e>;Char<s>;Char<c>;EscapedChar<\\t>;EscapedChar<\\\">;Char<x>;EscapedChar<\\\">;EscapedChar<\\n>"},
		{`"Hello, 世界! Привет!"`, false, "Const<.>;String<\"Hello, 世界! Привет!\">"},
		{`"some string`, true, "\nparse error near Char (line 1 symbol 12 - line 1 symbol 13):\n\"g\"\n"},
		{`"Hello, 世界! Привет!"`, false, "Const<.>;String<\"Hello, 世界! Привет!\">"},
		{`Hello, 世界! Привет!"`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 6):\n\"Hello\"\n"},
		{`"Hello, 世界! Привет!`, true, "\nparse error near Char (line 1 symbol 19 - line 1 symbol 20):\n\"!\"\n"},
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
		{`12345`, false, "Const<.>;Integer<12345>"},
		{`00000`, false, "Const<.>;Integer<00000>"},
		{`01abc`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 3):\n\"01\"\n"},
		{`123!@#`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 4):\n\"123\"\n"},
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
		{`true`, false, "Const<.>;Boolean<true>"},
		{`false`, false, "Const<.>;Boolean<false>"},
		{`trueFalse123`, false, "GettableExpr<.>;Identifier<trueFalse123>;ReservedWords<true>"},
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
		{`[]`, false, "Const<.>;List<[]>"},
		{`[[1], [[2], [3]]]`, false, "Const<.>;List<.>;List<.>;Integer<1>;List<.>;List<.>;Integer<2>;List<.>;Integer<3>"},
		{`[12]`, false, "Const<.>;List<.>;Const<.>;Integer<12>"},
		{`[12, true, "xxx"]`, false, "Const<.>;List<.>;Const<.>;Integer<12>;Const<.>;Boolean<true>;Const<.>;String<\"xxx\">"},
		{`[12, [true, "xxx"]]`, false, "Const<.>;List<.>;Const<.>;Integer<12>;List<.>;Const<.>;Boolean<true>;Const<.>;String<\"xxx\">"},
		{`[12, true "xxx"]`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`[12 true, "xxx"]`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`[12, true, "xxx"`, true, "\nparse error near String (line 1 symbol 12 - line 1 symbol 17):\n\"\\\"xxx\\\"\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
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
		{`let x = 1`, false, "Declaration<.>;Variable<.>;Identifier<x>;Const<.>;Integer<1>"},
		{`let x = "xxx"`, false, "Declaration<.>;Variable<.>;Identifier<x>;Const<.>;String<\"xxx\">"},
		{`let x = y`, false, "Declaration<.>;Variable<.>;Identifier<x>;GettableExpr<.>;Identifier<y>"},
		{`let x = `, true, "\nparse error near WS (line 1 symbol 8 - line 1 symbol 9):\n\" \"\n"},
		{`let x y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let x == y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let x != y `, true, "\nparse error near WS (line 1 symbol 6 - line 1 symbol 7):\n\" \"\n"},
		{`let func = true `, true, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 9):\n\"func\"\n"},
		{`strict x = true `, false, "Declaration<.>;StrictVariable<.>;Identifier<x>;Const<.>;Boolean<true>"},
		{`let strict = true `, true, "\nparse error near ReservedWords (line 1 symbol 5 - line 1 symbol 11):\n\"strict\"\n"},
		{`func a() = 1`, false, "Declaration<.>;Func<.>;Identifier<a>;Expr<.>"},
		{`func aaa(a: Int, b: String, c: Boolean) = 1`, false, "Declaration<.>;Func<.>;Identifier<aaa>;FuncArgSeq<.>;FuncArg<.>;Identifier<a>;OneGenericTypeAtom<Int>;FuncArgSeq<.>;FuncArg<.>;Identifier<b>;OneGenericTypeAtom<String>;FuncArgSeq<.>;FuncArg<.>;Identifier<c>;OneGenericTypeAtom<Boolean>;Expr<.>"},
		{`func f a: Int) = a`, true, "\nparse error near WS (line 1 symbol 7 - line 1 symbol 8):\n\" \"\n"},
		{`func f(a: Int = a`, true, "\nparse error near WS (line 1 symbol 14 - line 1 symbol 15):\n\" \"\n"},
		{`func f(a: Int) a`, true, "\nparse error near WS (line 1 symbol 15 - line 1 symbol 16):\n\" \"\n"},
		{`func f(a Int) = a`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`func f(a, b, c) = a`, true, "\nparse error near Identifier (line 1 symbol 8 - line 1 symbol 9):\n\"a\"\n"},
		{`func f(a Int, b: String, c) a`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`func let() = true`, true, "\nparse error near ReservedWords (line 1 symbol 6 - line 1 symbol 9):\n\"let\"\n"},
		{`let x = # xxx
true
`, false, "Declaration<.>;Variable<.>;Identifier<x>;Comment<.>;Expr<.>;Boolean<true>"},
		{`func xxx 
					(a: Int, b: Int) = # xxx
true
`, false, "Declaration<.>;Func<.>;Identifier<xxx>;FuncArgSeq<.>;FuncArg<.>;Identifier<a>;OneGenericTypeAtom<Int>;FuncArgSeq<.>;FuncArg<.>;Identifier<b>;OneGenericTypeAtom<Int>;Comment<.>;Expr<.>;Boolean<true>"},
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

func TestMathExpressions(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`1 + 2`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>"},
		{`1 - 2`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SubOp<->;Const<.>;Integer<2>"},
		{`1 * 2`, false, "Expr<.>;MultGroupOpAtom<.>;Const<.>;Integer<1>;MultGroupOp<.>;MulOp<*>;Const<.>;Integer<2>"},
		{`1 / 2`, false, "Expr<.>;MultGroupOpAtom<.>;Const<.>;Integer<1>;MultGroupOp<.>;DivOp</>;Const<.>;Integer<2>"},
		{`1 + 2 + 3`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<3>"},
		{`1+2+3`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<3>"},
		{`1 + (2 * 3)`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;MultGroupOpAtom<.>;Const<.>;Integer<2>;MultGroupOp<.>;MulOp<*>;Const<.>;Integer<3>"},
		{`1 + (2 * 3) / 4`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;MultGroupOpAtom<.>;MultGroupOpAtom<.>;Const<.>;Integer<2>;MultGroupOp<.>;MulOp<*>;Const<.>;Integer<3>;MultGroupOp<.>;DivOp</>;Const<.>;Integer<4>"},
		{`(1 + 2) * (3 - 4)`, false, "Expr<.>;MultGroupOpAtom<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;MultGroupOp<.>;MulOp<*>;SumGroupOpAtom<.>;Const<.>;Integer<3>;SumGroupOp<.>;SubOp<->;Const<.>;Integer<4>"},
		{`(1) + (2) * (3) - (4)`, false, "Expr<.>;MultGroupOpAtom<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;MultGroupOp<.>;MulOp<*>;SumGroupOpAtom<.>;Const<.>;Integer<3>;SumGroupOp<.>;SubOp<->;Const<.>;Integer<4>"},
		{`((1) + ((2) * (3)) - (4))`, false, "Expr<.>;MultGroupOpAtom<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;MultGroupOp<.>;MulOp<*>;SumGroupOpAtom<.>;Const<.>;Integer<3>;SumGroupOp<.>;SubOp<->;Const<.>;Integer<4>"},
		{`(1 + 2) * (3 - 4`, true, "\nparse error near Integer (line 1 symbol 16 - line 1 symbol 17):\n\"4\"\n"},
		{`(1  2) * (3 - 4)`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`(12 * (3 - 4)`, true, "\nparse error near ParExpr (line 1 symbol 7 - line 1 symbol 14):\n\"(3 - 4)\"\n"},
		{`(1 + 2) (3 - 4)`, true, "\nparse error near WS (line 1 symbol 8 - line 1 symbol 9):\n\" \"\n"},
		{`1 +    2      
					+ 3`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<3>"},
		{`1 +    2      
					- 3`, false, "Expr<.>;SumGroupOpAtom<.>;Const<.>;Integer<1>;SumGroupOp<.>;SumOp<+>;Const<.>;Integer<2>;SumGroupOp<.>;SubOp<->;Const<.>;Integer<3>"},
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

func TestExpressionSeparator(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`let x = {let y = true; y }; x`, false, "Declaration<.>;Variable<.>;Identifier<x>;Expr<.>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<true>;GettableExpr<.>;Identifier<y>;GettableExpr<.>;Identifier<x>"},
		{`let x = {let y = true; let z = true; y || z}; x`, false, "Declaration<.>;Variable<.>;Identifier<x>;Expr<.>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<true>;Declaration<.>;Variable<.>;Identifier<z>;Expr<.>;Const<.>;Boolean<true>;Expr<.>;Identifier<y>;OrOp<.>;Identifier<z>;GettableExpr<.>;Identifier<x>"},
		{`func f() = true; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Expr<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`func f() = {true}; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Block<.>;Expr<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`func f() = {let y = false; true}; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<false>;Const<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`func f() = {let y = false; func b() = y; true}; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<false>;Declaration<.>;Func<.>;Identifier<b>;Const<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`func f() = {let y = false ; func b() = y  ; true}		; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<false>;Declaration<.>;Func<.>;Identifier<b>;Const<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`func f() = {let y = false
		; func b() = y
		; true}
		; f()`, false, "Declaration<.>;Func<.>;Identifier<f>;Block<.>;Declaration<.>;Variable<.>;Identifier<y>;Expr<.>;Const<.>;Boolean<false>;Declaration<.>;Func<.>;Identifier<b>;Const<.>;Boolean<true>;FunctionCall<.>;Identifier<f>"},
		{`let x = {let y = true; ; y }; x`, true, "\nparse error near WS (line 1 symbol 23 - line 1 symbol 24):\n\" \"\n"},
		{`let x = {let y = true; y }; x;`, true, "\nparse error near Identifier (line 1 symbol 29 - line 1 symbol 30):\n\"x\"\n"},
		{`;let x = true`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
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

func TestIdentifiers(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`let123`, false, "Identifier<let123>"},
		{`abc`, false, "Identifier<abc>"},
		{`ABC`, false, "Identifier<ABC>"},
		{`Abc`, false, "Identifier<Abc>"},
		{`aBc`, false, "Identifier<aBc>"},
		{`a_b_c`, false, "Identifier<a_b_c>"},
		// TODO: Consecutive underscores are not allowed in Scala parser V1, consider adding style rule on this.
		{`A__B___C`, false, "Identifier<A__B___C>"},
		{`_a_b_c`, false, "Identifier<_a_b_c>"},
		{`let_123`, false, "Identifier<let_123>"},
		{`let`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 4):\n\"let\"\n"},
		{`let 123 = true`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
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
