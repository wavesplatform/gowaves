package stdlib

import (
	"encoding/json"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var structJsonPath = "../generate/ride_objects.json"

type actionField struct {
	Name             string   `json:"name"`
	Types            []string `json:"types"`
	ConstructorOrder int      `json:"constructorOrder"` // order for constructor
}

type actionsObject struct {
	LibVersion ast.LibraryVersion  `json:"version"`
	Deleted    *ast.LibraryVersion `json:"deleted,omitempty"`
	Fields     []actionField       `json:"fields"`
}

type rideObject struct {
	Name    string          `json:"name"`
	Actions []actionsObject `json:"actions"`
}

type rideObjects struct {
	Objects []rideObject `json:"objects"`
}

type ObjectFields struct {
	Name string
	Type Type
}

type ObjectsSignatures struct {
	Obj map[string][]ObjectFields
}

func (s *ObjectsSignatures) GetConstruct(name string, args []Type) (FunctionParams, bool) {
	fields, ok := s.Obj[name]
	if !ok {
		return FunctionParams{}, ok
	}
	if len(args) != len(fields) {
		return FunctionParams{}, ok
	}
	var resTypes []Type
	for i := range fields {
		if !fields[i].Type.Comp(args[i]) {
			return FunctionParams{}, ok
		}
		resTypes = append(resTypes, fields[i].Type)
	}
	return FunctionParams{
		ID:         ast.UserFunction(name),
		Arguments:  resTypes,
		ReturnType: SimpleType{Type: name},
	}, true
}

func (s *ObjectsSignatures) GetField(objType Type, fieldName string) (Type, bool) {
	t, ok := objType.(SimpleType)
	if !ok {
		return nil, false
	}

	fields, ok := s.Obj[t.Type]
	if !ok {
		return nil, false
	}
	var resType Type
	for _, f := range fields {
		if fieldName == f.Name {
			resType = f.Type
			break
		}
	}
	return resType, resType != nil
}

func (s *ObjectsSignatures) IsExist(name string) bool {
	_, ok := s.Obj[name]
	return ok
}

var (
	ObjectsByVersion = mustLoadObjects()
)

func parseObjectFieldsTypes(rawTypes []string) Type {
	var types []string
	for _, rawT := range rawTypes {
		t := strings.ReplaceAll(rawT, "ride", "")
		t = strings.ReplaceAll(t, "Bytes", "ByteVector")
		types = append(types, t)
	}

	resRawType := strings.Join(types, "|")

	return ParseType(resRawType)
}

func mustLoadObjects() map[ast.LibraryVersion]ObjectsSignatures {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	filePath := filepath.Clean(filepath.Join(pwd, structJsonPath))
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	jsonParser := json.NewDecoder(f)
	s := &rideObjects{}
	if err = jsonParser.Decode(s); err != nil {
		panic(err)
	}

	res := map[ast.LibraryVersion]ObjectsSignatures{
		ast.LibV1: {
			map[string][]ObjectFields{},
		},
		ast.LibV2: {
			map[string][]ObjectFields{},
		},
		ast.LibV3: {
			map[string][]ObjectFields{},
		},
		ast.LibV4: {
			map[string][]ObjectFields{},
		},
		ast.LibV5: {
			map[string][]ObjectFields{},
		},
		ast.LibV6: {
			map[string][]ObjectFields{},
		},
	}
	for _, obj := range s.Objects {
		sort.SliceStable(obj.Actions, func(i, j int) bool {
			return int(obj.Actions[i].LibVersion) < int(obj.Actions[j].LibVersion)
		})
		for _, ver := range obj.Actions {
			var resFields []ObjectFields
			sort.SliceStable(ver.Fields, func(i, j int) bool {
				return ver.Fields[i].ConstructorOrder < ver.Fields[j].ConstructorOrder
			})
			for _, f := range ver.Fields {
				resFields = append(resFields, ObjectFields{
					Name: f.Name,
					Type: parseObjectFieldsTypes(f.Types),
				})
			}
			maxVer := int(ast.CurrentMaxLibraryVersion())
			if ver.Deleted != nil {
				maxVer = int(*ver.Deleted)
			}
			for i := int(ver.LibVersion); i <= maxVer; i++ {
				res[ast.LibraryVersion(byte(i))].Obj[obj.Name] = resFields
			}
		}
	}

	return res
}
