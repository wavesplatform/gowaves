package stdlib

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

var defaultTypes = mustLoadDefaultTypes()

func DefaultTypes() map[ast.LibraryVersion]map[string]Type {
	return defaultTypes
}

type Type interface {
	String() string
	Equal(Type) bool
	EqualWithEntry(Type) bool
}

const MaxTupleLength = 22

var (
	Any            = SimpleType{"Any"}
	BooleanType    = SimpleType{"Boolean"}
	IntType        = SimpleType{"Int"}
	StringType     = SimpleType{"String"}
	ByteVectorType = SimpleType{"ByteVector"}
	BigIntType     = SimpleType{"BigInt"}
	ThrowType      = SimpleType{"Unknown"}

	CallableRetV3 = UnionType{Types: []Type{
		SimpleType{Type: "ScriptResult"},
		SimpleType{Type: "TransferSet"},
		SimpleType{Type: "WriteSet"},
	}}
	CallableRetV4 = ListType{Type: UnionType{Types: []Type{
		SimpleType{Type: "BinaryEntry"},
		SimpleType{Type: "BooleanEntry"},
		SimpleType{Type: "Burn"},
		SimpleType{Type: "DeleteEntry"},
		SimpleType{Type: "IntegerEntry"},
		SimpleType{Type: "Issue"},
		SimpleType{Type: "Reissue"},
		SimpleType{Type: "ScriptTransfer"},
		SimpleType{Type: "SponsorFee"},
		SimpleType{Type: "StringEntry"},
	}}}
	callableRetV5OnlyList = ListType{Type: UnionType{Types: []Type{
		SimpleType{Type: "BinaryEntry"},
		SimpleType{Type: "BooleanEntry"},
		SimpleType{Type: "Burn"},
		SimpleType{Type: "DeleteEntry"},
		SimpleType{Type: "IntegerEntry"},
		SimpleType{Type: "Issue"},
		SimpleType{Type: "Lease"},
		SimpleType{Type: "LeaseCancel"},
		SimpleType{Type: "Reissue"},
		SimpleType{Type: "ScriptTransfer"},
		SimpleType{Type: "SponsorFee"},
		SimpleType{Type: "StringEntry"},
	}}}
	CallableRetV5 = UnionType{Types: []Type{
		TupleType{Types: []Type{
			callableRetV5OnlyList,
			Any,
		}},
		callableRetV5OnlyList,
	}}
)

func ParseType(t string) Type {
	p := Types{Buffer: t}
	err := p.Init()
	if err != nil {
		panic(err)
	}
	err = p.Parse()
	if err != nil {
		panic(err)
	}
	node := p.AST()
	return handleTypes(node.up, t)
}

func handleTypes(node *node32, t string) Type {
	curNode := node.up
	var T Type
	switch curNode.pegRule {
	case ruleGenericType:
		T = handleGeneric(curNode, t)
	case ruleTupleType:
		T = handleTupleType(curNode, t)
	case ruleType:
		// check Types
		T = SimpleType{t[curNode.begin:curNode.end]}
	}
	curNode = curNode.next
	if curNode == nil {
		return T
	}

	resType := UnionType{Types: []Type{}}
	resType.Types = append(resType.Types, T)
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	resType.Types = append(resType.Types, handleTypes(curNode, t))
	return resType
}

func handleTupleType(node *node32, t string) Type {
	curNode := node.up
	var tupleTypes []Type
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == ruleTypes {
			tupleTypes = append(tupleTypes, handleTypes(curNode, t))
			curNode = curNode.next
		}
		if curNode == nil {
			break
		}
	}
	return TupleType{tupleTypes}
}

func handleGeneric(node *node32, t string) Type {
	curNode := node.up
	name := t[curNode.begin:curNode.end]
	if name != "List" {
		panic(errors.Errorf("Generig type should be List, but \"%s\"", name))
	}
	curNode = curNode.next
	if curNode == nil {
		return ListType{nil}
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}

	return ListType{Type: handleTypes(curNode, t)}
}

type SimpleType struct {
	Type string
}

func (t SimpleType) EqualWithEntry(rideType Type) bool {
	return t.Equal(rideType)
}

func (t SimpleType) Equal(rideType Type) bool {
	if t.Type == "Any" {
		if T, ok := rideType.(SimpleType); ok {
			if T.Type != "Unknown" {
				return true
			}
		} else {
			return true
		}
	}
	T, ok := rideType.(SimpleType)
	if !ok {
		return false
	}
	if T.Type == "Any" {
		return true
	}
	return t.Type == T.Type
}

func (t SimpleType) String() string {
	return t.Type
}

type UnionType struct {
	Types []Type
}

func (t UnionType) Equal(rideType Type) bool {
	if rideType == Any {
		return true
	}
	T, ok := rideType.(UnionType)
	if !ok {
		return false
	}
	for _, typeName := range T.Types {
		for _, checkTypeName := range t.Types {
			if !checkTypeName.Equal(typeName) {
				return false
			}
		}
	}
	return true
}

func (t UnionType) EqualWithEntry(rideType Type) bool {
	if rideType == Any {
		return true
	}
	if T, ok := rideType.(UnionType); ok {
		for _, typeName := range T.Types {
			eq := false
			for _, checkTypeName := range t.Types {
				if checkTypeName.EqualWithEntry(typeName) {
					eq = true
					break
				}
			}
			if !eq {
				return false
			}
		}
		return true
	}
	for _, typeName := range t.Types {
		if typeName.EqualWithEntry(rideType) {
			return true
		}
	}
	return false
}

func (t *UnionType) AppendType(rideType Type) {
	// need refactor
	if rideType == nil {
		return
	}
	if newT, ok := rideType.(UnionType); ok {
		listIndex := 0
		listExist := false
		for i, existType := range t.Types {
			if _, ok := existType.(ListType); ok {
				listIndex = i
				listExist = true
				break
			}
		}
		for _, newType := range newT.Types {
			if len(t.Types) == 0 {
				t.Types = append(t.Types, newType)
				continue
			}
			exist := false
			for _, existType := range t.Types {
				if newType.Equal(existType) {
					exist = true
					break
				}
			}
			if !exist {
				if _, ok := newType.(ListType); ok && listExist {
					list := t.Types[listIndex].(ListType)
					list.AppendList(newType)
				} else {
					t.Types = append(t.Types, newType)
				}
			}
		}
		return
	}
	if len(t.Types) == 0 {
		if rideType.Equal(ThrowType) {
			return
		}
		t.Types = append(t.Types, rideType)
		return
	}
	if rideType.Equal(ThrowType) {
		return
	}
	listIndex := 0
	listExist := false
	for i, existType := range t.Types {
		if _, ok := existType.(ListType); ok {
			listIndex = i
			listExist = true
			break
		}
	}
	for _, existType := range t.Types {
		if existType.Equal(rideType) {
			return
		}
	}
	if _, ok := rideType.(ListType); ok && listExist {
		list := t.Types[listIndex].(ListType)
		list.AppendList(rideType)
		t.Types[listIndex] = list
	} else {
		t.Types = append(t.Types, rideType)
	}
}

func (t UnionType) String() string {
	var stringTypes []string
	for _, T := range t.Types {
		stringTypes = append(stringTypes, T.String())
	}
	sort.Strings(stringTypes)
	return strings.Join(stringTypes, "|")
}

func (t UnionType) Simplify() Type {
	if len(t.Types) == 1 {
		return t.Types[0]
	}
	return t
}

func JoinTypes(types ...Type) Type {
	union := UnionType{Types: []Type{}}
	for _, t := range types {
		union.AppendType(t)
	}
	return union.Simplify()
}

type ListType struct {
	Type Type
}

func (t ListType) Equal(rideType Type) bool {
	T, ok := rideType.(ListType)
	if !ok {
		return false
	}
	if t.Type == nil && T.Type == nil {
		return true
	}
	if t.Type == nil || T.Type == nil {
		return false
	}
	return t.Type.Equal(T.Type)
}

func (t ListType) EqualWithEntry(rideType Type) bool {
	if T, ok := rideType.(ListType); ok {
		if t.Type == nil && T.Type == nil {
			return true
		}
		if t.Type == nil {
			return false
		}
		if T.Type == nil {
			return true
		}
		return t.Type.EqualWithEntry(T.Type)
	}
	return false
}

func (t ListType) String() string {
	if t.Type == nil {
		return "List[]"
	}
	return "List[" + t.Type.String() + "]"
}

func (t *ListType) AppendType(rideType Type) {
	union := UnionType{Types: []Type{}}
	union.AppendType(t.Type)
	union.AppendType(rideType)
	if len(union.Types) == 1 {
		t.Type = union.Types[0]
	} else {
		t.Type = union
	}
}

func (t *ListType) AppendList(rideType Type) {
	T := rideType.(ListType)
	union := UnionType{Types: []Type{}}
	union.AppendType(t.Type)
	union.AppendType(T.Type)
	if len(union.Types) == 1 {
		t.Type = union.Types[0]
	} else {
		t.Type = union
	}
}

type TupleType struct {
	Types []Type
}

func (t TupleType) Equal(rideType Type) bool {
	T, ok := rideType.(TupleType)
	if !ok {
		return false
	}
	if len(T.Types) != len(t.Types) {
		return false
	}
	for i := 0; i < len(t.Types); i++ {
		if t.Types[i] == nil || T.Types[i] == nil {
			continue
		}
		if !t.Types[i].Equal(T.Types[i]) {
			return false
		}
	}
	return true
}

func (t TupleType) EqualWithEntry(rideType Type) bool {
	T, ok := rideType.(TupleType)
	if !ok {
		return false
	}
	if len(T.Types) != len(t.Types) {
		return false
	}
	for i := 0; i < len(t.Types); i++ {
		if t.Types[i] == nil || T.Types[i] == nil {
			continue
		}
		if !t.Types[i].EqualWithEntry(T.Types[i]) {
			return false
		}
	}
	return true
}

func (t TupleType) String() string {
	var res string
	res += "("
	for i, k := range t.Types {
		if k == nil {
			res += "nil"
		} else {
			res += k.String()
		}
		if i < len(t.Types)-1 {
			res += ", "
		}
	}
	res += ")"
	return res
}

func loadNonConfigTypes(res map[ast.LibraryVersion]map[string]Type) {
	for i := int(ast.LibV1); i <= int(ast.LibV6); i++ {
		res[ast.LibraryVersion(byte(i))]["Int"] = IntType
		res[ast.LibraryVersion(byte(i))]["String"] = StringType
		res[ast.LibraryVersion(byte(i))]["Boolean"] = BooleanType
		res[ast.LibraryVersion(byte(i))]["ByteVector"] = ByteVectorType
		res[ast.LibraryVersion(byte(i))]["Unit"] = SimpleType{Type: "Unit"}
		res[ast.LibraryVersion(byte(i))]["Ceiling"] = SimpleType{Type: "Ceiling"}
		res[ast.LibraryVersion(byte(i))]["HalfUp"] = SimpleType{Type: "HalfUp"}
		res[ast.LibraryVersion(byte(i))]["HalfEven"] = SimpleType{Type: "HalfEven"}
		res[ast.LibraryVersion(byte(i))]["Down"] = SimpleType{Type: "Down"}
		res[ast.LibraryVersion(byte(i))]["Floor"] = SimpleType{Type: "Floor"}
		res[ast.LibraryVersion(byte(i))]["Any"] = SimpleType{Type: "Any"}
	}
	for i := int(ast.LibV1); i <= int(ast.LibV4); i++ {
		res[ast.LibraryVersion(byte(i))]["HalfDown"] = SimpleType{Type: "HalfDown"}
		res[ast.LibraryVersion(byte(i))]["Up"] = SimpleType{Type: "Up"}
	}
	for i := int(ast.LibV2); i <= int(ast.LibV6); i++ {
		res[ast.LibraryVersion(byte(i))]["OrderType"] = SimpleType{Type: "OrderType"}
	}
	for i := int(ast.LibV3); i <= int(ast.LibV6); i++ {
		res[ast.LibraryVersion(byte(i))]["DigestAlgorithmType"] = SimpleType{Type: "DigestAlgorithmType"}
	}
	for i := int(ast.LibV4); i <= int(ast.LibV6); i++ {
		res[ast.LibraryVersion(byte(i))]["BlockInfo"] = SimpleType{Type: "BlockInfo"}
	}
	for i := int(ast.LibV5); i <= int(ast.LibV6); i++ {
		res[ast.LibraryVersion(byte(i))]["BigInt"] = BigIntType

	}
	// This is necessary for an exact match with scala compiler
	res[ast.LibV1]["Transaction"] = UnionType{Types: []Type{
		SimpleType{"ReissueTransaction"},
		SimpleType{"BurnTransaction"},
		SimpleType{"MassTransferTransaction"},
		SimpleType{"ExchangeTransaction"},
		SimpleType{"TransferTransaction"},
		SimpleType{"SetAssetScriptTransaction"},
		SimpleType{"IssueTransaction"},
		SimpleType{"LeaseTransaction"},
		SimpleType{"LeaseCancelTransaction"},
		SimpleType{"CreateAliasTransaction"},
		SimpleType{"SetScriptTransaction"},
		SimpleType{"SponsorFeeTransaction"},
		SimpleType{"DataTransaction"},
	}}
	res[ast.LibV2]["Transaction"] = res[ast.LibV1]["Transaction"]
	res[ast.LibV3]["Transaction"] = UnionType{Types: []Type{
		SimpleType{"ReissueTransaction"},
		SimpleType{"BurnTransaction"},
		SimpleType{"MassTransferTransaction"},
		SimpleType{"ExchangeTransaction"},
		SimpleType{"TransferTransaction"},
		SimpleType{"SetAssetScriptTransaction"},
		SimpleType{"InvokeScriptTransaction"},
		SimpleType{"IssueTransaction"},
		SimpleType{"LeaseTransaction"},
		SimpleType{"LeaseCancelTransaction"},
		SimpleType{"CreateAliasTransaction"},
		SimpleType{"SetScriptTransaction"},
		SimpleType{"SponsorFeeTransaction"},
		SimpleType{"DataTransaction"},
	}}
	res[ast.LibV4]["Transaction"] = UnionType{Types: []Type{
		SimpleType{"ReissueTransaction"},
		SimpleType{"BurnTransaction"},
		SimpleType{"MassTransferTransaction"},
		SimpleType{"ExchangeTransaction"},
		SimpleType{"TransferTransaction"},
		SimpleType{"SetAssetScriptTransaction"},
		SimpleType{"InvokeScriptTransaction"},
		SimpleType{"UpdateAssetInfoTransaction"},
		SimpleType{"IssueTransaction"},
		SimpleType{"LeaseTransaction"},
		SimpleType{"LeaseCancelTransaction"},
		SimpleType{"CreateAliasTransaction"},
		SimpleType{"SetScriptTransaction"},
		SimpleType{"SponsorFeeTransaction"},
		SimpleType{"DataTransaction"},
	}}
	res[ast.LibV5]["Transaction"] = res[ast.LibV4]["Transaction"]
	res[ast.LibV6]["Transaction"] = UnionType{Types: []Type{
		SimpleType{"ReissueTransaction"},
		SimpleType{"BurnTransaction"},
		SimpleType{"MassTransferTransaction"},
		SimpleType{"ExchangeTransaction"},
		SimpleType{"TransferTransaction"},
		SimpleType{"SetAssetScriptTransaction"},
		SimpleType{"InvokeScriptTransaction"},
		SimpleType{"UpdateAssetInfoTransaction"},
		SimpleType{"InvokeExpressionTransaction"},
		SimpleType{"IssueTransaction"},
		SimpleType{"LeaseTransaction"},
		SimpleType{"LeaseCancelTransaction"},
		SimpleType{"CreateAliasTransaction"},
		SimpleType{"SetScriptTransaction"},
		SimpleType{"SponsorFeeTransaction"},
		SimpleType{"DataTransaction"},
	}}
}

func mustLoadDefaultTypes() map[ast.LibraryVersion]map[string]Type {
	res := map[ast.LibraryVersion]map[string]Type{
		ast.LibV1: {
			"Transaction": UnionType{Types: []Type{}},
		},
		ast.LibV2: {
			"Transaction": UnionType{Types: []Type{}},
		},
		ast.LibV3: {
			"Transaction": UnionType{Types: []Type{}},
		},
		ast.LibV4: {
			"Transaction": UnionType{Types: []Type{}},
		},
		ast.LibV5: {
			"Transaction": UnionType{Types: []Type{}},
		},
		ast.LibV6: {
			"Transaction": UnionType{Types: []Type{}},
		},
	}

	f, err := embedFS.ReadFile("ride_objects.json")
	if err != nil {
		panic(err)
	}
	s := &rideObjects{}
	if err = json.Unmarshal(f, s); err != nil {
		panic(err)
	}
	appendRemainingStructs(s)
	loadNonConfigTypes(res)
	for _, obj := range s.Objects {
		sort.SliceStable(obj.Actions, func(i, j int) bool {
			return int(obj.Actions[i].LibVersion) < int(obj.Actions[j].LibVersion)
		})
		for _, ver := range obj.Actions {
			maxVer := int(ast.CurrentMaxLibraryVersion())
			if ver.Deleted != nil {
				maxVer = int(*ver.Deleted)
			}
			for i := int(ver.LibVersion); i <= maxVer; i++ {
				res[ast.LibraryVersion(byte(i))][obj.Name] = SimpleType{Type: obj.Name}
			}
		}
	}
	return res
}
