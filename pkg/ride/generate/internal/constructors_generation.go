package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
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

func checkListElementsTypes(cd *Coder, constructorName string, topListVarName string, info *listTypeInfo) {
	var helper func(listVarName string, info *listTypeInfo)

	helper = func(listVarName string, info *listTypeInfo) {
		cd.Line("for _, elem := range %s {", listVarName)
		cd.Line("switch t := elem.(type) {")

		onelineTypes := make([]string, 0, len(info.ElementTypes()))
		for _, tInfo := range info.ElementTypes() {
			switch t := tInfo.(type) {
			case *listTypeInfo:
				cd.Line("case %s: ", t.String())
				helper("t", t)
			default:
				onelineTypes = append(onelineTypes, t.String())
			}
		}

		cd.Line("case %s:", strings.Join(onelineTypes, ","))
		cd.Line("default:")
		cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' in %s list\", t.instanceOf())", constructorName, topListVarName)
		cd.Line("}")
		cd.Line("}")
	}
	helper(topListVarName, info)
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

			if len(arg.Types) == 1 {
				info := arg.Types[0]
				cd.Line("%s, ok := args_[%d].(%s)", varName, i, info)
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")

				if listInfo, ok := info.(*listTypeInfo); ok {
					cd.Line("// checks for list elements")
					checkListElementsTypes(cd, constructorName, varName, listInfo)
				}
			} else {
				cd.Line("var %s rideType", varName)
				cd.Line("switch v := args_[%d].(type) {", i)
				onelineTypes := make([]string, 0, len(arg.Types))
				for _, t := range arg.Types {
					onelineTypes = append(onelineTypes, t.String())
				}
				cd.Line("case %s:", strings.Join(onelineTypes, ", "))
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
