package compiler

import (
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
}

func TestStringDirectives(t *testing.T) {
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
			checkAST(t, test.expected, ast, test.src)
		}
	}
}
