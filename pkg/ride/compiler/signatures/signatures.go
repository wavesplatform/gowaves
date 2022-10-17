package signatures

import (
	"embed"
	"encoding/json"
	"reflect"
	"strings"
)

//go:embed funcs.json
var embedded embed.FS

var Funcs = mustLoadFuncs()

type FunctionsSignaturesJson struct {
	Funcs map[string]FunctionParamsJson `json:"funcs"`
}

type FunctionParamsJson struct {
	ID         string   `json:"id"`
	Arguments  []string `json:"arguments"`
	ReturnType string   `json:"return_type"`
}

type FunctionsSignatures struct {
	Funcs map[string]FunctionParams
}

type FunctionParams struct {
	ID         string
	Arguments  []Type
	ReturnType Type
}

func mustLoadFuncs() *FunctionsSignatures {
	f, err := embedded.ReadFile("funcs.json")
	if err != nil {
		panic(err)
	}
	s := &FunctionsSignaturesJson{}
	if err = json.Unmarshal(f, s); err != nil {
		panic(err)
	}
	res := &FunctionsSignatures{
		Funcs: map[string]FunctionParams{},
	}
	for k, v := range s.Funcs {
		var args []Type
		for _, a := range v.Arguments {
			args = append(args, ParseType(a))
		}
		res.Funcs[k] = FunctionParams{
			ID:         v.ID,
			Arguments:  args,
			ReturnType: ParseType(v.ReturnType),
		}
	}
	return res
}

func ParseType(t string) Type {
	// TODO(anton): check Type was declared
	types := strings.ReplaceAll(t, " ", "")
	if strings.HasPrefix(types, "List[") {
		return parseList(types)
	}
	if strings.Contains(types, "|") {
		return parseUnion(types)
	}
	return SimpleType{types}
}

func parseList(t string) Type {
	listType := strings.TrimPrefix(t, "List[")
	listType = strings.TrimSuffix(listType, "]")
	return ListType{ParseType(listType)}
}

func parseUnion(t string) Type {
	unionTypes := strings.Split(t, "|")
	res := map[string]Type{}
	for _, i := range unionTypes {
		T := ParseType(i)
		res[T.String()] = T
	}
	return UnionType{Types: res}
}

var (
	Undefined      = SimpleType{"Undefined"}
	Any            = SimpleType{"T"}
	BooleanType    = SimpleType{"Boolean"}
	IntType        = SimpleType{"Int"}
	StringType     = SimpleType{"String"}
	ByteVectorType = SimpleType{"ByteVector"}

	ListOfAny = ListType{Any}
)

type Type interface {
	Comp(Type) bool
	String() string
}

type SimpleType struct {
	Type string
}

func (t SimpleType) Comp(rideType Type) bool {
	if t.Type == "T" {
		return true
	}
	T, ok := rideType.(SimpleType)
	if !ok {
		return false
	}
	return t.Type == T.Type
}

func (t SimpleType) String() string {
	return t.Type
}

type UnionType struct {
	Types map[string]Type
}

func (t UnionType) Comp(rideType Type) bool {
	if T, ok := rideType.(UnionType); ok {
		return reflect.DeepEqual(t, T)
	}
	_, ok := t.Types[rideType.String()]
	return ok
}

func (t *UnionType) AppendType(rideType Type) {
	if T, ok := rideType.(UnionType); ok {
		for k, v := range T.Types {
			t.Types[k] = v
		}
		return
	}
	t.Types[rideType.String()] = rideType
}

func (t UnionType) String() string {
	var res string
	cnt := 0
	for k := range t.Types {
		res += k
		cnt++
		if cnt < len(t.Types) {
			res += "|"
		}
	}
	return res
}

type ListType struct {
	Type Type
}

func (t ListType) Comp(rideType Type) bool {
	if T, ok := rideType.(ListType); ok {
		return t.Type.Comp(T.Type)
	}
	return false
}

func (t ListType) String() string {
	return "List[" + t.Type.String() + "]"
}

func (t *ListType) AppendType(rideType Type) {
	resType := UnionType{Types: map[string]Type{}}
	if T, ok := rideType.(UnionType); ok {
		resType.AppendType(T)
	}
	resType.AppendType(t.Type)
	if T, ok := rideType.(UnionType); ok {
		resType.AppendType(T)
		return
	}
	resType.AppendType(rideType)
}

func (t *ListType) AppendList(rideType Type) {
	T := rideType.(ListType)
	resType := UnionType{Types: map[string]Type{}}
	resType.AppendType(t.Type)
	resType.AppendType(T.Type)
}
