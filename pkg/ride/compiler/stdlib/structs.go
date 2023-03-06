package stdlib

import (
	"encoding/json"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var structJsonPath = "../../generate/ride_objects.json"

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

type ObjectField struct {
	Name string
	Type Type
}

type ObjectInfo struct {
	NotConstruct bool
	Fields       []ObjectField
}

type ObjectsSignatures struct {
	Obj map[string]ObjectInfo
}

func (s *ObjectsSignatures) GetConstruct(name string, args []Type) (FunctionParams, bool) {
	info, ok := s.Obj[name]
	if !ok {
		return FunctionParams{}, ok
	}
	if info.NotConstruct {
		return FunctionParams{}, false
	}
	if len(args) != len(info.Fields) {
		return FunctionParams{}, false
	}
	var resTypes []Type
	for i := range info.Fields {
		if !info.Fields[i].Type.EqualWithEntry(args[i]) {
			return FunctionParams{}, false
		}
		resTypes = append(resTypes, info.Fields[i].Type)
	}
	return FunctionParams{
		ID:         ast.UserFunction(name),
		Arguments:  resTypes,
		ReturnType: SimpleType{Type: name},
	}, true
}

func (s *ObjectsSignatures) GetField(objType Type, fieldName string) (Type, bool) {
	if u, okU := objType.(UnionType); okU {
		resType := UnionType{Types: []Type{}}
		for _, t := range u.Types {
			fieldType, ok := s.getFieldForSimpleType(t, fieldName)
			if !ok {
				return nil, false
			}
			resType.AppendType(fieldType)
		}
		if len(resType.Types) == 1 {
			return resType.Types[0], true
		} else {
			return resType, true
		}
	}

	return s.getFieldForSimpleType(objType, fieldName)
}

func (s *ObjectsSignatures) getFieldForSimpleType(objType Type, fieldName string) (Type, bool) {
	t, ok := objType.(SimpleType)
	if !ok {
		return nil, false
	}

	info, ok := s.Obj[t.Type]
	if !ok {
		return nil, false
	}
	var resType Type
	for _, f := range info.Fields {
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
		if rawT == "rideAddressLike" {
			continue
		}
		t := strings.ReplaceAll(rawT, "ride", "")
		t = strings.ReplaceAll(t, "Bytes", "ByteVector")
		types = append(types, t)
	}

	resRawType := strings.Join(types, "|")

	return ParseType(resRawType)
}

func appendRemainingStructs(s *rideObjects) {
	remainingObjects := []rideObject{
		{
			Name: "Address",
			Actions: []actionsObject{
				{
					LibVersion: ast.LibV1,
					Deleted:    nil,
					Fields: []actionField{
						{
							Name:  "bytes",
							Types: []string{"rideBytes"},
						},
					},
				},
			},
		},
		{
			Name: "Alias",
			Actions: []actionsObject{
				{
					LibVersion: ast.LibV1,
					Deleted:    nil,
					Fields: []actionField{
						{
							Name:  "alias",
							Types: []string{"rideString"},
						},
					},
				},
			},
		},
	}
	s.Objects = append(s.Objects, remainingObjects...)
}

func changeName(name string) string {
	if name == "assetID" {
		return "assetId"
	}
	if name == "transactionID" {
		return "transactionId"
	}
	if name == "feeAssetID" {
		return "feeAssetId"
	}
	return name
}

func changeRideTypeFields(name string, fields []actionField) []actionField {
	switch name {
	case "Order":
		for i := range fields {
			switch fields[i].Name {
			case "assetPair":
				fields[i].Types = []string{"AssetPair"}
			case "orderType":
				fields[i].Types = []string{"Buy", "Sell"}
			}
		}
	case "ExchangeTransaction":
		for i := range fields {
			switch fields[i].Name {
			case "sellOrder", "buyOrder":
				fields[i].Types = []string{"Order"}
			}
		}
	}
	return fields
}

func mustLoadObjects() map[ast.LibraryVersion]ObjectsSignatures {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("No current file info")
	}
	filePath := filepath.Clean(filepath.Join(filepath.Dir(filename), structJsonPath))
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
	appendRemainingStructs(s)
	res := map[ast.LibraryVersion]ObjectsSignatures{
		ast.LibV1: {map[string]ObjectInfo{}},
		ast.LibV2: {map[string]ObjectInfo{}},
		ast.LibV3: {map[string]ObjectInfo{}},
		ast.LibV4: {map[string]ObjectInfo{}},
		ast.LibV5: {map[string]ObjectInfo{}},
		ast.LibV6: {map[string]ObjectInfo{}},
	}
	for _, obj := range s.Objects {
		sort.SliceStable(obj.Actions, func(i, j int) bool {
			return int(obj.Actions[i].LibVersion) < int(obj.Actions[j].LibVersion)
		})
		for _, ver := range obj.Actions {
			var resInfo ObjectInfo
			sort.SliceStable(ver.Fields, func(i, j int) bool {
				return ver.Fields[i].ConstructorOrder < ver.Fields[j].ConstructorOrder
			})
			ver.Fields = changeRideTypeFields(obj.Name, ver.Fields)
			for _, f := range ver.Fields {
				name := changeName(f.Name)
				resInfo.Fields = append(resInfo.Fields, ObjectField{
					Name: name,
					Type: parseObjectFieldsTypes(f.Types),
				})
			}
			if strings.HasSuffix(obj.Name, "Transaction") {
				resInfo.NotConstruct = true
			}
			maxVer := int(ast.CurrentMaxLibraryVersion())
			if ver.Deleted != nil {
				maxVer = int(*ver.Deleted)
			}
			for i := int(ver.LibVersion); i <= maxVer; i++ {
				res[ast.LibraryVersion(byte(i))].Obj[obj.Name] = resInfo
			}
		}
	}

	return res
}
