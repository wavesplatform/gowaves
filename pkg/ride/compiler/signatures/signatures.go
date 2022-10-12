package signatures

import (
	"encoding/json"
	"os"
)

type Type string

const (
	Undefined      Type = "Undefined"
	Any            Type = "T"
	BooleanType    Type = "Boolean"
	IntType        Type = "Integer"
	ListType       Type = "List"
	StringType     Type = "String"
	ByteVectorType Type = "ByteVector"
)

var Funcs = mustLoadFuncs()

type FunctionsSignatures struct {
	Funcs map[string]FunctionParams `json:"funcs"`
}

type FunctionParams struct {
	ID         string     `json:"id"`
	Arguments  [][]string `json:"arguments"`
	ReturnType string     `json:"return_type"`
}

func mustLoadFuncs() *FunctionsSignatures {
	f, err := os.Open("/Users/ailin/Projects/gowaves/pkg/ride/compiler/signatures/funcs.json")
	if err != nil {
		panic(err)
	}
	jsonParser := json.NewDecoder(f)
	s := &FunctionsSignatures{}
	if err = jsonParser.Decode(s); err != nil {
		panic(err)
	}
	return s
}
