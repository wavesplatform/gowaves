package ethabi

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
	Payments *Argument
	Sig      Signature
}

func NewMethodFromRideFunctionMeta(rideF meta.Function, addPayments bool) (Method, error) {
	args := make(Arguments, 0, len(rideF.Arguments))
	for i, rideT := range rideF.Arguments {
		// name is empty because we don't have any info in metadata about argument name
		t, err := NewArgumentFromRideTypeMeta("", rideT)
		if err != nil {
			return Method{}, errors.Wrapf(err,
				"failed to build ABI method %q (argument %d) from ride function metadata", rideF.Name, i,
			)
		}
		args = append(args, t)
	}
	sig, err := NewSignatureFromRideFunctionMeta(rideF, addPayments)
	if err != nil {
		return Method{}, errors.Wrapf(err,
			"failed to build function signature for ABI method with name %s", rideF.Name,
		)
	}
	var payments *Argument
	if addPayments {
		payments = &paymentsArgument
	}

	meth := Method{
		RawName:  rideF.Name,
		Inputs:   args,
		Payments: payments,
		Sig:      sig,
	}
	return meth, nil
}

func (m *Method) String() string {
	return m.Sig.String()
}
