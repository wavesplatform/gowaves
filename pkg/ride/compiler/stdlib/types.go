package stdlib

import (
	"encoding/json"
	"slices"
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
	AnyType        = anyType{}
	BooleanType    = SimpleType{"Boolean"}
	IntType        = SimpleType{"Int"}
	StringType     = SimpleType{"String"}
	ByteVectorType = SimpleType{"ByteVector"}
	BigIntType     = SimpleType{"BigInt"}
	ThrowType      = throwType{}

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
			AnyType,
		}},
		callableRetV5OnlyList,
	}}
)

func ParseType(t string) Type {
	return parseType(t, true)
}

func ParseRuntimeType(t string) Type {
	return parseType(t, false)
}

func parseType(t string, dropRuntimeTypes bool) Type {
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
	return handleTypes(node.up, t, dropRuntimeTypes)
}

func handleTypes(node *node32, s string, dropRuntimeTypes bool) Type {
	curNode := node.up
	var t Type
	switch curNode.pegRule {
	case ruleGenericType:
		t = handleGeneric(curNode, s, dropRuntimeTypes)
	case ruleTupleType:
		t = handleTupleType(curNode, s, dropRuntimeTypes)
	case ruleType:
		// check Types
		stringType := s[curNode.begin:curNode.end]
		switch stringType {
		case "Any":
			t = AnyType
		case "Unknown":
			t = ThrowType
		case "AddressLike": // Replace implementation specific AddressLike with Address, duplications handled later
			if dropRuntimeTypes {
				t = SimpleType{Type: "Address"}
			} else {
				t = SimpleType{Type: "AddressLike"}
			}
		default:
			t = SimpleType{stringType}
		}
	}
	curNode = curNode.next
	if curNode == nil {
		return t
	}

	resType := UnionType{Types: []Type{}}
	resType.AppendType(t)
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	resType.AppendType(handleTypes(curNode, s, dropRuntimeTypes))
	return resType
}

func handleTupleType(node *node32, t string, dropRuntimeTypes bool) Type {
	curNode := node.up
	var tupleTypes []Type
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == ruleTypes {
			tupleTypes = append(tupleTypes, handleTypes(curNode, t, dropRuntimeTypes))
			curNode = curNode.next
		}
		if curNode == nil {
			break
		}
	}
	return TupleType{tupleTypes}
}

func handleGeneric(node *node32, t string, dropRuntimeTypes bool) Type {
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

	return ListType{Type: handleTypes(curNode, t, dropRuntimeTypes)}
}

type throwType struct{}

func (t throwType) String() string {
	return "Unknown"
}

func (t throwType) Equal(other Type) bool {
	_, ok := other.(throwType)
	return ok
}

func (t throwType) EqualWithEntry(other Type) bool {
	return t.Equal(other)
}

type anyType struct{}

func (t anyType) String() string {
	return "Any"
}

func (t anyType) Equal(other Type) bool {
	_, ok := other.(anyType)
	return ok
}

func (t anyType) EqualWithEntry(_ Type) bool {
	return true
}

type SimpleType struct {
	Type string
}

func (t SimpleType) EqualWithEntry(other Type) bool {
	return t.Equal(other)
}

func (t SimpleType) Equal(other Type) bool {
	if o, ok := other.(SimpleType); ok {
		return t.Type == o.Type
	}
	return other.Equal(AnyType)
}

func (t SimpleType) String() string {
	return t.Type
}

type UnionType struct {
	Types []Type
}

func (t UnionType) Equal(other Type) bool {
	if o, ok := other.(UnionType); ok {
		for _, ot := range o.Types {
			for _, tt := range t.Types {
				if !tt.Equal(ot) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (t UnionType) EqualWithEntry(other Type) bool {
	switch o := other.(type) {
	case anyType:
		return false
	case UnionType:
		for _, ot := range o.Types {
			eq := false
			for _, tt := range t.Types {
				if tt.EqualWithEntry(ot) {
					eq = true
					break
				}
			}
			if !eq {
				return false
			}
		}
		return true
	default:
		for _, tt := range t.Types {
			if tt.EqualWithEntry(other) {
				return true
			}
		}
		return false
	}
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
			exist := slices.ContainsFunc(t.Types, newType.Equal)
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
	var types []string
	for _, tt := range t.Types {
		types = append(types, tt.String())
	}
	sort.Strings(types)
	return strings.Join(types, "|")
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

func (t ListType) Equal(other Type) bool {
	o, ok := other.(ListType)
	if !ok {
		return false
	}
	if t.Type == nil && o.Type == nil {
		return true
	}
	if t.Type == nil || o.Type == nil {
		return false
	}
	return t.Type.Equal(o.Type)
}

func (t ListType) EqualWithEntry(other Type) bool {
	if o, ok := other.(ListType); ok {
		if t.Type == nil && o.Type == nil {
			return true
		}
		if t.Type == nil {
			return false
		}
		if o.Type == nil {
			return true
		}
		return t.Type.EqualWithEntry(o.Type)
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
	o := rideType.(ListType)
	union := UnionType{Types: []Type{}}
	union.AppendType(t.Type)
	union.AppendType(o.Type)
	if len(union.Types) == 1 {
		t.Type = union.Types[0]
	} else {
		t.Type = union
	}
}

type TupleType struct {
	Types []Type
}

func (t TupleType) Equal(other Type) bool {
	o, ok := other.(TupleType)
	if !ok {
		return false
	}
	if len(o.Types) != len(t.Types) {
		return false
	}
	for i := 0; i < len(t.Types); i++ {
		if t.Types[i] == nil || o.Types[i] == nil {
			continue
		}
		if !t.Types[i].Equal(o.Types[i]) {
			return false
		}
	}
	return true
}

func (t TupleType) EqualWithEntry(other Type) bool {
	o, ok := other.(TupleType)
	if !ok {
		return false
	}
	if len(o.Types) != len(t.Types) {
		return false
	}
	for i := 0; i < len(t.Types); i++ {
		if t.Types[i] == nil || o.Types[i] == nil {
			continue
		}
		if !t.Types[i].EqualWithEntry(o.Types[i]) {
			return false
		}
	}
	return true
}

func (t TupleType) String() string {
	var res strings.Builder
	res.WriteString("(")
	for i, k := range t.Types {
		if k == nil {
			res.WriteString("nil")
		} else {
			res.WriteString(k.String())
		}
		if i < len(t.Types)-1 {
			res.WriteString(", ")
		}
	}
	res.WriteString(")")
	return res.String()
}

func loadNonConfigTypes(res map[ast.LibraryVersion]map[string]Type) {
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v]["Int"] = IntType
		res[v]["String"] = StringType
		res[v]["Boolean"] = BooleanType
		res[v]["ByteVector"] = ByteVectorType
		res[v]["Unit"] = SimpleType{Type: "Unit"}
		res[v]["Ceiling"] = SimpleType{Type: "Ceiling"}
		res[v]["HalfUp"] = SimpleType{Type: "HalfUp"}
		res[v]["HalfEven"] = SimpleType{Type: "HalfEven"}
		res[v]["Down"] = SimpleType{Type: "Down"}
		res[v]["Floor"] = SimpleType{Type: "Floor"}
		res[v]["Any"] = AnyType
	}
	for v := ast.LibV1; v <= ast.LibV4; v++ {
		res[v]["HalfDown"] = SimpleType{Type: "HalfDown"}
		res[v]["Up"] = SimpleType{Type: "Up"}
	}
	for v := ast.LibV2; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v]["OrderType"] = SimpleType{Type: "OrderType"}
	}
	for v := ast.LibV3; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v]["DigestAlgorithmType"] = SimpleType{Type: "DigestAlgorithmType"}
	}
	for v := ast.LibV4; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v]["BlockInfo"] = SimpleType{Type: "BlockInfo"}
	}
	for v := ast.LibV5; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v]["BigInt"] = BigIntType

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
	res[ast.LibV7]["Transaction"] = res[ast.LibV6]["Transaction"]
	res[ast.LibV8]["Transaction"] = res[ast.LibV7]["Transaction"]
}

func mustLoadDefaultTypes() map[ast.LibraryVersion]map[string]Type {
	res := make(map[ast.LibraryVersion]map[string]Type)
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		res[v] = map[string]Type{"Transaction": UnionType{Types: []Type{}}}
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
			return obj.Actions[i].LibVersion < obj.Actions[j].LibVersion
		})
		for _, ver := range obj.Actions {
			maxVer := ast.CurrentMaxLibraryVersion()
			if ver.Deleted != nil {
				maxVer = *ver.Deleted
			}
			for v := ver.LibVersion; v <= maxVer; v++ {
				res[v][obj.Name] = SimpleType{Type: obj.Name}
			}
		}
	}
	return res
}
