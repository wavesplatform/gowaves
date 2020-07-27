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
	m["Ceiling"] = "ceiling"
	m["Floor"] = "floor"
	m["HalfEven"] = "halfEven"
	m["Down"] = "down"
	m["Up"] = "up"
	m["HalfUp"] = "halfUp"
	m["HalfDown"] = "halfDown"

	m["NoAlg"] = "noAlg"
	m["Md5"] = "md5"
	m["Sha1"] = "sha1"
	m["Sha224"] = "sha224"
	m["Sha256"] = "rideSha256"
	m["Sha384"] = "sha384"
	m["Sha512"] = "sha512"
	m["Sha3224"] = "sha3224"
	m["Sha3256"] = "sha3256"
	m["Sha3384"] = "sha3384"
	m["Sha3512"] = "sha3512"

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

func createFunctionsList(sb *strings.Builder, ver string, m map[string]string) {
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
	sb.WriteString(fmt.Sprintf("func functions%s(id int) rideFunction {\n", ver))
	sb.WriteString(fmt.Sprintf("if id < 0 || id > %d {\n", len(keys)-1))
	sb.WriteString("return nil\n")
	sb.WriteString("}\n")
	sb.WriteString(fmt.Sprintf("return _functions_%s[id]\n}\n", ver))
}

func main() {
	sb := new(strings.Builder)
	sb.WriteString("// Code generated by fride/generate/main.go. DO NOT EDIT.\n")
	sb.WriteString("\n")
	sb.WriteString("package fride\n")
	createFunctionsList(sb, "V2", functionsV2())
	createFunctionsList(sb, "V3", functionsV3())
	createFunctionsList(sb, "V4", functionsV4())
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
