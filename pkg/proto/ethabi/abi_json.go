package ethabi

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type argABI struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Components []argABI `json:"components,omitempty"`
}

type abi struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Constant        bool     `json:"constant"`
	Payable         bool     `json:"payable"`
	StateMutability string   `json:"stateMutability"`
	Inputs          []argABI `json:"inputs"`
	Outputs         []argABI `json:"outputs"`
}

func getArgumentABI(argType *Type) (argABI, error) {
	a := argABI{}
	if argType == nil {
		return a, nil
	}

	// this is the types that correspond with Ride
	switch argType.T {
	case TupleType:
		// this case is used only for payments
		a.Type = "tuple"
		a.Components = make([]argABI, 0, len(argType.TupleFields))
		for _, tupleElem := range argType.TupleFields {
			internalElem, err := getArgumentABI(&tupleElem.Type)
			if err != nil {
				return a, errors.Errorf("failed to parse slice type, %v", err)
			}
			internalElem.Name = tupleElem.Name
			a.Components = append(a.Components, internalElem)
		}
	case SliceType:
		internalElem, err := getArgumentABI(argType.Elem)
		if err != nil {
			return a, errors.Errorf("failed to parse slice type, %v", err)
		}
		a.Type = fmt.Sprintf("%s[]", internalElem.Type)
		a.Components = internalElem.Components
	case IntType:
		builder := intTextBuilder{
			size:     argType.Size,
			unsigned: false,
		}
		t, err := builder.MarshalText()
		if err != nil {
			return argABI{}, errors.Wrapf(err, "failed to create JSON argABI for type %q", argType.String())
		}
		a.Type = string(t)
	case UintType:
		builder := intTextBuilder{
			size:     argType.Size,
			unsigned: true,
		}
		t, err := builder.MarshalText()
		if err != nil {
			return argABI{}, errors.Wrapf(err, "failed to create JSON argABI for type %q", argType.String())
		}
		a.Type = string(t)
	case FixedBytesType:
		builder := fixedBytesTextBuilder{
			size: argType.Size,
		}
		t, err := builder.MarshalText()
		if err != nil {
			return argABI{}, errors.Wrapf(err, "failed to create JSON argABI for type %q", argType.String())
		}
		a.Type = string(t)
	case BoolType:
		a.Type = "bool"
	case AddressType:
		a.Type = "address"
	case BytesType:
		a.Type = "bytes"
	case StringType:
		a.Type = "string"
	default:
		return a, errors.Errorf("abi: unknown type %q", argType.T.String())
	}

	return a, nil
}

func makeJSONABIForMethod(method Method) (abi, error) {
	arguments := make([]argABI, 0)
	for _, arg := range method.Inputs {
		a, err := getArgumentABI(&arg.Type)
		if err != nil {
			return abi{}, errors.Errorf("failed to get json abi, %v", err)
		}
		a.Name = arg.Name

		arguments = append(arguments, a)
	}
	if method.Payments != nil {
		payment, err := getArgumentABI(&method.Payments.Type)
		if err != nil {
			return abi{}, errors.Errorf("failed to parse payments to json abi, %v", err)
		}
		payment.Name = method.Payments.Name
		arguments = append(arguments, payment)
	}

	methodABI := abi{
		Name:            method.RawName,
		Type:            "function",
		Constant:        false,
		Payable:         false,
		StateMutability: "nonpayable",
		Inputs:          arguments,
		Outputs:         make([]argABI, 0),
	}
	return methodABI, nil
}

func MakeJsonABI(methods []Method) ([]byte, error) {
	abiResult := make([]abi, 0, len(methods))

	for _, method := range methods {
		methodABI, err := makeJSONABIForMethod(method)
		if err != nil {
			return nil, err
		}
		abiResult = append(abiResult, methodABI)
	}

	return json.Marshal(abiResult)
}
