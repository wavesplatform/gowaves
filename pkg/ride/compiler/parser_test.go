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
		require.Fail(t, fmt.Sprintf("invalid node expectation %q", s))
	}
	name := strings.TrimSpace(s[:valueBegin])
	value := strings.TrimSpace(s[valueBegin+1 : valueEnd])
	if name == "" || value == "" {
		require.Fail(t, fmt.Sprintf("invalid node expectation %q", s))
	}
	return nodeExpectation{name: name, value: value}
}

func checkAST(t *testing.T, expected string, ast *node32, buffer string) {
	exps := make([]nodeExpectation, 0)
	for _, s := range strings.Split(strings.TrimSuffix(expected, ";"), ";") {
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
		{`{-# IMPORT lib1,my_lib2 #-}`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib1>;PathString<my_lib2>"},
		{`{-# IMPORT lib3.ride,dir/lib4.ride #-}`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride>;PathString<dir/lib4.ride>"},
		{`{-# STDLIB_version 123 #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"STDLIB_\"\n"},
		{`{-# NAME #-}`, true, "\nparse error near WS (line 1 symbol 9 - line 1 symbol 10):\n\" \"\n"},
		{`{-# 123 #-}`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`{-# CONTENT_TYPE account #-}`, false, "Directive<.>;DirectiveName<CONTENT_TYPE>;PathString<account>"},
		{`{-# CONTENT-TYPE ACCOUNT #-}`, true, "\nparse error near DirectiveName (line 1 symbol 5 - line 1 symbol 12):\n\"CONTENT\"\n"},
		{`{-# IMPORT lib3.ride,dir\lib4.ride #-}`, true, "\nparse error near PathString (line 1 symbol 22 - line 1 symbol 25):\n\"dir\"\n"},
		{`{-# IMPORT lib3.ride #-} # comment`, false, "Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride>"},
		{`	{-# STDLIB_VERSION 6 #-}
				{-# IMPORT lib3.ride,lib4.ride #-} # comment
				{-# CONTENT_TYPE ACCOUNT #-}
				{-# SCRIPT_TYPE DAPP #-}
			`, false,
			"Directive<.>;DirectiveName<STDLIB_VERSION>;IntString<6>;" +
				"Directive<.>;DirectiveName<IMPORT>;PathString<lib3.ride>;PathString<lib4.ride>;" +
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
		{`[]`, false, "List<[]>"},
		{`[[1], [[2], [3]]]`, false, "List<.>;List<.>;Integer<1>;List<.>;List<.>;Integer<2>;List<.>;Integer<3>"},
		{`[12]`, false, "List<.>;Const<.>;Integer<12>"},
		{`[12, true, "xxx"]`, false, "List<.>;Const<.>;Integer<12>;Const<.>;Boolean<true>;Const<.>;String<\"xxx\">"},
		{`[12, [true, "xxx"]]`, false, "List<.>;Const<.>;Integer<12>;List<.>;Const<.>;Boolean<true>;Const<.>;String<\"xxx\">"},
		{`[12, true "xxx"]`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`[12 true, "xxx"]`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`[12, true, "xxx"`, true, "\nparse error near String (line 1 symbol 12 - line 1 symbol 17):\n\"\\\"xxx\\\"\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
		{`12, true, "xxx"]`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 3):\n\"12\"\n"},
		{`let x = []`, false, "Variable<.>;Identifier<x>;List<.>"},
		{`let x = [1, 2, 3]`, false, "Variable<.>;Identifier<x>;List<.>;Integer<1>;Integer<2>;Integer<3>"},
		{`x[0]`, false, "GettableExpr<.>;Identifier<x>;ListAccess<.>;Const<.>;Integer<0>"},
		{`[1, 2, 3][0]`, false, "GettableExpr<.>;List<.>;Integer<1>;Integer<2>;Integer<3>;ListAccess<.>;Const<.>;Integer<0>"},
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
		{`func aaa(a: Int, b: String, c: Boolean) = 1`, false, "Declaration<.>;Func<.>;Identifier<aaa>;FuncArgSeq<.>;FuncArg<.>;Identifier<a>;Type<Int>;FuncArgSeq<.>;FuncArg<.>;Identifier<b>;Type<String>;FuncArgSeq<.>;FuncArg<.>;Identifier<c>;Type<Boolean>;Expr<.>"},
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
`, false, "Declaration<.>;Func<.>;Identifier<xxx>;FuncArgSeq<.>;FuncArg<.>;Identifier<a>;Type<Int>;FuncArgSeq<.>;FuncArg<.>;Identifier<b>;Type<Int>;Comment<.>;Expr<.>;Boolean<true>"},
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
		{`a_b_c_`, false, "Identifier<a_b_c_>"},
		{`_a_b_c`, false, "Identifier<_a_b_c>"},
		{`_a_b_c_`, false, "Identifier<_a_b_c_>"},
		{`_a2`, false, "Identifier<_a2>"},
		{`_a_2`, false, "Identifier<_a_2>"},
		{`_a_2_`, false, "Identifier<_a_2_>"},
		{`let_123`, false, "Identifier<let_123>"},
		{`let_1_a_b_c`, false, "Identifier<let_1_a_b_c>"},
		{`let_1_a_b_c_`, false, "Identifier<let_1_a_b_c_>"},
		{`let`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 4):\n\"let\"\n"},
		{`let__1_a_b_c_`, true, "\nparse error near ReservedWords (line 1 symbol 1 - line 1 symbol 4):\n\"let\"\n"},
		{`let_1_a__b_c_`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 8):\n\"let_1_a\"\n"},
		{`let_1_a_b_c__`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 12):\n\"let_1_a_b_c\"\n"},
		{`A__B___C`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 2):\n\"A\"\n"},
		{`let 123 = true`, true, "\nparse error near WS (line 1 symbol 4 - line 1 symbol 5):\n\" \"\n"},
		{`_1two`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
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

func TestAnnotations(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`@Annotation(foo)func f() = []`, false, "AnnotatedFunc<.>;AnnotationSeq<.>;Annotation<.>;Identifier<Annotation>;IdentifierSeq<.>;Identifier<foo>;Func<.>;Identifier<f>"},
		{`@Annotation(foo) @Notation(bar, baz) func f() = []`, false, "AnnotatedFunc<.>;AnnotationSeq<.>;Annotation<.>;Identifier<Annotation>;IdentifierSeq<.>;Identifier<foo>;Annotation<.>;Identifier<Notation>;IdentifierSeq<.>;Identifier<bar>;Identifier<baz>;Func<.>;Identifier<f>"},
		{`@Annotation()func f() = []`, true, "\nparse error near Identifier (line 1 symbol 2 - line 1 symbol 12):\n\"Annotation\"\n"},
		{`@(x)func f() = []`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
		{`@ func f() = []`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
		{`@func func f() = []`, true, "\nparse error near ReservedWords (line 1 symbol 2 - line 1 symbol 6):\n\"func\"\n"},
		{`@@Annotation(foo)func f() = []`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
		{`@Annotation(foo func f() = []`, true, "\nparse error near WS (line 1 symbol 16 - line 1 symbol 17):\n\" \"\n"},
		{`@Annotation foo) func f() = []`, true, "\nparse error near WS (line 1 symbol 12 - line 1 symbol 13):\n\" \"\n"},
		{`@Annotation(foo) @Notation(bar,) func f() = []`, true, "\nparse error near Identifier (line 1 symbol 28 - line 1 symbol 31):\n\"bar\"\n"},
		{`@Annotation(foo @Notation bar,baz) func f() = []`, true, "\nparse error near WS (line 1 symbol 16 - line 1 symbol 17):\n\" \"\n"},
		{`@Annotation(foo, @Notation, bar,baz) func f() = []`, true, "\nparse error near WS (line 1 symbol 17 - line 1 symbol 18):\n\" \"\n"},
		{`@Annotation(foo #comment
)  #comment
#comment
func #comment
f( a: #comment
Type #comment
) #comment
= []`, false, "AnnotatedFunc<.>;AnnotationSeq<.>;Annotation<.>;Identifier<Annotation>;IdentifierSeq<.>;Identifier<foo>;Func<.>;Identifier<f>"},
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

func TestHugeScript(t *testing.T) {
	sb := strings.Builder{}
	for i := 0; i < 10000; i++ {
		sb.WriteString(fmt.Sprintf("let i%d = true\n", i))
	}
	sb.WriteString("i9999\n")
	ast, _, err := buildAST(t, sb.String(), false)
	require.Nil(t, err)
	require.NotNil(t, ast)
}

func TestUnderscoreInNumbers(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`1000000`, false, "Integer<1000000>"},
		{`1_000_000`, false, "Integer<1_000_000>"},
		{`1_0_0_0_0_0_0`, false, "Integer<1_0_0_0_0_0_0>"},
		{`1_0_0_0_0_0_0_`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 14):\n\"1_0_0_0_0_0_0\"\n"},
		{`100__000`, true, "\nparse error near Integer (line 1 symbol 1 - line 1 symbol 4):\n\"100\"\n"},
		{`_100`, true, "\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n"},
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

func TestDeclarationsOrder(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`let x = 1 @Callable(i) func f()=[]`, false, "Declaration<.>;Variable<.>;Identifier<x>;AnnotatedFunc<.>;Annotation<.>;Identifier<Callable>;Identifier<i>;Func<.>;Identifier<f>"},
		{`@Callable(i) func f()=[] let x = 1`, true, "\nparse error near WS (line 1 symbol 25 - line 1 symbol 26):\n\" \"\n"},
		{`func a() = 1 @Callable(i) func f()=[]`, false, "Declaration<.>;Func<.>;Identifier<a>;AnnotatedFunc<.>;Annotation<.>;Identifier<Callable>;Identifier<i>;Func<.>;Identifier<f>"},
		{`@Callable(i) func f()=[] func a() = 1`, true, "\nparse error near WS (line 1 symbol 25 - line 1 symbol 26):\n\" \"\n"},
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

func TestListOpExpressions(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`1 :: list`, false, "Expr<.>;Const<.>;Integer<1>;ConsOp<::>;GettableExpr<.>;Identifier<list>"},
		{`list :+ 1`, false, "Expr<.>;GettableExpr<.>;Identifier<list>;AppendOp<:+>;Const<.>;Integer<1>"},
		{`list1 ++ list2`, false, "Expr<.>;GettableExpr<.>;Identifier<list1>;ConcatOp<++>;GettableExpr<.>;Identifier<list2>"},
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

func TestTypes(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`func f(x: Int) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;Type<Int>"},
		{`func f(x: Int|String) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;Type<Int>;Types<.>;Type<String>"},
		{`func f(x: List[String]) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;GenericType<.>;Type<List>;Types<.>;Type<String>"},
		{`func f(x: List[String | Int]) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;GenericType<.>;Type<List>;Types<.>;Type<String>;Types<.>;Type<Int>"},
		{`func f(x: List[List[String]]) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;GenericType<.>;Type<List>;Types<.>;GenericType<.>;Type<List>;Types<.>;Type<String>"},
		{`func f(x: (Int, String)) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;TupleType<.>;Type<Int>;Types<.>;Type<String>"},
		{`func f(x: List[(Int, String)]) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;GenericType<.>;Type<List>;Types<.>;TupleType<.>;Types<.>;Type<Int>;Types<.>;Type<String>"},
		{`func f(x: List[Boolean | (Int, String)]) = true`, false, "FuncArgSeq<.>;FuncArg<.>;Types<.>;GenericType<.>;Type<List>;Types<.>;Type<Boolean>;Types<.>;TupleType<.>;Types<.>;Type<Int>;Types<.>;Type<String>"},
		{`func f(x: |Int|String) = true`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`func f(x: Int||String) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 14):\n\"Int\"\n"},
		{`func f(x: Int|String|) = true`, true, "\nparse error near Type (line 1 symbol 15 - line 1 symbol 21):\n\"String\"\n"},
		{`func f(x: ListString]) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 21):\n\"ListString\"\n"},
		{`func f(x: List String]) = true`, true, "\nparse error near WS (line 1 symbol 15 - line 1 symbol 16):\n\" \"\n"},
		{`func f(x: List[String) = true`, true, "\nparse error near Type (line 1 symbol 16 - line 1 symbol 22):\n\"String\"\n"},
		{`func f(x: List[String)) = true`, true, "\nparse error near Type (line 1 symbol 16 - line 1 symbol 22):\n\"String\"\n"},
		{`func f(x: List(String]) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 15):\n\"List\"\n"},
		{`func f(x: List(String)) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 15):\n\"List\"\n"},
		{`func f(x: List[String  Int]) = true`, true, "\nparse error near WS (line 1 symbol 23 - line 1 symbol 24):\n\" \"\n"},
		{`func f(x: List[String || Int]) = true`, true, "\nparse error near WS (line 1 symbol 22 - line 1 symbol 23):\n\" \"\n"},
		{`func f(x: List[ | String | Int]) = true`, true, "\nparse error near WS (line 1 symbol 16 - line 1 symbol 17):\n\" \"\n"},
		{`func f(x: List[String | Int |]) = true`, true, "\nparse error near WS (line 1 symbol 28 - line 1 symbol 29):\n\" \"\n"},
		{`func f(x: (Int String)) = true`, true, "\nparse error near WS (line 1 symbol 15 - line 1 symbol 16):\n\" \"\n"},
		{`func f(x: (,Int, String)) = true`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`func f(x: (Int, String, )) = true`, true, "\nparse error near WS (line 1 symbol 24 - line 1 symbol 25):\n\" \"\n"},
		{`func f(x: Int, String)) = true`, true, "\nparse error near Identifier (line 1 symbol 16 - line 1 symbol 22):\n\"String\"\n"},
		{`func f(x: (Int, String) = true`, true, "\nparse error near WS (line 1 symbol 24 - line 1 symbol 25):\n\" \"\n"},
		{`func f(x: List[Int, String)]) = true`, true, "\nparse error near Type (line 1 symbol 16 - line 1 symbol 19):\n\"Int\"\n"},
		{`func f(x: List[(Int String)]) = true`, true, "\nparse error near WS (line 1 symbol 20 - line 1 symbol 21):\n\" \"\n"},
		{`func f(x: List[(,Int, String)]) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 15):\n\"List\"\n"},
		{`func f(x: List[(Int, String,)]) = true`, true, "\nparse error near Type (line 1 symbol 22 - line 1 symbol 28):\n\"String\"\n"},
		{`func f(x: List[(Int, String]) = true`, true, "\nparse error near Type (line 1 symbol 22 - line 1 symbol 28):\n\"String\"\n"},
		{`func f(x: List([Int, String]) = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 15):\n\"List\"\n"},
		{`func f(x: List([Int, String)] = true`, true, "\nparse error near Type (line 1 symbol 11 - line 1 symbol 15):\n\"List\"\n"},
		{`func f(x: List[Boolean | | (Int, String)]) = true`, true, "\nparse error near WS (line 1 symbol 25 - line 1 symbol 26):\n\" \"\n"},
		{`func f(x: List[Boolean  (Int, String)]) = true`, true, "\nparse error near WS (line 1 symbol 24 - line 1 symbol 25):\n\" \"\n"},
		{`func f(x: List[Boolean | (Int | String)]) = true`, true, "\nparse error near Type (line 1 symbol 33 - line 1 symbol 39):\n\"String\"\n"},
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

func TestTuples(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`func f() = (true, "xxx", 123)`, false, "Expr<.>;Tuple<.>;Const<.>;Boolean<true>;Const<.>;String<\"xxx\">;Const<.>;Integer<123>"},
		{`func f() = {let x = true; (x, "xxx", 123)}`, false, "Expr<.>;Expr<.>;Tuple<.>;GettableExpr<.>;Identifier<x>;Const<.>;String<\"xxx\">;Const<.>;Integer<123>"},
		{`func f() = (x, 123, {1+1}, (1, 2))`, false, "Expr<.>;Tuple<.>;GettableExpr<.>;Identifier<x>;Const<.>;Integer<123>;Block<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;Integer<2>"},
		{`func f() = (x, 123, {1+1}, (1, 2))`, false, "Expr<.>;Tuple<.>;GettableExpr<.>;Identifier<x>;Const<.>;Integer<123>;Block<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;Integer<2>"},
		{`let (a, b, c) = f()`, false, "Declaration<.>;Variable<.>;TupleRef<.>;Identifier<a>;Identifier<b>;Identifier<c>;Expr<.>"},
		{`strict (a, b, c) = f()`, false, "Declaration<.>;StrictVariable<.>;TupleRef<.>;Identifier<a>;Identifier<b>;Identifier<c>;Expr<.>"},
		{`x._1`, false, "Expr<.>;GettableExpr<.>;Identifier<x>;TupleAccess<_1>"},
		{`func f() = (1, 2, 4`, true, "\nparse error near Integer (line 1 symbol 19 - line 1 symbol 20):\n\"4\"\n"},
		{`func f() = 1, 2, 4`, true, "\nparse error near Integer (line 1 symbol 12 - line 1 symbol 13):\n\"1\"\n"},
		{`func f() = 1, 2, 4)`, true, "\nparse error near Integer (line 1 symbol 12 - line 1 symbol 13):\n\"1\"\n"},
		{`func f() = (1 2, 4)`, true, "\nparse error near WS (line 1 symbol 14 - line 1 symbol 15):\n\" \"\n"},
		{`func f() = (1, 2 4)`, true, "\nparse error near WS (line 1 symbol 17 - line 1 symbol 18):\n\" \"\n"},
		{`func f() = (1, (2, 4)`, true, "\nparse error near Tuple (line 1 symbol 16 - line 1 symbol 22):\n\"(2, 4)\"\n"},
		{`func f() = (1, 2, 4))`, true, "\nparse error near Tuple (line 1 symbol 12 - line 1 symbol 21):\n\"(1, 2, 4)\"\n"},
		{`func f() = (1, (2 4))`, true, "\nparse error near WS (line 1 symbol 18 - line 1 symbol 19):\n\" \"\n"},
		{`let a, b, c = f()`, true, "\nparse error near Identifier (line 1 symbol 5 - line 1 symbol 6):\n\"a\"\n"},
		{`let (a, b, c = f()`, true, "\nparse error near WS (line 1 symbol 13 - line 1 symbol 14):\n\" \"\n"},
		{`let a, b, c) = f()`, true, "\nparse error near Identifier (line 1 symbol 5 - line 1 symbol 6):\n\"a\"\n"},
		{`let (a b, c) = f()`, true, "\nparse error near WS (line 1 symbol 7 - line 1 symbol 8):\n\" \"\n"},
		{`let (a, b c) = f()`, true, "\nparse error near WS (line 1 symbol 10 - line 1 symbol 11):\n\" \"\n"},
		{`let (a, (b, c)) = f()`, true, "\nparse error near WS (line 1 symbol 8 - line 1 symbol 9):\n\" \"\n"},
		{`x.__1`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 2):\n\"x\"\n"},
		{`x._1_`, true, "\nparse error near TupleAccess (line 1 symbol 3 - line 1 symbol 5):\n\"_1\"\n"},
		{`x.1`, true, "\nparse error near Identifier (line 1 symbol 1 - line 1 symbol 2):\n\"x\"\n"},
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

func TestMatching(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`match false {case x: Boolean => true case _ => false }`, false, "Match<.>;Const<.>;Boolean<false>;Case<.>;ValuePattern<.>;Identifier<x>;Type<Boolean>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match false {case _: Boolean => true case _ => false }`, false, "Match<.>;Const<.>;Boolean<false>;Case<.>;ValuePattern<.>;Placeholder<.>;Type<Boolean>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case x: (Int, String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;ValuePattern<.>;Identifier<x>;TupleType<.>;Type<Int>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case _: (Int, String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;ValuePattern<.>;Placeholder<.>;TupleType<.>;Type<Int>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (x: Int, y: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Identifier<x>;Type<Int>;Identifier<y>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_: Int, y: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Type<Int>;Identifier<y>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_: Int, _: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Type<Int>;Placeholder<.>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (x: Int, _: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Identifier<x>;Type<Int>;Placeholder<.>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (x: Int, _) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Identifier<x>;Type<Int>;Placeholder<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_: Int, _) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Type<Int>;Placeholder<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_, y: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Identifier<y>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_, _: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Placeholder<.>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_, _) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Placeholder<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (x: Int, "y") => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Identifier<x>;Type<Int>;Const<.>;String<\"y\">;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_: Int, "y") => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Type<Int>;Const<.>;String<\"y\">;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (_, "y") => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Placeholder<.>;Const<.>;String<\"y\">;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (100, y: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Const<.>;Integer<100>;Identifier<y>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (100, _: String) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Const<.>;Integer<100>;Placeholder<.>;Type<String>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (100, _) => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Const<.>;Integer<100>;Placeholder<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (100, "y") => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Const<.>;Integer<100>;Const<.>;String<\"y\">;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match (1, "x") {case (1+2, "y"+"z") => true case _ => false }`, false, "Match<.>;Tuple<.>;Const<.>;Integer<1>;Const<.>;String<\"x\">;Case<.>;TuplePattern<.>;Expr<.>;Const<.>;Integer<1>;Const<.>;Integer<2>;Expr<.>;Const<.>;String<\"y\">;Const<.>;String<\"z\">;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = 1+2) => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;Expr<1+2>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = x) => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;Identifier<x>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = {let z = 1; z})  => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;Block<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = [1, 2, 3][0])  => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;List<.>;ListAccess<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = {f()})  => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;Block<.>;FunctionCall<.>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
		{`match tx {case LeaseCancelTransaction(leaseId = base58'', fee = {x})  => true case _ => false }`, false, "Match<.>;Identifier<tx>;Case<.>;ObjectPattern<.>;Identifier<LeaseCancelTransaction>;ObjectFieldsPattern<.>;Identifier<leaseId>;Base58<.>;ObjectFieldsPattern<.>;Identifier<fee>;Block<.>;Identifier<x>;BlockWithoutPar<.>;Case<.>;Placeholder<.>;BlockWithoutPar<.>"},
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

func TestBasicExpressions(t *testing.T) {
	for _, test := range []struct {
		src      string
		fail     bool
		expected string
	}{
		{`1 == 0 || 3 == 2`, false, "Expr<.>;Const<1>;EqOp<.>;Const<0>;OrOp<.>;Const<3>;EqOp<.>;Const<2>"},
		{`1 == 0 => 3 == 2`, true, "\nparse error near WS (line 1 symbol 7 - line 1 symbol 8):\n\" \"\n"},
		{`1 => 0`, true, "\nparse error near WS (line 1 symbol 2 - line 1 symbol 3):\n\" \"\n"},
		{`3 + 2 > 2 + 1`, false, "Expr<.>;Const<3>;SumOp<.>;Const<2>;GtOp<.>;Const<2>;SumOp<.>;Const<1>"},
		{`1 >= 0 || 3 > 2`, false, "Expr<.>;Const<1>;GeOp<.>;Const<0>;OrOp<.>;Const<3>;GtOp<.>;Const<2>"},
		{`false || sigVerify(base64'TElLRQ==', base58'222', base16'abcdf1')`, false, "Expr<.>;Boolean<false>;OrOp<.>;FunctionCall<.>;Identifier<.>;Base64<base64'TElLRQ=='>;Base58<base58'222'>;Base16<base16'abcdf1'>"},
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

func TestScope(t *testing.T) {
	for _, test := range []struct {
		src  string
		fail bool
	}{
		{`let a = {func bar(i: Int) = i; bar(1)};let b = {func bar(i: Int) = i;bar(a)}`, false},
	} {
		ast, _, err := buildAST(t, test.src, false)
		if test.fail {
			assert.Error(t, err, test.src)
		} else {
			require.Nil(t, err)
			require.NotNil(t, ast)
		}
	}
}
