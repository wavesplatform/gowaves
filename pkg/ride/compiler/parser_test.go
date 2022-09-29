package compiler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildAST(t *testing.T, src string) (*node32, []rune, error) {
	p := Parser{Buffer: src}
	err := p.Init()
	require.NoError(t, err)
	err = p.Parse()
	if err != nil {
		return nil, nil, err
	}
	return p.AST(), p.buffer, nil
}

func astLookUp(node *node32, buffer string, rule string) (*node32, string, bool) {
	for node != nil {
		rs := rul3s[node.pegRule]
		if rs == rule {
			quote := string([]rune(buffer)[node.begin:node.end])
			return node, quote, true
		}
		if node.up != nil {
			return astLookUp(node.up, buffer, rule)
		}
		node = node.next
	}
	return nil, "", false
}

func checkAST(t *testing.T, expected string, ast *node32, buffer string) {
	nodes := strings.Split(expected, ";")
	for _, node := range nodes {
		valueBegin := strings.Index(node, "<")
		valueEnd := strings.LastIndex(node, ">")
		if valueBegin == -1 || valueEnd == -1 {
			assert.Fail(t, "invalid expected node %q", node, buffer)
		}
		name := node[:valueBegin]
		value := node[valueBegin+1 : valueEnd]
		if name == "" || value == "" {
			assert.Fail(t, "invalid expected node %q", node, buffer)
		}
		var (
			val string
			ok  bool
		)
		ast, val, ok = astLookUp(ast, buffer, name)
		assert.True(t, ok, buffer)
		assert.NotNil(t, node, buffer)
		if value != "*" {
			assert.Equal(t, value, val, buffer)
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
	} {
		ast, _, err := buildAST(t, test.src)
		if test.fail {
			assert.EqualError(t, err, test.expected, test.src)
		} else {
			checkAST(t, test.expected, ast, test.src)
		}
	}
}
