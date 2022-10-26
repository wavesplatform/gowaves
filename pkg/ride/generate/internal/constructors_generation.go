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
	sort.Slice(arguments, func(i, j int) bool {
		return arguments[i].ConstructorOrder < arguments[j].ConstructorOrder
	})
	return arguments, nil
}

// TODO: FINISH IT TODAY
func checkListElementsTypes(cd *Coder, argument actionField, info *listTypeInfo) {

	cd.Line("for i, elem := range %s {", arg.Name)
	cd.Line("switch elem {")
	for _, tInfo := range info.ElementTypes() {
		cd.Line("case %s: ", tInfo.String())
		if lInfo, ok := tInfo.(*listTypeInfo); ok {
			checkListElementsTypes(cd, argument, lInfo)
		}
	}
	cd.Line("}")
	cd.Line("}")
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
		cd.Line("func %s(_ environment, args ...rideType) (rideType, error) {", constructorName)

		arguments, err := extractConstructorArguments(act.Name, act.Fields)
		if err != nil {
			panic(errors.Wrap(err, act.Name).Error())
		}

		cd.Line("if err := checkArgs(args, %d); err != nil {", len(arguments))
		cd.Line("return nil, errors.Wrap(err, \"%s\")", constructorName)
		cd.Line("}")
		cd.Line("")

		for i, arg := range arguments {
			if len(arg.Types) == 1 {
				info := arg.Types[0]
				cd.Line("%s, ok := args[%d].(%s)", arg.Name, i, info)
				cd.Line("if !ok {")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s'\", args[%d].instanceOf())", constructorName, i)
				cd.Line("}")

				// add checks for list elements
				if listInfo, ok := info.(*listTypeInfo); ok {
					checkListElementsTypes(cd, arg, listInfo)
				}
			} else {
				cd.Line("var %s rideType", arg.Name)
				cd.Line("switch v := args[%d].(type) {", i)
				for _, t := range arg.Types {
					cd.Line("case %s:", t)
					cd.Line("%s = v", arg.Name)
				}
				cd.Line("default:")
				cd.Line("return nil, errors.Errorf(\"%s: unexpected type '%%s'\", args[%d].instanceOf())", constructorName, i)
				cd.Line("}")
			}
			cd.Line("")
		}

		argsStr := make([]string, len(act.Fields))
		for i, field := range act.Fields {
			if field.ConstructorOrder == -1 {
				// generate default value
				cd.Line("var %s %s", field.Name, getType(field.Types))
			}
			argsStr[i] = field.Name
		}

		cd.Line("return newRide%s(%s), nil", act.StructName, strings.Join(argsStr, ", "))
		cd.Line("}")
		cd.Line("")
	}

	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
