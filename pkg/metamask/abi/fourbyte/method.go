package fourbyte

import (
	"fmt"
	"strings"
)

type Method struct {
	RawName string // RawName is the raw method name parsed from ABI
	Inputs  Arguments

	str string
	// Sig returns the methods string signature according to the ABI spec.
	// e.g.		function foo(uint32 a, int b) = "foo(uint32,int256)"
	// Please note that "int" is substitute for its canonical representation "int256"
	Sig Signature
}

// NewMethod creates a new Method.
// A method should always be created using NewMethod.
// It also precomputes the sig representation and the string representation
// of the method.
// TODO(nickeskov): remove outputs
func NewMethod(rawName string, inputs Arguments) Method {
	inputNames := make([]string, 0, len(inputs))
	for _, input := range inputs {
		inputNames = append(inputNames, fmt.Sprintf("%v %v", input.Type, input.Name))
	}
	// calculate the signature and method id. Note only function
	// has meaningful signature and id.

	return Method{
		RawName: rawName,
		Inputs:  inputs,
		str:     fmt.Sprintf("function %v(%v)", rawName, strings.Join(inputNames, ", ")),
		Sig:     NewSignature(rawName, inputs),
	}
}

func (m *Method) String() string {
	return m.str
}

func (m *Method) IsERC20() bool {
	_, isERC20 := erc20Methods[m.Sig.Selector()]
	return isERC20
}
