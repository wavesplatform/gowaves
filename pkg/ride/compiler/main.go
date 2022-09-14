package main

//go:generate peg -output=compiler.peg.go compiler.peg

import (
	"fmt"
	"log"
	"os"
)

func main() {
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	res := Result{Buffer: string(b)}
	res.Pretty = true
	err = res.Init()
	if err != nil {
		log.Fatal(err)
	}
	if err := res.Parse(); err != nil {
		log.Fatal(err)
	}
	ast := res.AST()
	fmt.Printf("%v/n", ast)
	fmt.Printf("%v/n", translatePositions(res.buffer, []int{int(ast.up.token32.end)})[int(ast.up.token32.end)])
	res.PrintSyntaxTree()
}
