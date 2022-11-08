package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func constructorName(act *actionsObject) string {
	return strings.ToLower(string(act.StructName[0])) + act.StructName[1:] + "Constructor"
}

func argVarName(act *actionField) string {
	return act.Name
}

func extractConstructorArguments(args []actionField) ([]actionField, error) {
	arguments := []actionField{}
	seenOrders := map[int]bool{}

	for _, field := range args {
		if field.ConstructorOrder == -1 {
			continue
		}
		if seen := seenOrders[field.ConstructorOrder]; seen {
			return nil, errors.Errorf("Duplicate constructor_order: %d", field.ConstructorOrder)
		}
		seenOrders[field.ConstructorOrder] = true
		arguments = append(arguments, field)
	}

	for i := 0; i < len(seenOrders); i++ {
		if !seenOrders[i] {
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

type constructorStructInfo struct {
	rideName   string
	goName     string
	argsNumber int
}

type versionInfo struct {
	version        ast.LibraryVersion
	newStructs     []constructorStructInfo // new structs or modified structs
	removedStructs map[string]bool         // structs removed in this version
}

func newVersionInfo(version ast.LibraryVersion) *versionInfo {
	return &versionInfo{
		version:        version,
		newStructs:     make([]constructorStructInfo, 0),
		removedStructs: make(map[string]bool),
	}
}

type versionInfos map[ast.LibraryVersion]*versionInfo

func (vInfos versionInfos) addNewStruct(version ast.LibraryVersion, info constructorStructInfo) {
	if _, ok := vInfos[version]; !ok {
		fmt.Printf("adding ver %d\n", version)
		vInfos[version] = newVersionInfo(version)
	}

	vInfo := vInfos[version]
	vInfo.newStructs = append(vInfo.newStructs, info)
}

func (vInfos versionInfos) addRemoved(version ast.LibraryVersion, name string) {
	if _, ok := vInfos[version]; !ok {
		fmt.Printf("adding ver %d\n", version)
		vInfos[version] = newVersionInfo(version)
	}

	vInfo := vInfos[version]
	vInfo.removedStructs[name] = true
}

func constructorsFunctions(ver ast.LibraryVersion, m map[string]string) {
	verInfo, ok := verInfos[ver]
	if !ok {
		panic(fmt.Sprintf("version %d is missing in verInfos", ver))
	}

	for name := range verInfo.removedStructs {
		delete(m, name)
	}
	for _, structInfo := range verInfo.newStructs {
		m[structInfo.rideName] = structInfo.goName
	}
}

func constructorsCatalogue(ver ast.LibraryVersion, m map[string]int) {
	for name := range verInfos[ver].removedStructs {
		delete(m, name)
	}
	for _, structInfo := range verInfos[ver].newStructs {
		m[structInfo.rideName] = structInfo.argsNumber
	}
}

func constructorsEvaluationCatalogueEvaluatorV1(ver ast.LibraryVersion, m map[string]int) {
	for name := range verInfos[ver].removedStructs {
		delete(m, name)
	}
	for _, structInfo := range verInfos[ver].newStructs {
		m[structInfo.rideName] = 0
	}
}

func constructorsEvaluationCatalogueEvaluatorV2(ver ast.LibraryVersion, m map[string]int) {
	for name := range verInfos[ver].removedStructs {
		delete(m, name)
	}
	for _, structInfo := range verInfos[ver].newStructs {
		m[structInfo.rideName] = 1
	}
}

func processVerInfos() error {
	var maxVersion byte = 0
	for ver := range verInfos {
		if byte(ver) > maxVersion {
			maxVersion = byte(ver)
		}
	}

	existingStructs := map[string]constructorStructInfo{}
	for ver := ast.LibV1; ver <= ast.CurrentMaxLibraryVersion(); ver++ {
		verInfo := verInfos[ver]
		if verInfo == nil {
			verInfo = newVersionInfo(ver)
			verInfos[ver] = verInfo
		}

		for name := range verInfo.removedStructs {
			delete(existingStructs, name)
		}
		for _, structInfo := range verInfo.newStructs {
			existingStructs[structInfo.rideName] = structInfo
		}

		verInfo.newStructs = make([]constructorStructInfo, 0, len(existingStructs))
		for _, structInfo := range existingStructs {
			verInfo.newStructs = append(verInfo.newStructs, structInfo)
		}

		// // functionsV[N]
		// cd.Line("func constructorsFunctionsV%d(m map[string]string) {", verInfo.version)
		// for name := range verInfo.removedStructs {
		// 	cd.Line("delete(m, \"%s\")", name)
		// }
		// for _, structInfo := range existingStructs {
		// 	cd.Line("m[\"%s\"] = \"%s\"", structInfo.rideName, structInfo.goName)
		// }
		// cd.Line("}")
		// cd.Line("")

		// // catalogueV[N]
		// cd.Line("func constructorsCatalogueV%d(m map[string]int) {", verInfo.version)
		// for name := range verInfo.removedStructs {
		// 	cd.Line("delete(m, \"%s\")", name)
		// }
		// for _, structInfo := range existingStructs {
		// 	cd.Line("m[\"%s\"] = %d", structInfo.rideName, structInfo.argsNumber)
		// }
		// cd.Line("}")
		// cd.Line("")

		// // constructorsEvaluationCatalogueV[N]EvaluatorV1
		// cd.Line("func constructorsEvaluationCatalogueV%dEvaluatorV1(m map[string]int) {", verInfo.version)
		// for _, structInfo := range existingStructs {
		// 	cd.Line("m[\"%s\"] = 0", structInfo.rideName)
		// }
		// cd.Line("}")
		// cd.Line("")

		// // constructorsEvaluationCatalogueV[N]EvaluatorV2
		// cd.Line("func constructorsEvaluationCatalogueV%dEvaluatorV2(m map[string]int) {", verInfo.version)
		// for _, structInfo := range existingStructs {
		// 	cd.Line("m[\"%s\"] = 1", structInfo.rideName)
		// }
		// cd.Line("}")
		// cd.Line("")
	}

	return nil
}

func constructorsHandleRideObject(cd *Coder, obj *rideObject) error {
	if obj.SkipConstructor {
		return nil
	}

	for _, act := range obj.Actions {
		constructorName := constructorName(&act)
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
			varName := argVarName(&arg)

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
				for _, t := range arg.Types {
					cd.Line("case %s:", t)
					cd.Line("%s = v", varName)
				}
				cd.Line("default:")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, varName, i)
				cd.Line("}")
			}
			cd.Line("")
		}

		argsStr := make([]string, len(act.Fields))
		for i, arg := range act.Fields {
			varName := argVarName(&arg)
			if arg.ConstructorOrder == -1 {
				cd.Line("// default values for internal fields")
				cd.Line("var %s %s", varName, getType(arg.Types))
			}
			argsStr[i] = varName
		}

		cd.Line("")
		cd.Line("return %s(%s), nil", rideActionConstructorName(&act), strings.Join(argsStr, ", "))
		cd.Line("}")
		cd.Line("")

		if act.Deleted != nil {
			verInfos.addRemoved(*act.Deleted, obj.Name)
		}

		verInfos.addNewStruct(act.LibVersion, constructorStructInfo{
			rideName:   obj.Name,
			goName:     constructorName,
			argsNumber: len(arguments),
		})
	}

	return nil
}

var verInfos = versionInfos{}

func GenerateConstructors(fn string) {
	s, err := parseConfig()
	if err != nil {
		panic(err)
	}

	cd := NewCoder("ride")
	cd.Import("github.com/pkg/errors")

	for _, obj := range s.Objects {
		if err := constructorsHandleRideObject(cd, &obj); err != nil {
			panic(err)
		}
	}

	if err := cd.Save(fn); err != nil {
		panic(err)
	}

	if err := processVerInfos(); err != nil {
		panic(err)
	}

	// cd = NewCoder("internal")
	// if err := generateConstructorFunctionCatalogues(cd, verInfos); err != nil {
	// 	panic(err)
	// }

	// if err := cd.Save(fnCatalogue); err != nil {
	// 	panic(err)
	// }
}
