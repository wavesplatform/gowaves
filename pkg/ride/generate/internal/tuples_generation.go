package internal

import (
	"fmt"
	"strings"
)

func createTuples(cd *Coder) {
	for n := 2; n <= 22; n++ {
		name := fmt.Sprintf("tuple%d", n)
		elements := make([]string, 0, n)
		phs := make([]string, 0, n)
		instances := make([]string, 0, n)
		comparisons := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			elements = append(elements, fmt.Sprintf("el%d", i))
			phs = append(phs, "%s")
			instances = append(instances, fmt.Sprintf("a.el%d.instanceOf()", i))
			comparisons = append(comparisons, fmt.Sprintf("a.el%d.eq(o.el%d)", i, i))
		}
		cd.Line("type %s struct {", name)
		for _, el := range elements {
			cd.Line("%s rideType", el)
		}
		cd.Line("}")
		cd.Line("")
		cd.Line("func newTuple%d(_ environment, args ...rideType) (rideType, error) {", n)
		cd.Line("if len(args) != %d {", n)
		cd.Line("return nil, errors.New(\"invalid number of arguments\")")
		cd.Line("}")
		cd.Line("return %s{", name)
		for i, el := range elements {
			cd.Line("%s: args[%d],", el, i)
		}
		cd.Line("}, nil")
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) get(name string) (rideType, error) {", name)
		cd.Line("if !strings.HasPrefix(name, \"_\") {")
		cd.Line("return nil, errors.Errorf(\"%s has no element '%%s'\", name)", name)
		cd.Line("}")
		cd.Line("i, err := strconv.Atoi(strings.TrimPrefix(name, \"_\"))")
		cd.Line("if err != nil {")
		cd.Line("return nil, errors.Errorf(\"%s has no element '%%s'\", name)", name)
		cd.Line("}")
		cd.Line("switch i {")
		for i, el := range elements {
			cd.Line("case %d:", i+1)
			cd.Line("return a.%s, nil", el)
		}
		cd.Line("default:")
		cd.Line("return nil, errors.Errorf(\"%s has no element '%%s'\", name)", name)
		cd.Line("}")
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) instanceOf() string {", name)
		cd.Line("return fmt.Sprintf(\"(%s)\", %s)", strings.Join(phs, ", "), strings.Join(instances, ", "))
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) eq(other rideType) bool {", name)
		cd.Line("if a.instanceOf() != other.instanceOf() {")
		cd.Line("return false")
		cd.Line("}")
		cd.Line("o, ok := other.(%s)", name)
		cd.Line("if !ok {")
		cd.Line("return false")
		cd.Line("}")
		cd.Line("return %s", strings.Join(comparisons, " && "))
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) size() int {", name)
		cd.Line("return %d", n)
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) lines() []string {", name)
		cd.Line("return []string{a.String()}")
		cd.Line("}")
		cd.Line("")
		cd.Line("func (a %s) String() string {", name)
		cd.Line("sb := new(strings.Builder)")
		cd.Line("sb.WriteRune('(')")
		for i, el := range elements {
			cd.Line("sb.WriteString(a.%s.String())", el)
			if i < len(elements)-1 {
				cd.Line("sb.WriteRune(',')")
				cd.Line("sb.WriteRune(' ')")
			}
		}
		cd.Line("sb.WriteRune(')')")
		cd.Line("return sb.String()")
		cd.Line("}")
		cd.Line("")
	}
}

func GenerateTuples(fn string) {
	cd := NewCoder("ride")
	cd.Import("fmt")
	cd.Import("strconv")
	cd.Import("strings")
	cd.Import("github.com/pkg/errors")
	cd.Line("type rideTuple interface {")
	cd.Line("get(name string) (rideType, error)")
	cd.Line("size() int")
	cd.Line("}")
	cd.Line("")
	createTuples(cd)
	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
