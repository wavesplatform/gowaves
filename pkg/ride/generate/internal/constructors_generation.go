package internal

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
)

func extractConstructorArguments(name string, args []actionField) ([]actionField, error) {
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

func GenerateConstructors(fn string) {
	s, err := parseConfig()
	if err != nil {
		panic(err)
	}

	cd := NewCoder("ride")
	cd.Import("github.com/pkg/errors")

	for _, act := range s.Actions {
		if !act.GenConstructor {
			continue
		}

		constructorName := act.Name + "Constructor"
		cd.Line("func %s(_ environment, args_ ...rideType) (rideType, error) {", constructorName)

		arguments, err := extractConstructorArguments(act.Name, act.Fields)
		if err != nil {
			panic(errors.Wrap(err, act.Name).Error())
		}

		cd.Line("if err := checkArgs(args_, %d); err != nil {", len(arguments))
		cd.Line("return nil, errors.Wrap(err, \"%s\")", constructorName)
		cd.Line("}")
		cd.Line("")

		for i, arg := range arguments {
			if len(arg.Types) == 1 {
				info := arg.Types[0]
				cd.Line("%s, ok := args_[%d].(%s)", arg.Name, i, info)
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, arg.Name, i)
				cd.Line("}")

				if listInfo, ok := info.(*listTypeInfo); ok {
					cd.Line("// checks for list elements")
					checkListElementsTypes(cd, constructorName, arg.Name, listInfo)
				}
			} else {
				cd.Line("var %s rideType", arg.Name)
				cd.Line("switch v := args_[%d].(type) {", i)
				for _, t := range arg.Types {
					cd.Line("case %s:", t)
					cd.Line("%s = v", arg.Name)
				}
				cd.Line("default:")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s' for %s\", args_[%d].instanceOf())", constructorName, arg.Name, i)
				cd.Line("}")
			}
			cd.Line("")
		}

		argsStr := make([]string, len(act.Fields))
		for i, field := range act.Fields {
			if field.ConstructorOrder == -1 {
				cd.Line("// default values for internal fields")
				cd.Line("var %s %s", field.Name, getType(field.Types))
			}
			argsStr[i] = field.Name
		}

		cd.Line("")
		cd.Line("return newRide%s(%s), nil", act.StructName, strings.Join(argsStr, ", "))
		cd.Line("}")
		cd.Line("")
	}

	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
