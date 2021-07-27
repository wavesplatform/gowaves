package abi

import (
	"encoding/json"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
)

func parseNew(data []byte) (*fourbyte.DecodedCallData, error) {
	db, err := fourbyte.NewDatabase()
	if err != nil {
		fmt.Println(err)
	}
	decodedData, err := db.ParseCallDataNew(data)
	return decodedData, err
}

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
			arguments = append(arguments, a)
		}
		arguments = append(arguments, Arg{Name: "", Type: "tuple[]", Components: []Arg{{Name: "", Type: "address"}, {Name: "", Type: "uint256"}}})
		m := ABI{Name: method.RawName, Type: "function", Inputs: arguments}
		abi = append(abi, m)
	}

	return json.Marshal(abi)
}
