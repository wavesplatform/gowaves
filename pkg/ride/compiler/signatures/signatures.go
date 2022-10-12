package signatures

import (
	"embed"
	"encoding/json"
)

//go:embed funcs.json
var embedded embed.FS

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
	f, err := embedded.ReadFile("funcs.json")
	if err != nil {
		panic(err)
	}
	s := &FunctionsSignatures{}
	if err = json.Unmarshal(f, s); err != nil {
		panic(err)
	}
	return s
}
