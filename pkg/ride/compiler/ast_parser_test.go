package compiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleAST(t *testing.T) {
	//t.SkipNow()
	src := `# asdasdasd
{-# STDLIB_VERSION 6 #-} #  asdasdasd
{-# CONTENT_TYPE EXPRESSION #-}
{-# SCRIPT_TYPE ASSET #-}

let kmsdc = sigVerify(base58'FYCT9GxhR2igEeyf9SWGi85bebBVaTAf9WUihEQnnBa9', base58'FYCT9GxhR2igEeyf9SWGi85bebBVaTAf9WUihEQnnBa9', base58'FYCT9GxhR2igEeyf9SWGi85bebBVaTAf9WUihEQnnBa9')

#let ad = if true then 3 else false

#let asd = FOLD<10>(10, 10, asd)
#let asd = 5

#let asdasd = -10

#let jnsad = test[123]

#let GetFuncCall = sad.asdasd()

#let FuncCall = asd()

#let b = "asd"

#let f = base58'asdasdasdasdasd'

#let c = true

#let a = -(1 + 12312)

let asd = [1  + 123, 124, 3284]

let sdf = someFuncCall(10, 100, 1000)

let asd = asd.asdasd

let jfj = {
    let asdasd = 123123
    -1000
    1000
}

func someFunc(someArg1: Int, someArd2: String) = {
    let k = 1000
    true
}

@Verifier(tx)
func verify() = {
    sigVerify(tx.bodyBytes, tx.proofs[0], base58'FYCT9GxhR2igEeyf9SWGi85bebBVaTAf9WUihEQnnBa9')
    #true
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
