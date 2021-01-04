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
	m["303"] = "takeString"
	m["304"] = "dropString"
	m["305"] = "sizeString"
	m["400"] = "sizeList"
	m["401"] = "getList"
	m["410"] = "intToBytes"
	m["411"] = "stringToBytes"
	m["412"] = "booleanToBytes"
	m["420"] = "intToString"
	m["421"] = "booleanToString"
	m["500"] = "sigVerify"
	m["501"] = "keccak256"
	m["502"] = "blake2b256"
	m["503"] = "sha256"
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
	m["getInteger"] = 30
	m["getBoolean"] = 30
	m["getBinary"] = 30
	m["getString"] = 30
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
	m["504"] = "rsaVerify"
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
	m["1201"] = "bytesToInt"
	m["1202"] = "bytesToIntWithOffset"
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
	m["@extrUser(addressFromString)"] = "addressValueFromString"
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
	// New constructors
	m["IntegerEntry"] = "checkedIntDataEntry"
	m["BooleanEntry"] = "checkedBooleanDataEntry"
	m["BinaryEntry"] = "checkedBytesDataEntry"
	m["StringEntry"] = "checkedStringDataEntry"
	m["DeleteEntry"] = "checkedDeleteEntry"
	m["Reissue"] = "reissue"
	m["Burn"] = "burn"
	m["SponsorFee"] = "sponsorship"

	// Functions
	delete(m, "wavesBalance") // Remove wavesBalanceV3
	m["contains"] = "contains"
	m["containsElement"] = "containsElement"
	m["valueOrElse"] = "valueOrElse"
	m["405"] = "median"
	m["406"] = "max"
	m["407"] = "min"
	delete(m, "700") // remove CheckMerkleProof
	m["701"] = "rebuildMerkleRoot"
	m["800"] = "bls12Groth16Verify"
	m["801"] = "bn256Groth16Verify"
	m["900"] = "ecRecover"
	delete(m, "1003") // Remove assetBalanceV3
	m["1004"] = "assetInfoV4"
	m["1007"] = "wavesBalanceV4"
	m["1008"] = "assetBalanceV4"
	m["1062"] = "addressFromString"
	m["1070"] = "transferFromProtobuf"
	m["1080"] = "calculateAssetID"
	m["1090"] = "simplifiedIssue"
	m["1091"] = "fullIssue"
	m["1100"] = "limitedCreateList"
	m["1101"] = "appendToList"
	m["1102"] = "concatList"
	m["1103"] = "indexOfList"
	m["1104"] = "lastIndexOfList"
	m["1209"] = "makeString"
	for i, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		m[strconv.Itoa(2400+i)] = fmt.Sprintf("bls12Groth16Verify_%d", l)
	}
	for i, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		m[strconv.Itoa(2450+i)] = fmt.Sprintf("bn256Groth16Verify_%d", l)
	}
	for i, l := range []int{8, 16, 32, 64, 128} {
		m[strconv.Itoa(2500+i)] = fmt.Sprintf("sigVerify_%d", l)
	}
	for i, l := range []int{16, 32, 64, 128} {
		m[strconv.Itoa(2600+i)] = fmt.Sprintf("rsaVerify_%d", l)
		m[strconv.Itoa(2700+i)] = fmt.Sprintf("keccak256_%d", l)
		m[strconv.Itoa(2800+i)] = fmt.Sprintf("blake2b256_%d", l)
		m[strconv.Itoa(2900+i)] = fmt.Sprintf("sha256_%d", l)
	}
	m["@extrNative(1062)"] = "addressFromString"
	for i := 2; i <= 22; i++ {
		m[strconv.Itoa(1300+i-2)] = fmt.Sprintf("newTuple%d", i)
	}
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
	m["Reissue"] = 3
	m["Burn"] = 2
	m["SponsorFee"] = 2

	m["201"] = 6
	m["202"] = 6
	m["203"] = 2
	m["300"] = 20
	m["303"] = 20
	m["304"] = 20
	m["405"] = 20
	m["406"] = 3
	m["407"] = 3
	m["411"] = 8
	m["500"] = 200
	m["501"] = 200
	m["502"] = 200
	m["503"] = 200
	m["504"] = 1000
	m["600"] = 3
	m["601"] = 1
	m["602"] = 35
	m["603"] = 40
	m["701"] = 30
	m["800"] = 2700 // BLS12
	m["801"] = 1650 // BN256
	m["900"] = 70
	m["1001"] = 20
	m["1004"] = 15
	m["1005"] = 5
	m["1006"] = 60
	m["1007"] = 10
	m["1008"] = 10
	m["1050"] = 10
	m["1051"] = 10
	m["1052"] = 10
	m["1053"] = 10
	m["1060"] = 5
	m["1062"] = 1
	m["1070"] = 5
	m["1080"] = 10
	m["1090"] = 1
	m["1091"] = 1
	m["1100"] = 1
	m["1101"] = 1
	m["1102"] = 4
	m["1103"] = 5
	m["1104"] = 5
	m["1200"] = 7
	m["1201"] = 1
	m["1202"] = 1
	m["1203"] = 3
	m["1204"] = 3
	m["1205"] = 75
	m["1206"] = 2
	m["1207"] = 3
	m["1208"] = 3
	m["1209"] = 30
	for i, c := range []int{1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000, 2100, 2200, 2300, 2400, 2500, 2600} {
		m[strconv.Itoa(2400+i)] = c
	}
	for i, c := range []int{800, 850, 950, 1000, 1050, 1100, 1150, 1200, 1250, 1300, 1350, 1400, 1450, 1550, 1600} {
		m[strconv.Itoa(2450+i)] = c
	}
	for i, c := range []int{47, 57, 70, 102, 172} {
		m[strconv.Itoa(2500+i)] = c
	}
	for i, c := range []int{500, 550, 625, 750} {
		m[strconv.Itoa(2600+i)] = c
	}
	for i, c := range []int{10, 25, 50, 100} {
		m[strconv.Itoa(2700+i)] = c
		m[strconv.Itoa(2800+i)] = c
		m[strconv.Itoa(2900+i)] = c
	}

	m["@extrNative(1050)"] = 10
	m["@extrNative(1051)"] = 10
	m["@extrNative(1052)"] = 10
	m["@extrNative(1053)"] = 10
	m["@extrNative(1062)"] = 1

	m["contains"] = 3
	m["containsElement"] = 5
	m["value"] = 2
	m["valueOrElse"] = 2
	m["valueOrErrorMessage"] = 2
	m["addressFromPublicKey"] = 63
	m["dropRightBytes"] = 6
	m["takeRightBytes"] = 6 // For bytes, also takeRight(bytes) should be 6
	m["takeRight"] = 20     // For strings
	m["dropRight"] = 20     // For strings
	m["drop"] = 6
	m["take"] = 20
	m["cons"] = 1
	m["parseIntValue"] = 2
	//Tuple constructors
	for i := 2; i <= 22; i++ {
		m[strconv.Itoa(1300+i-2)] = 1
	}

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

	sb.WriteString(fmt.Sprintf("var Constants%s = []string{", ver))
	for i, k := range keys {
		sb.WriteString(fmt.Sprintf("\"%s\"", k))
		if i < len(keys)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}\n")
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
			sb.WriteString(fmt.Sprintf("func create%s(env RideEnvironment, args ...rideType) (rideType, error) {\n", tn))
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

	sb.WriteString(fmt.Sprintf("var Catalogue%s = map[string]int{", ver))
	for i, k := range keys {
		sb.WriteString(fmt.Sprintf("\"%s\":%d", k, c[k]))
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

func createTuples(sb *strings.Builder) {
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
		sb.WriteString(fmt.Sprintf("type %s struct {\n", name))
		for _, el := range elements {
			sb.WriteString(fmt.Sprintf("%s rideType\n", el))
		}
		sb.WriteString("}\n\n")
		sb.WriteString(fmt.Sprintf("func newTuple%d(_ RideEnvironment, args ...rideType) (rideType, error) {\n", n))
		sb.WriteString(fmt.Sprintf("if len(args) != %d {\n", n))
		sb.WriteString("return nil, errors.New(\"invalid number of arguments\")\n")
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("return %s{\n", name))
		for i, el := range elements {
			sb.WriteString(fmt.Sprintf("%s: args[%d],\n", el, i))
		}
		sb.WriteString("}, nil\n")
		sb.WriteString("}\n\n")
		sb.WriteString(fmt.Sprintf("func (a %s) get(name string) (rideType, error) {\n", name))
		sb.WriteString("if !strings.HasPrefix(name, \"_\") {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s has no element '%%s'\", name)\n", name))
		sb.WriteString("}\n")
		sb.WriteString("i, err := strconv.Atoi(strings.TrimPrefix(name, \"_\"))\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s has no element '%%s'\", name)\n", name))
		sb.WriteString("}\n")
		sb.WriteString("switch i {\n")
		for i, el := range elements {
			sb.WriteString(fmt.Sprintf("case %d:\n", i+1))
			sb.WriteString(fmt.Sprintf("return a.%s, nil\n", el))
		}
		sb.WriteString("default:\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s has no element '%%s'\", name)\n", name))
		sb.WriteString("}\n")
		sb.WriteString("}\n\n")
		sb.WriteString(fmt.Sprintf("func (a %s) instanceOf() string {\n", name))
		sb.WriteString(fmt.Sprintf("return fmt.Sprintf(\"(%s)\", %s)\n", strings.Join(phs, ", "), strings.Join(instances, ", ")))
		sb.WriteString("}\n\n")
		sb.WriteString(fmt.Sprintf("func (a %s) eq(other rideType) bool {\n", name))
		sb.WriteString("if a.instanceOf() != other.instanceOf() {\n")
		sb.WriteString("return false\n")
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("o, ok := other.(%s)\n", name))
		sb.WriteString("if !ok {\n")
		sb.WriteString("return false\n")
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("return %s\n", strings.Join(comparisons, " && ")))
		sb.WriteString("}\n\n")
	}
}

func main() {
	sb := new(strings.Builder)
	sb.WriteString("// Code generated by ride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package ride\n")
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
	sb.WriteString("// Code generated by ride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package ride\n")
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
	sb.WriteString("// Code generated by ride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package ride\n")
	sb.WriteString("import (")
	sb.WriteString("\"crypto/rsa\"\n")
	sb.WriteString("sh256 \"crypto/sha256\"\n")
	sb.WriteString("\"crypto/x509\"\n")
	sb.WriteString("\"github.com/pkg/errors\"\n")
	sb.WriteString("\"github.com/wavesplatform/gowaves/pkg/crypto\"\n")
	sb.WriteString("c2 \"github.com/wavesplatform/gowaves/pkg/ride/crypto\"\n")
	sb.WriteString(")\n")
	for _, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		fn := fmt.Sprintf("bls12Groth16Verify_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(env RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("if err := checkArgs(args, 3); err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("key, ok := args[0].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[0].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("proof, ok := args[1].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[1].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("inputs, ok := args[2].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[2].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(inputs); l > 32*%d {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid inputs size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("ok, err := crypto.Bls12381{}.Groth16Verify(key, proof, inputs)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString("return rideUnit{}, err\n")
		sb.WriteString("}\n")
		sb.WriteString("return rideBoolean(ok), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		fn := fmt.Sprintf("bn256Groth16Verify_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(env RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("if err := checkArgs(args, 3); err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("key, ok := args[0].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[0].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("proof, ok := args[1].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[1].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("inputs, ok := args[2].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[2].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(inputs); l > 32*%d {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid inputs size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("ok, err := crypto.Bn256{}.Groth16Verify(key, proof, inputs)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString("return rideUnit{}, err\n")
		sb.WriteString("}\n")
		sb.WriteString("return rideBoolean(ok), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{8, 16, 32, 64, 128} {
		fn := fmt.Sprintf("sigVerify_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(env RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("if err := checkArgs(args, 3); err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("message, ok := args[0].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[0].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(message); l > %d*1024 {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid message size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("signature, ok := args[1].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[1].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("pkb, ok := args[2].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[2].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("pk, err := crypto.NewPublicKeyFromBytes(pkb)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString("return rideBoolean(false), nil\n")
		sb.WriteString("}\n")
		sb.WriteString("sig, err := crypto.NewSignatureFromBytes(signature)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString("return rideBoolean(false), nil\n")
		sb.WriteString("}\n")
		sb.WriteString("ok = crypto.Verify(pk, sig, message)\n")
		sb.WriteString("return rideBoolean(ok), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		fn := fmt.Sprintf("rsaVerify_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(_ RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("if err := checkArgs(args, 4); err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("digest, err := digest(args[0])\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("message, ok := args[1].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[1].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(message); l > %d*1024 {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid message size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("sig, ok := args[2].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[2].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("pk, ok := args[3].(rideBytes)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: unexpected argument type '%%s'\", args[3].instanceOf())\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("key, err := x509.ParsePKIXPublicKey(pk)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s: invalid public key\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("k, ok := key.(*rsa.PublicKey)\n")
		sb.WriteString("if !ok {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.New(\"%s: not an RSA key\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("d := message\n")
		sb.WriteString("if digest != 0 {\n")
		sb.WriteString("h := digest.New()\n")
		sb.WriteString("_, _ = h.Write(message)\n")
		sb.WriteString("d = h.Sum(nil)\n")
		sb.WriteString("}\n")
		sb.WriteString("ok, err = c2.VerifyPKCS1v15(k, digest, d, sig)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("return rideBoolean(ok), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		fn := fmt.Sprintf("keccak256_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(env RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("data, err := bytesOrStringArg(args)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(data); l > %d*1024 {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid data size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("d, err := crypto.Keccak256(data)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("return rideBytes(d.Bytes()), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		fn := fmt.Sprintf("blake2b256_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(_ RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("data, err := bytesOrStringArg(args)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(data); l > %d*1024 {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid data size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("d, err := crypto.FastHash(data)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("return rideBytes(d.Bytes()), nil\n")
		sb.WriteString("}\n\n")
	}
	for _, l := range []int{16, 32, 64, 128} {
		fn := fmt.Sprintf("sha256_%d", l)
		sb.WriteString(fmt.Sprintf("func %s(_ RideEnvironment, args ...rideType) (rideType, error) {\n", fn))
		sb.WriteString("data, err := bytesOrStringArg(args)\n")
		sb.WriteString("if err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString(fmt.Sprintf("if l := len(data); l > %d*1024 {\n", l))
		sb.WriteString(fmt.Sprintf("return nil, errors.Errorf(\"%s: invalid data size %%d\", l)\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("h := sh256.New()\n")
		sb.WriteString("if _, err = h.Write(data); err != nil {\n")
		sb.WriteString(fmt.Sprintf("return nil, errors.Wrap(err, \"%s\")\n", fn))
		sb.WriteString("}\n")
		sb.WriteString("d := h.Sum(nil)\n")
		sb.WriteString("return rideBytes(d), nil\n")
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

	sb = new(strings.Builder)
	sb.WriteString("// Code generated by ride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package ride\n")
	sb.WriteString("import (\n")
	sb.WriteString("\"fmt\"\n")
	sb.WriteString("\"strconv\"\n")
	sb.WriteString("\"strings\"\n")
	sb.WriteString("\"github.com/pkg/errors\"\n")
	sb.WriteString(")\n")
	createTuples(sb)
	code = sb.String()
	b, err = format.Source([]byte(code))
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("tuples.go", b, 0644)
	if err != nil {
		panic(err)
	}
}
