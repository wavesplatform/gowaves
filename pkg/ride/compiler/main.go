package main

//go:generate peg -output=compiler.peg.go compiler.peg

// Install https://github.com/pointlander/peg
// go install github.com/pointlander/peg@latest
// Using see peg -h

import (
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
	res.PrintSyntaxTree()
}
