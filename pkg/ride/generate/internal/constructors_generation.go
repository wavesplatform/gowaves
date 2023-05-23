package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"
	"github.com/wavesplatform/gowaves/pkg/ride/generate/internal/vinfo"
)

func constructorName(act actionsObject) string {
	return strings.ToLower(string(act.StructName[0])) + act.StructName[1:] + "Constructor"
}

func argVarName(act actionField) string {
	return act.Name
}

func extractConstructorArguments(args []actionField) ([]actionField, error) {
	var arguments []actionField
	seenOrders := map[int]struct{}{}

	for _, field := range args {
		if _, ok := seenOrders[field.ConstructorOrder]; ok {
			return nil, errors.Errorf("Duplicate constructor_order: %d", field.ConstructorOrder)
		}
		seenOrders[field.ConstructorOrder] = struct{}{}
		arguments = append(arguments, field)
	}

	for i := 0; i < len(seenOrders); i++ {
		if _, ok := seenOrders[i]; !ok {
			return nil, errors.Errorf("constructor_order %d is missing", i)
		}
	}

	sort.Slice(arguments, func(i, j int) bool {
		return arguments[i].ConstructorOrder < arguments[j].ConstructorOrder
	})

	return arguments, nil
}

func checkListElementsTypes(cd *Coder, constructorName string, topListVarName string, list stdlib.ListType) {
	var helper func(listVarName string, ti stdlib.Type)

	helper = func(listVarName string, t stdlib.Type) {
		cd.Line("for _, el := range %s {", listVarName)
		cd.Line("switch te := el.(type) {")

		switch tt := t.(type) {
		case stdlib.SimpleType:
			cd.Line("case %s:", getType(tt))
		case stdlib.UnionType:
			ts, l, lt := getUnionType(tt)
			if l {
				cd.Line("case %s:", getType(lt))
				helper("te", lt.Type)
			}
			cd.Line("case %s:", ts)
		case stdlib.TupleType:
			cd.Line("case tuple%d:", len(tt.Types))
			for i, elt := range tt.Types {
				cd.Line("if _, ok := te.el%d.(%s); !ok {", i+1, getType(elt))
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' of element %d in %s list tuple\", te.el%d.instanceOf())", constructorName, i+1, topListVarName, i+1)
				cd.Line("}")
			}
		case stdlib.ListType:
			cd.Line("case %s: ", getType(tt))
			helper(listVarName, tt.Type)
		}
		cd.Line("default:")
		cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' in %s list\", te.instanceOf())", constructorName, topListVarName)
		cd.Line("}")
		cd.Line("}")
	}
	helper(topListVarName, list.Type)
}

func constructorsFunctions(ver ast.LibraryVersion, m map[string]string) {
	verInfo, ok := vinfo.GetVerInfos()[ver]
	if !ok {
		panic(fmt.Sprintf("version %d is missing in vinfo.GetVerInfos()", ver))
	}

	for _, name := range verInfo.RemovedStructs {
		delete(m, name)
	}
	for _, structInfo := range verInfo.NewStructs {
		m[structInfo.RideName] = structInfo.GoName
	}
}

func constructorsCatalogue(ver ast.LibraryVersion, m map[string]int) {
	for _, name := range vinfo.GetVerInfos()[ver].RemovedStructs {
		delete(m, name)
	}
	for _, structInfo := range vinfo.GetVerInfos()[ver].NewStructs {
		m[structInfo.RideName] = structInfo.ArgsNumber
	}
}

func constructorsEvaluationCatalogueEvaluatorV1(ver ast.LibraryVersion, m map[string]int) {
	for _, name := range vinfo.GetVerInfos()[ver].RemovedStructs {
		delete(m, name)
	}
	for _, structInfo := range vinfo.GetVerInfos()[ver].NewStructs {
		m[structInfo.RideName] = 0
	}
}

func constructorsEvaluationCatalogueEvaluatorV2(ver ast.LibraryVersion, m map[string]int) {
	for _, name := range vinfo.GetVerInfos()[ver].RemovedStructs {
		delete(m, name)
	}
	for _, structInfo := range vinfo.GetVerInfos()[ver].NewStructs {
		m[structInfo.RideName] = 1
	}
}

func processVerInfos() error {
	existingStructs := map[string]vinfo.ConstructorStructInfo{}
	for ver := ast.LibV1; ver <= ast.CurrentMaxLibraryVersion(); ver++ {
		verInfo := vinfo.GetVerInfos()[ver]
		if verInfo == nil {
			verInfo = vinfo.NewVersionInfo(ver)
			vinfo.GetVerInfos()[ver] = verInfo
		}

		for _, name := range verInfo.RemovedStructs {
			delete(existingStructs, name)
		}
		for _, structInfo := range verInfo.NewStructs {
			existingStructs[structInfo.RideName] = structInfo
		}

		verInfo.NewStructs = make([]vinfo.ConstructorStructInfo, 0, len(existingStructs))
		for _, structInfo := range existingStructs {
			verInfo.NewStructs = append(verInfo.NewStructs, structInfo)
		}
	}

	return nil
}

func constructorsHandleRideObject(cd *Coder, obj rideObject) error {
	if obj.SkipConstructor {
		return nil
	}

	for _, act := range obj.Actions {
		constructorName := constructorName(act)
		cd.Line("func %s(_ environment, args_ ...rideType) (rideType, error) {", constructorName)

		arguments, err := extractConstructorArguments(act.Fields)
		if err != nil {
			panic(errors.Wrap(err, obj.Name).Error())
		}

		cd.Line("if err := checkArgs(args_, %d); err != nil {", len(arguments))
		cd.Line("return nil, errors.Wrap(err, \"%s\")", constructorName)
		cd.Line("}")
		cd.Line("")

		for i, arg := range arguments {
			varName := argVarName(arg)
			t := stdlib.ParseRuntimeType(arg.Type)
			switch att := t.(type) {
			case stdlib.SimpleType:
				cd.Line("%s, ok := args_[%d].(%s)", varName, i, getType(att))
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")
			case stdlib.TupleType:
				l := len(att.Types)
				cd.Line("%s, ok := args_[%d].(tuple%d)", varName, i, l)
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")
				for i, elt := range att.Types {
					cd.Line("if _, ok := t.el%d.(%s); !ok {", i+1, getType(elt))
					cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' of element %d in %s tuple\", t.instanceOf())", constructorName, i+1, varName)
					cd.Line("}")
				}

			case stdlib.ListType:
				cd.Line("%s, ok := args_[%d].(%s)", varName, i, getType(att))
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")
				cd.Line("// checks for list elements")
				checkListElementsTypes(cd, constructorName, varName, att)

			case stdlib.UnionType:
				ts, _, _ := getUnionType(att)
				cd.Line("var %s rideType", varName)
				cd.Line("switch v := args_[%d].(type) {", i)
				cd.Line("case %s:", ts)
				cd.Line("%s = v", varName)
				cd.Line("default:")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")
			}
			cd.Line("")
		}

		argsStr := make([]string, len(act.Fields))
		for i, arg := range act.Fields {
			varName := argVarName(arg)
			argsStr[i] = varName
		}

		cd.Line("")
		cd.Line("return %s(%s), nil", rideActionConstructorName(act), strings.Join(argsStr, ", "))
		cd.Line("}")
		cd.Line("")

		if act.Deleted != nil {
			vinfo.GetVerInfos().AddRemoved(*act.Deleted, obj.Name)
		}

		vinfo.GetVerInfos().AddNewStruct(act.LibVersion, vinfo.ConstructorStructInfo{
			RideName:   obj.Name,
			GoName:     constructorName,
			ArgsNumber: len(arguments),
		})
	}

	return nil
}

func GenerateConstructors(configPath, fn string) {
	s, err := parseConfig(configPath)
	if err != nil {
		panic(err)
	}

	cd := NewCoder("ride")
	cd.Import("github.com/pkg/errors")

	for _, obj := range s.Objects {
		if err := constructorsHandleRideObject(cd, obj); err != nil {
			panic(err)
		}
	}

	if err := cd.Save(fn); err != nil {
		panic(err)
	}

	if err := processVerInfos(); err != nil {
		panic(err)
	}
}
