package compiler

import (
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

//go:generate peg -output=parser.peg.go ride.peg

func CompileToTree(code string) (*ast.Tree, []error) {
	pp := Parser{Buffer: code}
	err := pp.Init()
	if err != nil {
		return nil, []error{err}
	}
	err = pp.Parse()
	if err != nil {
		return nil, []error{err}
	}
	ap := newASTParser(pp.AST(), pp.buffer)
	ap.parse()
	if len(ap.errorsList) > 0 {
		return nil, ap.errorsList
	}
	return ap.tree, nil
}

func Compile(code string, compact, removeUnused bool) ([]byte, []error) {
	tree, errs := CompileToTree(code)
	if len(errs) > 0 {
		return nil, errs
	}
	if removeUnused && tree.IsDApp() {
		removeUnusedCode(tree)
	}
	if compact && tree.IsDApp() {
		comp := NewCompaction(tree)
		comp.Compact()
	}
	res, err := serialization.SerializeTree(tree)
	if err != nil {
		return nil, []error{err}
	}
	return res, nil
}
