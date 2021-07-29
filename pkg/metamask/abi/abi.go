package abi

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
)

func parseRide(data []byte) (*fourbyte.DecodedCallData, error) {
	db, err := fourbyte.NewDatabase()
	if err != nil {
		fmt.Println(err)
	}
	decodedData, err := db.ParseCallDataRide(data)
	return decodedData, err
}

func getJsonAbi(metaDApp map[fourbyte.Selector]fourbyte.Method) ([]byte, error) {
	// Define a tiny fake ABI struct for JSON marshalling
	type Arg struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Components []Arg  `json:"components,omitempty"`
	}
	type ABI struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Inputs []Arg  `json:"inputs"`
	}

	var abi []ABI

	for _, method := range metaDApp {
		arguments := make([]Arg, 0)
		for _, arg := range method.Inputs {
			a := Arg{Name: arg.Name, Type: arg.Type.String()}

			switch arg.Type.T {
			case fourbyte.TupleTy:
				a.Type = "tuple[]"
			case fourbyte.SliceTy:
				a.Type = fmt.Sprintf(arg.Type.String(), "[]")
			case fourbyte.StringTy: // variable arrays are written at the end of the return bytes
				a.Type = "string"
			case fourbyte.IntTy:
				a.Type = "int64"
			case fourbyte.UintTy:
				a.Type = "uint256"
			case fourbyte.BoolTy:
				a.Type = "bool"
			case fourbyte.AddressTy:
				a.Type = "address"
			case fourbyte.BytesTy:
				a.Type = "bytes"
			default:
				return nil, errors.Errorf("abi: unknown type %s", arg.Type.String())
			}

			arguments = append(arguments, a)
		}
		// payments
		arguments = append(arguments, Arg{Name: "", Type: "tuple[]", Components: []Arg{{Name: "", Type: "address"}, {Name: "", Type: "uint256"}}})
		m := ABI{Name: method.RawName, Type: "function", Inputs: arguments}
		abi = append(abi, m)
	}

	return json.Marshal(abi)
}
