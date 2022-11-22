package compiler

import (
	"encoding/base64"

	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

//go:generate peg -output=parser.peg.go ride.peg

func Compile(code string) (string, []error) {
	p := Parser{Buffer: code}
	err := p.Init()
	if err != nil {
		return "", []error{err}
	}
	err = p.Parse()
	if err != nil {
		return "", []error{err}
	}
	astParser := NewASTParser(p.AST(), p.buffer)
	astParser.Parse()
	astParser.Tree.Meta.Version = 2
	if len(astParser.ErrorsList) > 0 {
		return "", astParser.ErrorsList
	}
	res, err := serialization.SerializeTreeV2(astParser.Tree)
	if err != nil {
		return "", []error{err}
	}
	return base64.StdEncoding.EncodeToString(res), nil
}
