package internal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
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
	constructorsFunctions(ast.LibV2, m)
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
	constructorsCatalogue(ast.LibV2, m)
	return m
}

func evaluationCatalogueV2EvaluatorV1() map[string]int {
	// In Scala implementation order
	m := catalogueV2()
	m["isDefined"] = 7
	m["!="] = 5
	m["-"] = 2
	m["!"] = 2
	m["takeRightBytes"] = 6
	m["dropRightBytes"] = 6
	m["dropRight"] = 6
	m["takeRight"] = 6
	m["extract"] = 5
	m["getInteger"] = 9
	m["getBoolean"] = 9
	m["getBinary"] = 9
	m["getString"] = 9
	m["addressFromString"] = 20
	m["addressFromPublicKey"] = 65
	m["wavesBalance"] = 102
	m["Address"] = 0
	m["Alias"] = 0
	constructorsEvaluationCatalogueEvaluatorV1(ast.LibV2, m)

	return m
}

func evaluationCatalogueV2EvaluatorV2() map[string]int {
	m := catalogueV2()
	m["Address"] = 1
	m["Alias"] = 1
	constructorsEvaluationCatalogueEvaluatorV2(ast.LibV2, m)
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

	constructorsFunctions(ast.LibV3, m)
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
	constructorsCatalogue(ast.LibV3, m)
	return m
}

func evaluationCatalogueV3EvaluatorV1() map[string]int {
	m := catalogueV3()
	m["isDefined"] = 7
	m["throw"] = 2
	m["!="] = 5
	m["-"] = 2
	m["!"] = 2
	m["takeRightBytes"] = 6
	m["dropRightBytes"] = 6
	m["dropRight"] = 6
	m["takeRight"] = 6
	m["extract"] = 5
	m["value"] = 5
	m["valueOrErrorMessage"] = 5
	m["parseIntValue"] = 26
	m["getInteger"] = 9
	m["getBoolean"] = 9
	m["getBinary"] = 9
	m["getString"] = 9
	m["addressFromString"] = 20
	m["addressFromPublicKey"] = 65
	m["@extrNative(1050)"] = 107
	m["@extrNative(1051)"] = 107
	m["@extrNative(1052)"] = 107
	m["@extrNative(1053)"] = 107
	m["@extrNative(1040)"] = 17
	m["@extrNative(1041)"] = 17
	m["@extrNative(1042)"] = 17
	m["@extrNative(1043)"] = 17
	m["@extrUser(getInteger)"] = 16
	m["@extrUser(getBoolean)"] = 16
	m["@extrUser(getBinary)"] = 16
	m["@extrUser(getString)"] = 16
	m["@extrUser(addressFromString)"] = 26
	m["wavesBalance"] = 102
	m["Address"] = 0
	m["Alias"] = 0
	constructorsEvaluationCatalogueEvaluatorV1(ast.LibV3, m)
	return m
}

func evaluationCatalogueV3EvaluatorV2() map[string]int {
	m := catalogueV3()
	m["throw"] = 2
	m["Ceiling"] = 1
	m["Floor"] = 1
	m["HalfEven"] = 1
	m["Down"] = 1
	m["Up"] = 1
	m["HalfUp"] = 1
	m["HalfDown"] = 1
	m["NoAlg"] = 1
	m["Md5"] = 1
	m["Sha1"] = 1
	m["Sha224"] = 1
	m["Sha256"] = 1
	m["Sha384"] = 1
	m["Sha512"] = 1
	m["Sha3224"] = 1
	m["Sha3256"] = 1
	m["Sha3384"] = 1
	m["Sha3512"] = 1
	m["Unit"] = 1
	m["Address"] = 1
	m["Alias"] = 1
	constructorsEvaluationCatalogueEvaluatorV2(ast.LibV3, m)
	return m
}

func functionsV4() map[string]string {
	m := functionsV3()

	// Functions
	delete(m, "extract")
	delete(m, "addressFromString")
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
	m["1105"] = "listRemoveByIndex"
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
	m["@extrNative(1062)"] = "addressValueFromString"
	for i := 2; i <= 22; i++ {
		m[strconv.Itoa(1300+i-2)] = fmt.Sprintf("newTuple%d", i)
	}
	constructorsFunctions(ast.LibV4, m)
	return m
}

func catalogueV4() map[string]int {
	m := catalogueV3()

	delete(m, "extract")
	delete(m, "addressFromString")
	delete(m, "wavesBalance")
	delete(m, "700")
	delete(m, "1003")

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
	m["1105"] = 7
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
	// Tuple constructors
	for i := 2; i <= 22; i++ {
		m[strconv.Itoa(1300+i-2)] = 1
	}

	constructorsCatalogue(ast.LibV4, m)
	return m
}

func evaluationCatalogueV4EvaluatorV1() map[string]int {
	m := catalogueV4()
	m["isDefined"] = 7
	m["throw"] = 2
	m["!="] = 5
	m["-"] = 2
	m["!"] = 2
	m["value"] = 5
	m["valueOrErrorMessage"] = 5
	m["parseIntValue"] = 8
	m["contains"] = 12
	m["valueOrElse"] = 5
	m["containsElement"] = 12
	m["takeRightBytes"] = 11
	m["dropRightBytes"] = 11
	m["dropRight"] = 25
	m["takeRight"] = 25
	m["getInteger"] = 9
	m["getBoolean"] = 9
	m["getBinary"] = 9
	m["getString"] = 9
	m["addressFromPublicKey"] = 59
	m["@extrNative(1050)"] = 17
	m["@extrNative(1051)"] = 17
	m["@extrNative(1052)"] = 17
	m["@extrNative(1053)"] = 17
	m["@extrNative(1040)"] = 17
	m["@extrNative(1041)"] = 17
	m["@extrNative(1042)"] = 17
	m["@extrNative(1043)"] = 17
	m["@extrUser(getInteger)"] = 16
	m["@extrUser(getBoolean)"] = 16
	m["@extrUser(getBinary)"] = 16
	m["@extrUser(getString)"] = 16
	m["@extrUser(addressFromString)"] = 7
	m["@extrNative(1062)"] = 7
	m["Address"] = 0
	m["Alias"] = 0
	constructorsEvaluationCatalogueEvaluatorV1(ast.LibV4, m)
	return m
}

func evaluationCatalogueV4EvaluatorV2() map[string]int {
	m := catalogueV4()
	m["throw"] = 2
	m["Ceiling"] = 1
	m["Floor"] = 1
	m["HalfEven"] = 1
	m["Down"] = 1
	m["Up"] = 1
	m["HalfUp"] = 1
	m["HalfDown"] = 1
	m["NoAlg"] = 1
	m["Md5"] = 1
	m["Sha1"] = 1
	m["Sha224"] = 1
	m["Sha256"] = 1
	m["Sha384"] = 1
	m["Sha512"] = 1
	m["Sha3224"] = 1
	m["Sha3256"] = 1
	m["Sha3384"] = 1
	m["Sha3512"] = 1
	m["Unit"] = 1
	m["Address"] = 1
	m["Alias"] = 1
	constructorsEvaluationCatalogueEvaluatorV2(ast.LibV4, m)
	return m
}

func functionsV5() map[string]string {
	m := functionsV4()
	m["118"] = "powBigInt"
	m["119"] = "logBigInt"
	m["310"] = "toBigInt"
	m["311"] = "sumBigInt"
	m["312"] = "subtractBigInt"
	m["313"] = "multiplyBigInt"
	m["314"] = "divideBigInt"
	m["315"] = "moduloBigInt"
	m["316"] = "fractionBigInt"
	m["317"] = "fractionBigIntRounds"
	m["318"] = "unaryMinusBigInt"
	m["319"] = "gtBigInt"
	m["320"] = "geBigInt"
	m["408"] = "maxListBigInt"
	m["409"] = "minListBigInt"
	m["413"] = "bigIntToBytes"
	m["414"] = "bytesToBigInt"
	m["415"] = "bytesToBigIntLim"
	m["416"] = "bigIntToInt"
	m["422"] = "bigIntToString"
	m["423"] = "stringToBigInt"
	m["424"] = "stringToBigIntOpt"
	m["425"] = "medianListBigInt"
	m["1009"] = "hashScriptAtAddress"
	m["1020"] = "invoke"
	m["1021"] = "reentrantInvoke"
	m["1054"] = "isDataStorageUntouched"
	m["1055"] = "intFromSelfState"
	m["1056"] = "booleanFromSelfState"
	m["1057"] = "bytesFromSelfState"
	m["1058"] = "stringFromSelfState"
	m["1081"] = "calculateLeaseID"
	m["1092"] = "simplifiedLease"
	m["1093"] = "fullLease"
	m["fraction"] = "fractionIntRounds"
	m["@extrNative(1055)"] = "intValueFromSelfState"
	m["@extrNative(1056)"] = "booleanValueFromSelfState"
	m["@extrNative(1057)"] = "bytesValueFromSelfState"
	m["@extrNative(1058)"] = "stringValueFromSelfState"
	constructorsFunctions(ast.LibV5, m)
	return m
}

func catalogueV5() map[string]int {
	m := catalogueV4()
	m["107"] = 14
	m["118"] = 200
	m["119"] = 200
	m["310"] = 1
	m["311"] = 8
	m["312"] = 8
	m["313"] = 64
	m["314"] = 64
	m["315"] = 64
	m["316"] = 128
	m["317"] = 128
	m["318"] = 8
	m["319"] = 8
	m["320"] = 8
	m["408"] = 192
	m["409"] = 192
	m["413"] = 65
	m["414"] = 65
	m["415"] = 65
	m["416"] = 1
	m["422"] = 65
	m["423"] = 65
	m["424"] = 65
	m["425"] = 160
	m["1009"] = 200
	m["1020"] = 75
	m["1021"] = 75
	m["1054"] = 10
	m["1055"] = 10
	m["1056"] = 10
	m["1057"] = 10
	m["1058"] = 10
	m["1081"] = 1
	m["1092"] = 1
	m["1093"] = 1
	m["fraction"] = 17
	m["@extrNative(1055)"] = 10
	m["@extrNative(1056)"] = 10
	m["@extrNative(1057)"] = 10
	m["@extrNative(1058)"] = 10
	delete(m, "Up")
	delete(m, "HalfDown")

	constructorsCatalogue(ast.LibV5, m)
	return m
}

func evaluationCatalogueV5EvaluatorV1() map[string]int {
	m := catalogueV5()
	m["isDefined"] = 7
	m["throw"] = 2
	m["!="] = 5
	m["-"] = 2
	m["!"] = 2
	m["value"] = 5
	m["valueOrErrorMessage"] = 5
	m["parseIntValue"] = 8
	m["contains"] = 12
	m["valueOrElse"] = 5
	m["containsElement"] = 12
	m["takeRightBytes"] = 11
	m["dropRightBytes"] = 11
	m["dropRight"] = 25
	m["takeRight"] = 25
	m["fraction"] = 135
	m["getInteger"] = 9
	m["getBoolean"] = 9
	m["getBinary"] = 9
	m["getString"] = 9
	m["addressFromPublicKey"] = 59
	m["@extrNative(1050)"] = 17
	m["@extrNative(1051)"] = 17
	m["@extrNative(1052)"] = 17
	m["@extrNative(1053)"] = 17
	m["@extrNative(1040)"] = 17
	m["@extrNative(1041)"] = 17
	m["@extrNative(1042)"] = 17
	m["@extrNative(1043)"] = 17
	m["@extrUser(getInteger)"] = 16
	m["@extrUser(getBoolean)"] = 16
	m["@extrUser(getBinary)"] = 16
	m["@extrUser(getString)"] = 16
	m["@extrUser(addressFromString)"] = 7
	m["@extrNative(1055)"] = 16
	m["@extrNative(1056)"] = 16
	m["@extrNative(1057)"] = 16
	m["@extrNative(1058)"] = 16
	m["@extrNative(1062)"] = 7
	m["Address"] = 0
	m["Alias"] = 0
	constructorsEvaluationCatalogueEvaluatorV1(ast.LibV5, m)
	return m
}

func evaluationCatalogueV5EvaluatorV2() map[string]int {
	m := catalogueV5()
	m["throw"] = 2
	m["Ceiling"] = 1
	m["Floor"] = 1
	m["HalfEven"] = 1
	m["Down"] = 1
	m["HalfUp"] = 1
	m["NoAlg"] = 1
	m["Md5"] = 1
	m["Sha1"] = 1
	m["Sha224"] = 1
	m["Sha256"] = 1
	m["Sha384"] = 1
	m["Sha512"] = 1
	m["Sha3224"] = 1
	m["Sha3256"] = 1
	m["Sha3384"] = 1
	m["Sha3512"] = 1
	m["Unit"] = 1
	m["Address"] = 1
	m["Alias"] = 1
	constructorsEvaluationCatalogueEvaluatorV2(ast.LibV5, m)
	return m
}

func functionsV6() map[string]string {
	m := functionsV5()
	delete(m, "fraction")
	m["3"] = "getType"
	m["110"] = "fractionIntRounds"
	m["204"] = "takeRightBytes"
	m["205"] = "dropRightBytes"
	m["306"] = "takeRightString"
	m["307"] = "dropRightString"
	m["1350"] = "sizeTuple"
	m["sqrt"] = "sqrt"
	m["sqrtBigInt"] = "sqrtBigInt"
	m["1063"] = "addressFromPublicKeyStrict"
	m["1205"] = "splitStringV6"
	m["1209"] = "makeStringV6"
	m["1210"] = "makeString2C"
	m["1211"] = "makeString11C"
	m["1212"] = "splitString4C"
	m["1213"] = "splitString51C"
	constructorsFunctions(ast.LibV6, m)
	return m
}

func catalogueV6() map[string]int {
	m := catalogueV5()
	m["3"] = 1
	m["107"] = 1
	m["108"] = 28
	m["110"] = 1
	m["118"] = 270
	m["204"] = 6
	m["205"] = 6
	m["300"] = 1
	m["306"] = 20
	m["307"] = 20
	m["316"] = 1
	m["317"] = 1
	m["422"] = 1
	m["500"] = 180
	m["501"] = 195
	m["502"] = 136
	m["503"] = 118
	m["1061"] = 1
	m["1063"] = 1
	m["1205"] = 1
	m["1209"] = 1
	m["1210"] = 2
	m["1211"] = 11
	m["1212"] = 4
	m["1213"] = 51
	m["1350"] = 1
	for i, c := range []int{43, 50, 64, 93, 150} {
		m[strconv.Itoa(2500+i)] = c
	}
	for i, c := range []int{20, 39, 74, 147} {
		m[strconv.Itoa(2700+i)] = c
	}
	for i, c := range []int{13, 29, 58, 115} {
		m[strconv.Itoa(2800+i)] = c
	}
	for i, c := range []int{12, 23, 47, 93} {
		m[strconv.Itoa(2900+i)] = c
	}
	m["fraction"] = 4
	m["sqrt"] = 2
	m["sqrtBigInt"] = 5

	constructorsCatalogue(ast.LibV6, m)
	return m
}

func evaluationCatalogueV6EvaluatorV1() map[string]int {
	m := catalogueV6()
	m["throw"] = 2
	m["Ceiling"] = 0
	m["Floor"] = 0
	m["HalfEven"] = 0
	m["Down"] = 0
	m["HalfUp"] = 0
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
	m["Address"] = 0
	m["Alias"] = 0
	constructorsEvaluationCatalogueEvaluatorV1(ast.LibV6, m)
	return m
}

func evaluationCatalogueV6EvaluatorV2() map[string]int {
	m := catalogueV6()
	m["throw"] = 2
	m["Ceiling"] = 1
	m["Floor"] = 1
	m["HalfEven"] = 1
	m["Down"] = 1
	m["HalfUp"] = 1
	m["NoAlg"] = 1
	m["Md5"] = 1
	m["Sha1"] = 1
	m["Sha224"] = 1
	m["Sha256"] = 1
	m["Sha384"] = 1
	m["Sha512"] = 1
	m["Sha3224"] = 1
	m["Sha3256"] = 1
	m["Sha3384"] = 1
	m["Sha3512"] = 1
	m["Unit"] = 1
	m["Address"] = 1
	m["Alias"] = 1
	constructorsEvaluationCatalogueEvaluatorV2(ast.LibV6, m)
	return m
}

func constructorsFromConstants(m map[string]string, c map[string]constantDescription) {
	for _, v := range c {
		if v.constructor == "" {
			m[v.typeName] = fmt.Sprintf("create%s", v.typeName)
		}
	}
}

func createFunctionsList(cd *Coder, ver string, m map[string]string, c, ec1, ec2 map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create sorted list of functions
	cd.Line("var _functions_%s [%d]rideFunction", ver, len(keys))
	cd.Line("var _functions_map_%s map[string]rideFunction", ver)
	cd.Line("func init() {")
	// Create map of functions
	values := make([]string, len(keys))
	pairs := make([]string, len(keys))
	for i, k := range keys {
		values[i] = m[k]
		pairs[i] = fmt.Sprintf("\"%s\": %s", k, m[k])
	}
	cd.Line("_functions_%s = [%d]rideFunction{%s}", ver, len(keys), strings.Join(values, ", "))
	cd.Line("_functions_map_%s = map[string]rideFunction{%s}", ver, strings.Join(pairs, ", "))
	cd.Line("}")
	cd.Line("")

	// Create list of costs
	costs := make([]string, len(keys))
	pairs = make([]string, len(keys))
	ec1Pairs := make([]string, len(keys))
	ec2Pairs := make([]string, len(keys))
	for i, k := range keys {
		costs[i] = strconv.Itoa(c[k])
		ec1Pairs[i] = fmt.Sprintf("\"%s\":%d", k, ec1[k])
		ec2Pairs[i] = fmt.Sprintf("\"%s\":%d", k, ec2[k])
		pairs[i] = fmt.Sprintf("\"%s\":%d", k, c[k])
	}
	cd.Line("var _catalogue_%s = [...]int{%s}", ver, strings.Join(costs, ", "))
	cd.Line("")

	// Create map of function costs
	cd.Line("var Catalogue%s = map[string]int{%s}", ver, strings.Join(pairs, ", "))
	cd.Line("")

	// Create map of evaluation costs of functions and constructors
	cd.Line("var EvaluationCatalogue%sEvaluatorV1 = map[string]int{%s}", ver, strings.Join(ec1Pairs, ", "))
	cd.Line("var EvaluationCatalogue%sEvaluatorV2 = map[string]int{%s}", ver, strings.Join(ec2Pairs, ", "))

	// Create string of concatenated names of functions
	cd.Line("const _names_%s = \"%s\"", ver, strings.Join(keys, ""))

	// Create indexes for names extraction
	idx := 0
	indexes := make([]string, len(keys))
	for i, k := range keys {
		idx += len(k)
		indexes[i] = strconv.Itoa(idx)
	}
	cd.Line("var _index_%s = [...]int{0, %s} ", ver, strings.Join(indexes, ", "))
	cd.Line("")

	cd.Line("func functionName%s(i int) string {", ver)
	cd.Line("if i < 0 || i > %d {", len(keys)-1)
	cd.Line("return \"\"")
	cd.Line("}")
	cd.Line("return _names_%s[_index_%s[i]:_index_%s[i+1]]", ver, ver, ver)
	cd.Line("}")
	cd.Line("")

	cd.Line("func function%s(id int) rideFunction {", ver)
	cd.Line("if id < 0 || id > %d {", len(keys)-1)
	cd.Line("return nil")
	cd.Line("}")
	cd.Line("return _functions_%s[id]", ver)
	cd.Line("}")
	cd.Line("")

	cd.Line("func functions%s(name string) (rideFunction, bool) {", ver)
	cd.Line("f, ok := _functions_map_%s[name]", ver)
	cd.Line("return f, ok")
	cd.Line("}")
	cd.Line("")

	cd.Line("func expressionFunctions%s(name string) (rideFunction, bool) {", ver)
	cd.Line("if name == \"1020\" || name == \"1021\" {")
	cd.Line("return nil, false")
	cd.Line("}")
	cd.Line("f, ok := _functions_map_%s[name]", ver)
	cd.Line("return f, ok")
	cd.Line("}")
	cd.Line("")

	cd.Line("func checkFunction%s(name string) (uint16, bool) {", ver)
	cd.Line("for i := 0; i <= %d; i++ {", len(keys)-1)
	cd.Line("if _names_%s[_index_%s[i]:_index_%s[i+1]] == name {", ver, ver, ver)
	cd.Line("return uint16(i), true")
	cd.Line("}")
	cd.Line("}")
	cd.Line("return 0, false")
	cd.Line("}")
	cd.Line("")
	cd.Line("func cost%s(id int) int {", ver)
	cd.Line("if id < 0 || id > %d {", len(keys)-1)
	cd.Line("return -1")
	cd.Line("}")
	cd.Line("return _catalogue_%s[id]", ver)
	cd.Line("}")
	cd.Line("")
}

func GenerateFunctions(fn string) {
	cd := NewCoder("ride")
	createFunctionsList(cd, "V2", functionsV2(), catalogueV2(), evaluationCatalogueV2EvaluatorV1(), evaluationCatalogueV2EvaluatorV2())
	createFunctionsList(cd, "V3", functionsV3(), catalogueV3(), evaluationCatalogueV3EvaluatorV1(), evaluationCatalogueV3EvaluatorV2())
	createFunctionsList(cd, "V4", functionsV4(), catalogueV4(), evaluationCatalogueV4EvaluatorV1(), evaluationCatalogueV4EvaluatorV2())
	createFunctionsList(cd, "V5", functionsV5(), catalogueV5(), evaluationCatalogueV5EvaluatorV1(), evaluationCatalogueV5EvaluatorV2())
	createFunctionsList(cd, "V6", functionsV6(), catalogueV6(), evaluationCatalogueV6EvaluatorV1(), evaluationCatalogueV6EvaluatorV2())
	if err := cd.Save(fn); err != nil {
		panic(err)
	}
}
