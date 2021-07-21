package fourbyte

import (
	"fmt"
	"strings"
)

type FunctionType byte

const (
	Callable FunctionType = iota
	Verifier
)

type Method struct {
	Type FunctionType

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
func NewMethod(rawName string, funType FunctionType, inputs, outputs Arguments) Method {
	var (
		inputNames  = make([]string, len(inputs))
		outputNames = make([]string, len(outputs))
	)
	for i, input := range inputs {
		inputNames[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	for i, output := range outputs {
		outputNames[i] = output.Type.String()
		if len(output.Name) > 0 {
			outputNames[i] += fmt.Sprintf(" %v", output.Name)
		}
	}
	// calculate the signature and method id. Note only function
	// has meaningful signature and id.
	var (
		sig Signature
	)
	if funType == Callable {
		sig = NewSignature(rawName, inputs)
	}

	identity := fmt.Sprintf("function %v", rawName)
	if funType == Verifier {
		identity = "verifier"
	}

	str := fmt.Sprintf("%v(%v) returns(%v)", identity, strings.Join(inputNames, ", "), strings.Join(outputNames, ", "))

	return Method{
		RawName: rawName,
		Type:    funType,

		Inputs: inputs,

		str: str,

		Sig: sig,
	}
}

func (m *Method) String() string {
	return m.str
}

func (m *Method) IsERC20() bool {
	_, isERC20 := erc20Methods[m.Sig.Selector()]
	return isERC20
}
