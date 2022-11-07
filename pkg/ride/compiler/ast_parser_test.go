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
