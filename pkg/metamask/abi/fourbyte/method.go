package fourbyte

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

type Method struct {
	RawName string // RawName is the raw method name parsed from ABI
	Inputs  Arguments
	// Sig returns the methods string signature according to the ABI spec.
	// e.g.		function foo(uint32 a, int b) = "foo(uint32,int256)"
	// Please note that "int" is substitute for its canonical representation "int256"
	Sig Signature
}

// NewMethod creates a new Method.
// A method should always be created using NewMethod.
// It also precomputes the sig representation and the string representation
// of the method.
func NewMethod(rawName string, inputs Arguments) Method {
	return Method{
		RawName: rawName,
		Inputs:  inputs,
		Sig:     NewSignature(rawName, inputs),
	}
}

func NewMethodFromRideFunctionMeta(rideF meta.Function) (Method, error) {
	args := make(Arguments, 0, len(rideF.Arguments))
	for _, rideT := range rideF.Arguments {
		// nickeskov: empty because we don't have any info in metadata about argument name
		t, err := NewArgumentFromRideTypeMeta("", rideT)
		if err != nil {
			return Method{}, errors.Wrapf(err,
				"failed to build ABI method with name %q from ride function metadata", rideF.Name,
			)
		}
		args = append(args, t)
	}
	return NewMethod(rideF.Name, args), nil
}

func (m *Method) String() string {
	return m.Sig.String()
}

func (m *Method) IsERC20() bool {
	_, isERC20 := erc20Methods[m.Sig.Selector()]
	return isERC20
}
