package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"
)

func removeVersionFromString(name string) string {
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		vs := fmt.Sprintf("V%d", v)
		name = strings.ReplaceAll(name, vs, "")
	}
	return name
}

func getType(t stdlib.Type) string {
	switch t.(type) {
	case stdlib.SimpleType:
		return "ride" + t.String()
	case stdlib.ListType:
		return "rideList"
	default:
		return "rideType"
	}
}

func getUnionType(union stdlib.UnionType) (string, bool, stdlib.ListType) {
	ns := make([]string, 0, len(union.Types))
	hasList := false
	listType := stdlib.ListType{}
	for _, t := range union.Types {
		switch tt := t.(type) {
		case stdlib.ListType:
			hasList = true
			listType = tt
		case stdlib.SimpleType:
			ns = append(ns, getType(t))
		}
	}
	return strings.Join(ns, ", "), hasList, listType
}

func rideActionConstructorName(act actionsObject) string {
	return "newRide" + act.StructName
}

func stringMapToSortedList(m map[string]struct{}) []string {
	list := make([]string, 0, len(m))
	for k := range m {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

func GenerateObjects(configPath, fn string) {
	s, err := parseConfig(configPath)
	if err != nil {
		panic(err)
	}
	cd := NewCoder("ride")
	cd.Import("strings")
	cd.Import("github.com/pkg/errors")

	fields := map[string]struct{}{}
	names := map[string]struct{}{}
	for _, o := range s.Objects {
		for _, a := range o.Actions {
			name := removeVersionFromString(a.StructName)
			names[name] = struct{}{}
			for _, f := range a.Fields {
				fields[f.Name] = struct{}{}
			}
		}
	}

	fieldsList := stringMapToSortedList(fields)
	namesList := stringMapToSortedList(names)

	cd.Line("const (")
	for _, n := range namesList {
		firstLetter := string(n[0])
		firstLetter = strings.ToLower(firstLetter)
		str := firstLetter + n[1:]
		cd.Line("%sTypeName = \"%s\"", str, n)
	}
	cd.Line(")")
	cd.Line("")
	cd.Line("const (")
	for _, f := range fieldsList {
		str := strings.ReplaceAll(f, "Id", "ID")
		cd.Line("%sField = \"%s\"", str, f)
	}
	cd.Line(")")
	cd.Line("")

	for _, obj := range s.Objects {
		for _, act := range obj.Actions {
			// Struct Implementation
			cd.Line("type ride%s struct {", act.StructName)
			for _, field := range act.Fields {
				ft := stdlib.ParseRuntimeType(field.Type)
				cd.Line("%s %s", field.Name, getType(ft))
			}
			cd.Line("}")
			cd.Line("")

			// Constructor
			constructorName := rideActionConstructorName(act)
			arguments := make([]string, len(act.Fields))
			for i, field := range act.Fields {
				ft := stdlib.ParseRuntimeType(field.Type)
				arguments[i] = fmt.Sprintf("%s %s", field.Name, getType(ft))
			}
			cd.Line("func %s(%s) ride%s {", constructorName, strings.Join(arguments, ", "), act.StructName)
			cd.Line("return ride%s{", act.StructName)
			for _, field := range act.Fields {
				cd.Line("%s: %s,", field.Name, field.Name)
			}
			cd.Line("}")
			cd.Line("}")
			cd.Line("")

			// instanceOf method
			cd.Line("func (o ride%s) instanceOf() string {", act.StructName)
			cd.Line("return \"%s\"", obj.Name)
			cd.Line("}")
			cd.Line("")

			// eq method
			cd.Line("func (o ride%s) eq(other rideType) bool {", act.StructName)
			cd.Line("if oo, ok := other.(ride%s); ok {", act.StructName)
			for _, field := range act.Fields {
				cd.Line("if !o.%s.eq(oo.%s) {", field.Name, field.Name)
				cd.Line("return false")
				cd.Line("}")
			}
			cd.Line("return true")
			cd.Line("}")
			cd.Line("return false")
			cd.Line("}")
			cd.Line("")

			// get method
			cd.Line("func (o ride%s) get(prop string) (rideType, error) {", act.StructName)
			cd.Line("switch prop {")
			cd.Line("case \"$instance\":")
			cd.Line("return rideString(\"%s\"), nil", obj.Name)
			for _, field := range act.Fields {
				cd.Line("case \"%s\":", field.Name)
				cd.Line("return o.%s, nil", field.Name)
			}
			cd.Line("default:")
			cd.Line("return nil, errors.Errorf(\"type '%%s' has no property '%%s'\", o.instanceOf(), prop)")
			cd.Line("}")
			cd.Line("}")
			cd.Line("")

			//copy method
			for i, field := range act.Fields {
				arguments[i] = fmt.Sprintf("o.%s", field.Name)
			}
			cd.Line("func (o ride%s) copy() rideType {", act.StructName)
			cd.Line("return %s(%s)", constructorName, strings.Join(arguments, ", "))
			cd.Line("}")
			cd.Line("")

			// lines method
			cd.Line("func (o ride%s) lines() []string {", act.StructName)
			cd.Line("r := make([]string, 0, %d)", len(act.Fields)+2)
			cd.Line("r = append(r, \"%s(\")", obj.Name)
			sort.SliceStable(act.Fields, func(i, j int) bool {
				return act.Fields[i].Order < act.Fields[j].Order
			})
			for _, field := range act.Fields {
				cd.Line("r = append(r, fieldLines(\"%s\", o.%s.lines())...)", field.Name, field.Name)
			}
			cd.Line("r = append(r, \")\")")
			cd.Line("return r")
			cd.Line("}")
			cd.Line("")

			// String method
			cd.Line("func (o ride%s) String() string {", act.StructName)
			cd.Line("return strings.Join(o.lines(), \"\\n\")")
			cd.Line("}")
			cd.Line("")

			// SetProofs (only for transactions)
			if act.SetProofs {
				cd.Line("func (o ride%s) setProofs(proofs rideList) rideProven {", act.StructName)
				cd.Line("o.proofs = proofs")
				cd.Line("return o")
				cd.Line("}")
				cd.Line("")
				cd.Line("func (o ride%s) getProofs() rideList {", act.StructName)
				cd.Line("return o.proofs")
				cd.Line("}")
				cd.Line("")
			}
		}
	}
	// ResetProofs (only for transactions)
	cd.Line("func resetProofs(obj rideType) error {")
	cd.Line("switch tx := obj.(type) {")
	cd.Line("case rideProven:")
	cd.Line("tx.setProofs(rideList{})")
	cd.Line("default:")
	cd.Line("return errors.Errorf(\"type '%%s' is not tx\", obj.instanceOf())")
	cd.Line("}")
	cd.Line("return nil")
	cd.Line("}")
	cd.Line("")

	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
