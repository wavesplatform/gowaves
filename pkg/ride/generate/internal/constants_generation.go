package internal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type constantDescription struct {
	typeName    string
	constructor string
}

func constantsV1() map[string]constantDescription {
	return map[string]constantDescription{
		"tx":     {constructor: "newTx"},
		"unit":   {constructor: "newUnit"},
		"height": {constructor: "newHeight"},
	}
}

func constantsV2() map[string]constantDescription {
	c := constantsV1()
	c["Sell"] = constantDescription{"Sell", ""}
	c["Buy"] = constantDescription{"Buy", ""}

	c["CEILING"] = constantDescription{"Ceiling", ""}
	c["FLOOR"] = constantDescription{"Floor", ""}
	c["HALFEVEN"] = constantDescription{"HalfEven", ""}
	c["DOWN"] = constantDescription{"Down", ""}
	c["UP"] = constantDescription{"Up", ""}
	c["HALFUP"] = constantDescription{"HalfUp", ""}
	c["HALFDOWN"] = constantDescription{"HalfDown", ""}

	c["nil"] = constantDescription{constructor: "newNil"}
	return c
}

func constantsV3() map[string]constantDescription {
	c := constantsV2()
	c["NOALG"] = constantDescription{"NoAlg", ""}
	c["MD5"] = constantDescription{"Md5", ""}
	c["SHA1"] = constantDescription{"Sha1", ""}
	c["SHA224"] = constantDescription{"Sha224", ""}
	c["SHA256"] = constantDescription{"Sha256", ""}
	c["SHA384"] = constantDescription{"Sha384", ""}
	c["SHA512"] = constantDescription{"Sha512", ""}
	c["SHA3224"] = constantDescription{"Sha3224", ""}
	c["SHA3256"] = constantDescription{"Sha3256", ""}
	c["SHA3384"] = constantDescription{"Sha3384", ""}
	c["SHA3512"] = constantDescription{"Sha3512", ""}

	c["this"] = constantDescription{constructor: "newThis"}
	c["lastBlock"] = constantDescription{constructor: "newLastBlock"}
	return c
}

func constantsV4() map[string]constantDescription {
	return constantsV3()
}

func constantsV5() map[string]constantDescription {
	c := constantsV4()
	delete(c, "UP")
	delete(c, "HALFDOWN")
	return c
}

func constantsV6() map[string]constantDescription {
	return constantsV5()
}

func createConstants(cd *Coder, ver string, c map[string]constantDescription) {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	qks := make([]string, len(keys))
	for i, k := range keys {
		qks[i] = strconv.Quote(k)
	}
	cd.Line("var Constants%s = []string{%s}", ver, strings.Join(qks, ", "))

	cd.Line("const _constants_%s = \"%s\"", ver, strings.Join(keys, ""))

	constructors := make([]string, len(keys))
	for i, k := range keys {
		if c[k].constructor == "" {
			constructors[i] = fmt.Sprintf("new%s", c[k].typeName)
		} else {
			constructors[i] = c[k].constructor
		}
	}
	cd.Line("var _constructors_%s = [...]rideConstructor{%s}", ver, strings.Join(constructors, ", "))

	idx := 0
	positions := make([]string, len(keys))
	for i, k := range keys {
		idx += len(k)
		positions[i] = strconv.Itoa(idx)
	}
	cd.Line("var _c_index_%s = [...]int{0, %s}", ver, strings.Join(positions, ", "))
	cd.Line("")

	cd.Line("func constant%s(id int) rideConstructor {", ver)
	cd.Line("if id < 0 || id > %d {", len(keys)-1)
	cd.Line("return nil")
	cd.Line("}")
	cd.Line("return _constructors_%s[id]", ver)
	cd.Line("}")
	cd.Line("")
	cd.Line("func checkConstant%s(name string) (uint16, bool) {", ver)
	cd.Line("for i := 0; i <= %d; i++ {", len(keys)-1)
	cd.Line("if _constants_%s[_c_index_%s[i]:_c_index_%s[i+1]] == name {", ver, ver, ver)
	cd.Line("return uint16(i), true")
	cd.Line("}")
	cd.Line("}")
	cd.Line("return 0, false")
	cd.Line("}")
	cd.Line("")
}

func createConstructors(cd *Coder, c map[string]constantDescription) {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if c[k].constructor == "" {
			tn := c[k].typeName
			cd.Line("func new%s(environment) rideType {", tn)
			cd.Line("return rideNamedType{name: \"%s\"}", tn)
			cd.Line("}")
			cd.Line("")
			cd.Line("func create%s(_ environment, _ ...rideType) (rideType, error) {", tn)
			cd.Line("return rideNamedType{name: \"%s\"}, nil", tn)
			cd.Line("}")
			cd.Line("")
		}
	}
}

func GenerateConstants(fn string) {
	cd := NewCoder("ride")

	createConstants(cd, "V1", constantsV1())
	createConstants(cd, "V2", constantsV2())
	createConstants(cd, "V3", constantsV3())
	createConstants(cd, "V4", constantsV4())
	createConstants(cd, "V5", constantsV5())
	createConstants(cd, "V6", constantsV6())
	createConstructors(cd, constantsV4())

	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
