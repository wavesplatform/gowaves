package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
)

func functionsV2() map[string]string {
	m := make(map[string]string)
	m["0"] = "eq"
	m["1"] = "instanceOf"
	m["2"] = "throw"
	m["100"] = "sum"
	m["101"] = "sub"
	m["102"] = "gt"
	m["103"] = "ge"
	m["104"] = "mul"
	m["105"] = "div"
	m["106"] = "mod"
	m["107"] = "fraction"
	m["200"] = "sizeBytes"
	m["201"] = "takeBytes"
	m["202"] = "dropBytes"
	m["203"] = "concatBytes"
	m["300"] = "concatStrings"
	m["303"] = "takeStrings"
	m["304"] = "dropStrings"
	m["305"] = "sizeStrings"
	m["400"] = "sizeList"
	m["401"] = "getList"
	m["410"] = "intToBytes"
	m["411"] = "stringToBytes"
	m["412"] = "booleanToBytes"
	m["420"] = "intToString"
	m["421"] = "booleanToString"
	m["500"] = "unlimitedSigVerify"
	m["501"] = "unlimitedKeccak256"
	m["502"] = "unlimitedBlake2b256"
	m["503"] = "unlimitedSha256"
	m["600"] = "toBase58"
	m["601"] = "fromBase58"
	m["602"] = "toBase64"
	m["603"] = "fromBase64"
	m["1000"] = "transactionByID"
	m["1001"] = "transactionHeightByID"
	m["1003"] = "assetBalanceV3"
	m["1040"] = "intFromArray"
	m["1041"] = "booleanFromArray"
	m["1042"] = "bytesFromArray"
	m["1043"] = "stringFromArray"
	m["1050"] = "intFromState"
	m["1051"] = "booleanFromState"
	m["1052"] = "bytesFromState"
	m["1053"] = "stringFromState"
	m["1060"] = "addressFromRecipient"
	m["throw"] = "throw0"
	m["addressFromString"] = "addressFromString"
	m["!="] = "neq"
	m["isDefined"] = "isDefined"
	m["extract"] = "extract"
	m["dropRightBytes"] = "dropRightBytes"
	m["takeRightBytes"] = "takeRightBytes"
	m["takeRight"] = "takeRightString"
	m["dropRight"] = "dropRightString"
	m["!"] = "unaryNot"
	m["-"] = "unaryMinus"
	m["getInteger"] = "intFromArrayByIndex"
	m["getBoolean"] = "booleanFromArrayByIndex"
	m["getBinary"] = "bytesFromArrayByIndex"
	m["getString"] = "stringFromArrayByIndex"
	m["addressFromPublicKey"] = "addressFromPublicKey"
	m["wavesBalance"] = "wavesBalanceV3"
	m["Address"] = "address"
	m["Alias"] = "alias"
	m["DataEntry"] = "dataEntry"
	m["AssetPair"] = "assetPair"
	m["DataTransaction"] = "dataTransaction"
	return m
}

func catalogueV2() map[string]int {
	m := make(map[string]int)
	m["0"] = 1
	m["1"] = 1
	m["2"] = 1
	m["100"] = 1
	m["101"] = 1
	m["102"] = 1
	m["103"] = 1
	m["104"] = 1
	m["105"] = 1
	m["106"] = 1
	m["107"] = 1
	m["200"] = 1
	m["201"] = 1
	m["202"] = 1
	m["203"] = 10
	m["300"] = 10
	m["303"] = 1
	m["304"] = 1
	m["305"] = 1
	m["400"] = 2
	m["401"] = 2
	m["410"] = 1
	m["411"] = 1
	m["412"] = 1
	m["420"] = 1
	m["421"] = 1
	m["500"] = 100
	m["501"] = 10
	m["502"] = 10
	m["503"] = 10
	m["600"] = 10
	m["601"] = 10
	m["602"] = 10
	m["603"] = 10
	m["1000"] = 100
	m["1001"] = 100
	m["1003"] = 100
	m["1040"] = 10
	m["1041"] = 10
	m["1042"] = 10
	m["1043"] = 10
	m["1050"] = 100
	m["1051"] = 100
	m["1052"] = 100
	m["1053"] = 100
	m["1060"] = 100
	m["throw"] = 2
	m["addressFromString"] = 124
	m["!="] = 26
	m["isDefined"] = 35
	m["extract"] = 13
	m["dropRightBytes"] = 19
	m["takeRightBytes"] = 19
	m["takeRight"] = 19
	m["dropRight"] = 19
	m["!"] = 11
	m["-"] = 9
	m["getInteger"] = 10
	m["getBoolean"] = 10
	m["getBinary"] = 10
	m["getString"] = 10
	m["addressFromPublicKey"] = 82
	m["wavesBalance"] = 109
	m["Address"] = 1
	m["Alias"] = 1
	m["DataEntry"] = 2
	m["DataTransaction"] = 9
	m["AssetPair"] = 2
	return m
}

func functionsV3() map[string]string {
	m := functionsV2()
	m["108"] = "pow"
	m["109"] = "log"
	m["504"] = "unlimitedRSAVerify"
	m["604"] = "toBase16"
	m["605"] = "fromBase16"
	m["700"] = "checkMerkleProof"
	delete(m, "1000") // Native function transactionByID was disabled since v3
	m["1004"] = "assetInfoV3"
	m["1005"] = "blockInfoByHeight"
	m["1006"] = "transferByID"
	m["1061"] = "addressToString"
	m["1100"] = "createList"
	m["1200"] = "bytesToUTF8String"
	m["1201"] = "bytesToLong"
	m["1202"] = "bytesToLongWithOffset"
	m["1203"] = "indexOfSubstring"
	m["1204"] = "indexOfSubstringWithOffset"
	m["1205"] = "splitString"
	m["1206"] = "parseInt"
	m["1207"] = "lastIndexOfSubstring"
	m["1208"] = "lastIndexOfSubstringWithOffset"

	// Constructors for simple types
	constructorsFromConstants(m, constantsV3())

	m["Unit"] = "unit"

	// New user functions
	m["@extrNative(1050)"] = "intValueFromState"
	m["@extrNative(1051)"] = "booleanValueFromState"
	m["@extrNative(1052)"] = "bytesValueFromState"
	m["@extrNative(1053)"] = "stringValueFromState"
	m["@extrNative(1040)"] = "intValueFromArray"
	m["@extrNative(1041)"] = "booleanValueFromArray"
	m["@extrNative(1042)"] = "bytesValueFromArray"
	m["@extrNative(1043)"] = "stringValueFromArray"
	m["@extrUser(getInteger)"] = "intValueFromArrayByIndex"
	m["@extrUser(getBoolean)"] = "booleanValueFromArrayByIndex"
	m["@extrUser(getBinary)"] = "bytesValueFromArrayByIndex"
	m["@extrUser(getString)"] = "stringValueFromArrayByIndex"
	m["@extrUser(addressFromString)"] = "addressFromString"
	m["parseIntValue"] = "parseIntValue"
	m["value"] = "value"
	m["valueOrErrorMessage"] = "valueOrErrorMessage"
	m["WriteSet"] = "writeSet"
	m["TransferSet"] = "transferSet"
	m["ScriptTransfer"] = "scriptTransfer"
	m["ScriptResult"] = "scriptResult"
	return m
}

func catalogueV3() map[string]int {
	m := catalogueV2()
	m["108"] = 100
	m["109"] = 100
	m["504"] = 300
	m["604"] = 10
	m["605"] = 10
	m["700"] = 30
	delete(m, "1000")
	m["1004"] = 100
	m["1005"] = 100
	m["1006"] = 100
	m["1061"] = 10
	m["1070"] = 100
	m["1100"] = 2
	m["1200"] = 20
	m["1201"] = 10
	m["1202"] = 10
	m["1203"] = 20
	m["1204"] = 20
	m["1205"] = 100
	m["1206"] = 20
	m["1207"] = 20
	m["1208"] = 20
	m["throw"] = 1
	m["isDefined"] = 1
	m["!="] = 1
	m["!"] = 1
	m["-"] = 1
	m["Ceiling"] = 0
	m["Floor"] = 0
	m["HalfEven"] = 0
	m["Down"] = 0
	m["Up"] = 0
	m["HalfUp"] = 0
	m["HalfDown"] = 0
	m["NoAlg"] = 0
	m["Md5"] = 0
	m["Sha1"] = 0
	m["Sha224"] = 0
	m["Sha256"] = 0
	m["Sha384"] = 0
	m["Sha512"] = 0
	m["Sha3224"] = 0
	m["Sha3256"] = 0
	m["Sha3384"] = 0
	m["Sha3512"] = 0
	m["Unit"] = 0
	m["@extrNative(1040)"] = 10
	m["@extrNative(1041)"] = 10
	m["@extrNative(1042)"] = 10
	m["@extrNative(1043)"] = 10
	m["@extrNative(1050)"] = 100
	m["@extrNative(1051)"] = 100
	m["@extrNative(1052)"] = 100
	m["@extrNative(1053)"] = 100
	m["@extrUser(getInteger)"] = 10
	m["@extrUser(getBoolean)"] = 10
	m["@extrUser(getBinary)"] = 10
	m["@extrUser(getString)"] = 10
	m["@extrUser(addressFromString)"] = 124
	m["parseIntValue"] = 20
	m["value"] = 13
	m["valueOrErrorMessage"] = 13
	m["WriteSet"] = 1
	m["TransferSet"] = 1
	m["ScriptTransfer"] = 3
	m["ScriptResult"] = 2
	return m
}

func functionsV4() map[string]string {
	m := functionsV3()
	// Remove obsolete constructors
	delete(m, "ScriptResult")
	delete(m, "WriteSet")
	delete(m, "TransferSet")
	delete(m, "DataEntry")
	// Replace functions
	m["wavesBalance"] = "wavesBalanceV4"
	m["1003"] = "assetBalanceV4"
	m["1004"] = "assetInfoV4"
	// New constructors
	m["IntegerEntry"] = "checkedIntDataEntry"
	m["BooleanEntry"] = "checkedBooleanDataEntry"
	m["BinaryEntry"] = "checkedBytesDataEntry"
	m["StringEntry"] = "checkedStringDataEntry"
	m["DeleteEntry"] = "checkedDeleteEntry"
	//TODO: remove Issue constructor after updating test script in pkg/state/testdata/scripts/ride4_asset.base64
	m["Issue"] = "issue"
	m["Reissue"] = "reissue"
	m["Burn"] = "burn"
	m["SponsorFee"] = "sponsorship"

	// New functions
	m["contains"] = "contains"
	m["valueOrElse"] = "valueOrElse"
	m["1080"] = "calculateAssetID"
	m["1101"] = "appendToList"
	m["1102"] = "concatList"
	m["1103"] = "indexOfList"
	m["1104"] = "lastIndexOfList"
	m["405"] = "median"
	m["406"] = "max"
	m["407"] = "min"
	m["1100"] = "limitedCreateList"
	m["800"] = "unlimitedGroth16Verify"
	m["900"] = "ecRecover"
	for i, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		m[strconv.Itoa(2400+i)] = fmt.Sprintf("limitedGroth16Verify_%d", l)
	}
	for i, l := range []int{16, 32, 64, 128} {
		m[strconv.Itoa(2500+i)] = fmt.Sprintf("sigVerify_%d", l)
		m[strconv.Itoa(2600+i)] = fmt.Sprintf("rsaVerify_%d", l)
		m[strconv.Itoa(2700+i)] = fmt.Sprintf("keccak256_%d", l)
		m[strconv.Itoa(2800+i)] = fmt.Sprintf("blake2b256_%d", l)
		m[strconv.Itoa(2900+i)] = fmt.Sprintf("sha256_%d", l)
	}
	m["1070"] = "transferFromProtobuf"
	delete(m, "700") // remove CheckMerkleProof
	m["701"] = "rebuildMerkleRoot"
	m["1090"] = "simplifiedIssue"
	m["1091"] = "fullIssue"
	return m
}

func catalogueV4() map[string]int {
	m := catalogueV3()
	delete(m, "ScriptResult")
	delete(m, "WriteSet")
	delete(m, "TransferSet")
	delete(m, "DataEntry")
	m["IntegerEntry"] = 2
	m["BooleanEntry"] = 2
	m["BinaryEntry"] = 2
	m["StringEntry"] = 2
	m["DeleteEntry"] = 1
	m["Issue"] = 7
	m["Reissue"] = 3
	m["Burn"] = 2
	m["contains"] = 20
	m["valueOrElse"] = 13
	m["405"] = 10
	m["406"] = 3
	m["407"] = 3
	m["701"] = 30
	m["800"] = 3900
	m["900"] = 70
	m["1070"] = 5
	m["1080"] = 10
	m["1100"] = 2
	m["1101"] = 3
	m["1102"] = 10
	m["1103"] = 5
	m["1104"] = 5
	m["2400"] = 1900
	m["2401"] = 2000
	m["2402"] = 2150
	m["2403"] = 2300
	m["2404"] = 2450
	m["2405"] = 2550
	m["2406"] = 2700
	m["2407"] = 2900
	m["2408"] = 3000
	m["2409"] = 3150
	m["2410"] = 3250
	m["2411"] = 3400
	m["2412"] = 3500
	m["2413"] = 3650
	m["2414"] = 3750
	m["2500"] = 100
	m["2501"] = 110
	m["2502"] = 125
	m["2503"] = 150
	m["2600"] = 100
	m["2601"] = 500
	m["2602"] = 550
	m["2603"] = 625
	m["2700"] = 10
	m["2701"] = 25
	m["2702"] = 50
	m["2703"] = 100
	m["2800"] = 10
	m["2801"] = 25
	m["2802"] = 50
	m["2803"] = 100
	m["2900"] = 10
	m["2901"] = 25
	m["2902"] = 50
	m["2903"] = 100
	return m
}

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

func constructorsFromConstants(m map[string]string, c map[string]constantDescription) {
	for _, v := range c {
		if v.constructor == "" {
			m[v.typeName] = fmt.Sprintf("create%s", v.typeName)
		}
	}
}
func createConstants(sb *strings.Builder, ver string, c map[string]constantDescription) {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sb.WriteString(fmt.Sprintf("const _constants_%s = \"%s\"\n", ver, strings.Join(keys, "")))
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		if c[k].constructor == "" {
			m[k] = fmt.Sprintf("new%s", c[k].typeName)
		} else {
			m[k] = c[k].constructor
		}
	}
	sb.WriteString(fmt.Sprintf("var _constructors_%s = [...]rideConstructor{", ver))
	for i, k := range keys {
		sb.WriteString(m[k])
		if i < len(m)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n")
	idx := 0
	sb.WriteString(fmt.Sprintf("var _c_index_%s = [...]int{%d, ", ver, idx))
	for i, k := range keys {
		idx += len(k)
		sb.WriteString(fmt.Sprintf("%d", idx))
		if i < len(keys)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("func constant%s(id int) rideConstructor {\n", ver))
	sb.WriteString(fmt.Sprintf("if id < 0 || id > %d {\n", len(keys)-1))
	sb.WriteString("return nil\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("return _constructors_%s[id]\n}\n", ver))
	sb.WriteString(fmt.Sprintf("func checkConstant%s(name string) (uint16, bool) {\n", ver))
	sb.WriteString(fmt.Sprintf("for i := 0; i <= %d; i++ {\n", len(keys)-1))
	sb.WriteString(fmt.Sprintf("if _constants_%s[_c_index_%s[i]:_c_index_%s[i+1]] == name {\n", ver, ver, ver))
	sb.WriteString("return uint16(i), true\n")
	sb.WriteString("}\n}\n")
	sb.WriteString("return 0, false\n")
	sb.WriteString("}\n\n")
}

func createConstructors(sb *strings.Builder, c map[string]constantDescription) {
	for _, v := range c {
		if v.constructor == "" {
			tn := v.typeName
			sb.WriteString(fmt.Sprintf("func new%s(RideEnvironment) rideType {\n", tn))
			sb.WriteString(fmt.Sprintf("return rideNamedType{name: \"%s\"}\n", tn))
			sb.WriteString("}\n\n")
			sb.WriteString(fmt.Sprintf("func create%s(...rideType) (rideType, error) {\n", tn))
			sb.WriteString(fmt.Sprintf("return rideNamedType{name: \"%s\"}, nil\n", tn))
			sb.WriteString("}\n\n")
		}
	}
}

func createFunctionsList(sb *strings.Builder, ver string, m map[string]string, c map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create sorted list of functions
	sb.WriteString(fmt.Sprintf("var _functions_%s = [...]rideFunction{", ver))
	for i, k := range keys {
		sb.WriteString(m[k])
		if i < len(m)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n")

	// Create list of costs
	sb.WriteString(fmt.Sprintf("var _catalogue_%s = [...]int{", ver))
	for i, k := range keys {
		sb.WriteString(strconv.Itoa(c[k]))
		if i < len(m)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n")

	// Create string of concatenated names of functions
	sb.WriteString(fmt.Sprintf("const _names_%s = \"%s\"\n", ver, strings.Join(keys, "")))

	// Create indexes for names extraction
	idx := 0
	sb.WriteString(fmt.Sprintf("var _index_%s = [...]int{%d, ", ver, idx))
	for i, k := range keys {
		idx += len(k)
		sb.WriteString(fmt.Sprintf("%d", idx))
		if i < len(keys)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n\n")
	sb.WriteString(fmt.Sprintf("func functionName%s(i int) string {\n", ver))
	sb.WriteString(fmt.Sprintf("if i < 0 || i > %d {\n", len(keys)-1))
	sb.WriteString("return \"\"\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("return _names_%s[_index_%s[i]:_index_%s[i+1]]\n}\n", ver, ver, ver))
	sb.WriteString(fmt.Sprintf("func function%s(id int) rideFunction {\n", ver))
	sb.WriteString(fmt.Sprintf("if id < 0 || id > %d {\n", len(keys)-1))
	sb.WriteString("return nil\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("return _functions_%s[id]\n}\n", ver))
	sb.WriteString(fmt.Sprintf("func checkFunction%s(name string) (uint16, bool) {\n", ver))
	sb.WriteString(fmt.Sprintf("for i := 0; i <= %d; i++ {\n", len(keys)-1))
	sb.WriteString(fmt.Sprintf("if _names_%s[_index_%s[i]:_index_%s[i+1]] == name {\n", ver, ver, ver))
	sb.WriteString("return uint16(i), true\n")
	sb.WriteString("}\n}\n")
	sb.WriteString("return 0, false\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("func cost%s(id int) int {\n", ver))
	sb.WriteString(fmt.Sprintf("if id < 0 || id > %d {\n", len(keys)-1))
	sb.WriteString("return -1\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("return _catalogue_%s[id]\n}\n", ver))
}

func main() {
	sb := new(strings.Builder)
	sb.WriteString("// Code generated by fride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package fride\n")
	createFunctionsList(sb, "V2", functionsV2(), catalogueV2())
	createFunctionsList(sb, "V3", functionsV3(), catalogueV3())
	createFunctionsList(sb, "V4", functionsV4(), catalogueV4())
	code := sb.String()
	b, err := format.Source([]byte(code))
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("functions.go", b, 0644)
	if err != nil {
		panic(err)
	}

	sb = new(strings.Builder)
	sb.WriteString("// Code generated by fride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package fride\n")
	createConstants(sb, "V1", constantsV1())
	createConstants(sb, "V2", constantsV2())
	createConstants(sb, "V3", constantsV3())
	createConstants(sb, "V4", constantsV4())
	createConstructors(sb, constantsV4())
	code = sb.String()
	b, err = format.Source([]byte(code))
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("constants.go", b, 0644)
	if err != nil {
		panic(err)
	}

	sb = new(strings.Builder)
	sb.WriteString("// Code generated by fride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package fride\n")
	sb.WriteString("import (")
	sb.WriteString("\"github.com/pkg/errors\"\n")
	sb.WriteString(")\n")
	for _, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		sb.WriteString(fmt.Sprintf("func limitedGroth16Verify_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		sb.WriteString(fmt.Sprintf("func sigVerify_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		sb.WriteString(fmt.Sprintf("func rsaVerify_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		sb.WriteString(fmt.Sprintf("func keccak256_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		sb.WriteString(fmt.Sprintf("func blake2b256_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		sb.WriteString(fmt.Sprintf("func sha256_%d(...rideType) (rideType, error) {\n", l))
		sb.WriteString("return nil, errors.New(\"not implemented\")\n")
		sb.WriteString("}\n\n")
	}
	code = sb.String()
	b, err = format.Source([]byte(code))
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("functions_generated.go", b, 0644)
	if err != nil {
		panic(err)
	}
}
