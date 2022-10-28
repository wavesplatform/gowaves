package compiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleAST(t *testing.T) {
	//t.SkipNow()
	src := `{-# STDLIB_VERSION 6 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}

let block = {
	let (a, b, c) = (1, 2, 3)
	true
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
