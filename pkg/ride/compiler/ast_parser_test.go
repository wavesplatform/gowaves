package compiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleAST(t *testing.T) {
	src := ""
	ast, buf, err := buildAST(t, src)
	require.NoError(t, err)
	astParser := NewASTParser(ast, buf)
	astParser.Parse()
	for _, err := range astParser.ErrorsList {
		fmt.Println(err.Error())
	}
}
