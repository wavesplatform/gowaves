package main

//go:generate pigeon -o=compiler.go compiler.peg

import (
	"fmt"
	"log"
	"os"
)

func main() {
	in := os.Stdin
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		in = f
	}
	pn, err := ParseReader("", in)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v", pn)
}

// ProgramNode is a root node
type ProgramNode struct {
	declaration []Declaration
	headers     Directives
}

func newProgramNode(declaration []Declaration, headers Directives) (ProgramNode, error) {
	return ProgramNode{declaration, headers}, nil
}
