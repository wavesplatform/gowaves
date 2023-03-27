package stdlib

import (
	"encoding/json"
	"strconv"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

var funcsByVersion = mustLoadFuncs()

func FuncsByVersion() map[ast.LibraryVersion]FunctionsSignatures {
	return funcsByVersion
}

type FunctionsSignatures struct {
	Funcs map[string][]FunctionParams
}

type FunctionParams struct {
	ID         ast.Function
	Arguments  []Type
	ReturnType Type
}

func (sig *FunctionsSignatures) Get(name string, args []Type) (FunctionParams, bool) {
	overloaded, ok := sig.Funcs[name]
	if !ok {
		return FunctionParams{}, ok
	}
	var res FunctionParams
	for _, o := range overloaded {
		if len(o.Arguments) != len(args) {
			continue
		}
		isMatched := true
		for i := range o.Arguments {
			if !o.Arguments[i].EqualWithEntry(args[i]) {
				isMatched = false
				break
			}
		}
		if isMatched {
			res = o
			break
		}
	}
	if res.ID != nil {
		return getGenericFuncsSign(name, args, res), true
	}

	return FunctionParams{}, false
}

func (sig *FunctionsSignatures) Check(name string) bool {
	_, ok := sig.Funcs[name]
	return ok
}

type FunctionsSignaturesJson struct {
	Versions []FunctionsInVersions `json:"versions"`
}

type FunctionsInVersions struct {
	New    map[string][]FunctionParamsJson `json:"new"`
	Remove []string                        `json:"remove"`
}

type FunctionParamsJson struct {
	ID         string   `json:"id"`
	Arguments  []string `json:"arguments"`
	ReturnType string   `json:"return_type"`
}

func mustLoadFuncs() map[ast.LibraryVersion]FunctionsSignatures {
	f, err := embedFS.ReadFile("funcs.json")
	if err != nil {
		panic(err)
	}
	s := &FunctionsSignaturesJson{}
	if err = json.Unmarshal(f, s); err != nil {
		panic(err)
	}
	res := map[ast.LibraryVersion]FunctionsSignatures{}
	for v, funcs := range s.Versions {
		funcsInVersion := FunctionsSignatures{
			Funcs: map[string][]FunctionParams{},
		}
		if v > 0 {
			// copy prev version
			for name, over := range res[ast.LibraryVersion(byte(v))].Funcs {
				funcsInVersion.Funcs[name] = over
			}
		}
		for _, name := range funcs.Remove {
			delete(funcsInVersion.Funcs, name)
		}
		for name, over := range funcs.New {
			var funcsParams []FunctionParams
			for _, o := range over {
				var args []Type
				for _, a := range o.Arguments {
					args = append(args, ParseType(a))
				}
				var funcId ast.Function
				if _, err := strconv.ParseInt(o.ID, 10, 64); err != nil {
					funcId = ast.UserFunction(o.ID)
				} else {
					funcId = ast.NativeFunction(o.ID)
				}
				funcsParams = append(funcsParams, FunctionParams{
					ID:         funcId,
					Arguments:  args,
					ReturnType: ParseType(o.ReturnType),
				})
			}
			funcsInVersion.Funcs[name] = funcsParams
		}
		res[ast.LibraryVersion(byte(v+1))] = funcsInVersion
	}
	return res
}

// handleTemplateFuncs

func getGenericFuncsSign(name string, args []Type, findFuncPar FunctionParams) FunctionParams {
	switch name {
	case "extract", "value", "valueOrErrorMessage":
		var resType Type
		if u, ok := args[0].(UnionType); ok {
			uResType := UnionType{Types: []Type{}}
			for _, uT := range u.Types {
				if !uT.Equal(SimpleType{Type: "Unit"}) {
					uResType.AppendType(uT)
				}
			}
			if len(uResType.Types) != 1 {
				resType = uResType
			} else {
				resType = uResType.Types[0]
			}
		} else {
			resType = args[0]
		}
		findFuncPar.ReturnType = resType
	case "getElement":
		l := args[0].(ListType)
		findFuncPar.ReturnType = l.Type
	case "removeByIndex":
		findFuncPar.ReturnType = args[0]
	case "cons":
		l := args[1].(ListType)
		l.AppendType(args[0])
		findFuncPar.ReturnType = l
	case "valueOrElse":
		findFuncPar.ReturnType = args[1]
	}
	return findFuncPar
}
