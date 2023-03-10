package compiler

import (
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

//go:generate peg -output=parser.peg.go ride.peg

func CompileToTree(code string) (*ast.Tree, []error) {
	p := Parser{Buffer: code}
	err := p.Init()
	if err != nil {
		return nil, []error{err}
	}
	err = p.Parse()
	if err != nil {
		return nil, []error{err}
	}
	astParser := NewASTParser(p.AST(), p.buffer)
	astParser.Parse()
	astParser.Tree.Meta.Version = 2
	if len(astParser.ErrorsList) > 0 {
		return nil, astParser.ErrorsList
	}
	return astParser.Tree, nil
}

func Compile(code string) ([]byte, []error) {
	tree, errs := CompileToTree(code)
	if len(errs) > 0 {
		return nil, errs
	}
	res, err := serialization.SerializeTreeV2(tree)
	if err != nil {
		return nil, []error{err}
	}
	return res, nil
}
