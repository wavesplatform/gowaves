package abi

import (
	"encoding/json"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/metamask/abi/fourbyte"
	"regexp"
	"strings"
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

var selectorRegexp = regexp.MustCompile(`^([^\)]+)\(([A-Za-z0-9,\[\]]*)\)`)

func getJsonAbi(selector string, payments []fourbyte.Payment) ([]byte, error) {
	// Define a tiny fake ABI struct for JSON marshalling
	type Arg struct {
		Type string `json:"type"`
	}
	type ABI struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Inputs []Arg  `json:"inputs"`
	}
	// Validate the unescapedSelector and extract it's components
	groups := selectorRegexp.FindStringSubmatch(selector)
	if len(groups) != 3 {
		return nil, fmt.Errorf("invalid selector %q (%v matches)", selector, len(groups))
	}
	name := groups[1]
	args := groups[2]

	// Reassemble the fake ABI and constuct the JSON
	arguments := make([]Arg, 0)
	if len(args) > 0 {
		for _, arg := range strings.Split(args, ",") {
			arguments = append(arguments, Arg{arg})
		}
	}

	if payments != nil {
		// it means that payments are attached
		arg := "(address, uint256)[]" // payments
		arguments = append(arguments, Arg{arg})
	}
	return json.Marshal([]ABI{{name, "function", arguments}})
}
