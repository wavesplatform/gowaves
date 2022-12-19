package stdlib

import (
	"reflect"

	"github.com/pkg/errors"
)

type Type interface {
	Comp(Type) bool
	String() string
}

var (
	Any            = SimpleType{"Any"}
	BooleanType    = SimpleType{"Boolean"}
	IntType        = SimpleType{"Int"}
	StringType     = SimpleType{"String"}
	ByteVectorType = SimpleType{"ByteVector"}
	BigIntType     = SimpleType{"BigInt"}
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

	resType := UnionType{Types: map[string]Type{}}
	resType.AppendType(T)
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	resType.AppendType(handleTypes(curNode, t))
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

func (t SimpleType) Comp(rideType Type) bool {
	if t.Type == "Any" {
		return true
	}
	if _, ok := rideType.(UnionType); ok {
		return rideType.Comp(t)
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
	if t.Type == nil {
		return "List[]"
	}
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

type TupleType struct {
	Types []Type
}

func (t TupleType) Comp(rideType Type) bool {
	if T, ok := rideType.(TupleType); ok {
		for i := 0; i <= len(t.Types); i++ {
			if t.Types[i] == nil || T.Types[i] == nil {
				continue
			}
			if !t.Types[i].Comp(T.Types[i]) {
				return false
			}
		}
		return true
	}
	return false
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
