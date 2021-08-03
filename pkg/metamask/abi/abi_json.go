package abi

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
)

type argABI struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Components []argABI `json:"components,omitempty"`
}

type abi struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Inputs []argABI `json:"inputs"`
}

func getArgumentABI(argType *fourbyte.Type) (argABI, error) {
	a := argABI{}
	if argType == nil {
		return a, nil
	}

	// this is the types that correspond with Ride
	switch argType.T {
	case fourbyte.TupleTy:
		a.Type = "tuple"
		for i, tupleElem := range argType.TupleElems {
			internalElem, err := getArgumentABI(&tupleElem)
			if err != nil {
				return a, errors.Errorf("failed to parse slice type, %v", err)
			}
			internalElem.Name = argType.TupleRawNames[i]
			a.Components = append(a.Components, internalElem)
		}

	case fourbyte.SliceTy:
		internalElem, err := getArgumentABI(argType.Elem)
		if err != nil {
			return a, errors.Errorf("failed to parse slice type, %v", err)
		}
		a.Type = fmt.Sprintf("%s[]", internalElem.Type)
		a.Components = internalElem.Components

	case fourbyte.StringTy: // variable arrays are written at the end of the return bytes
		a.Type = "string"
	case fourbyte.IntTy:
		a.Type = "int64"
	case fourbyte.UintTy:
		a.Type = "uint8"
	case fourbyte.BoolTy:
		a.Type = "bool"
	case fourbyte.AddressTy:
		a.Type = "bytes"
	case fourbyte.BytesTy:
		a.Type = "bytes"
	default:
		return a, errors.Errorf("abi: unknown type %s", a.Type)
	}

	return a, nil
}

func getJsonAbi(metaDApp []fourbyte.Method) ([]byte, error) {
	var abiResult []abi

	for _, method := range metaDApp {
		arguments := make([]argABI, 0)
		for _, arg := range method.Inputs {
			a, err := getArgumentABI(&arg.Type)
			if err != nil {
				return nil, errors.Errorf("failed to get json abi, %v", err)
			}
			a.Name = arg.Name

			arguments = append(arguments, a)
		}
		if method.Payments != nil {
			payment, err := getArgumentABI(&method.Payments.Type)
			if err != nil {
				return nil, errors.Errorf("failed to parse payments to json abi, %v", err)
			}
			payment.Name = method.Payments.Name
			arguments = append(arguments, payment)
		}

		m := abi{Name: method.RawName, Type: "function", Inputs: arguments}
		abiResult = append(abiResult, m)
	}

	return json.Marshal(abiResult)
}
