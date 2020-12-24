// Code generated by ride/generate/main.go. DO NOT EDIT.

package ride

var _functions_V2 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionByID, transactionHeightByID, assetBalanceV3, sub, gt, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, fraction, throw, sizeBytes, takeBytes, dropBytes, concatBytes, concatStrings, takeString, dropString, sizeString, sizeList, getList, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, sigVerify, keccak256, blake2b256, sha256, toBase58, fromBase58, toBase64, fromBase64, address, alias, assetPair, dataEntry, dataTransaction, addressFromPublicKey, addressFromString, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, takeRightString, takeRightBytes, throw0, wavesBalanceV3}
var _catalogue_V2 = [...]int{11, 26, 9, 1, 1, 1, 100, 100, 100, 1, 1, 1, 1, 10, 10, 10, 10, 1, 100, 100, 100, 100, 1, 100, 1, 1, 1, 1, 1, 10, 10, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 100, 10, 10, 10, 10, 10, 10, 10, 1, 1, 2, 2, 9, 82, 124, 19, 19, 13, 30, 30, 30, 30, 35, 19, 19, 2, 109}
var CatalogueV2 = map[string]int{"!": 11, "!=": 26, "-": 9, "0": 1, "1": 1, "100": 1, "1000": 100, "1001": 100, "1003": 100, "101": 1, "102": 1, "103": 1, "104": 1, "1040": 10, "1041": 10, "1042": 10, "1043": 10, "105": 1, "1050": 100, "1051": 100, "1052": 100, "1053": 100, "106": 1, "1060": 100, "107": 1, "2": 1, "200": 1, "201": 1, "202": 1, "203": 10, "300": 10, "303": 1, "304": 1, "305": 1, "400": 2, "401": 2, "410": 1, "411": 1, "412": 1, "420": 1, "421": 1, "500": 100, "501": 10, "502": 10, "503": 10, "600": 10, "601": 10, "602": 10, "603": 10, "Address": 1, "Alias": 1, "AssetPair": 2, "DataEntry": 2, "DataTransaction": 9, "addressFromPublicKey": 82, "addressFromString": 124, "dropRight": 19, "dropRightBytes": 19, "extract": 13, "getBinary": 30, "getBoolean": 30, "getInteger": 30, "getString": 30, "isDefined": 35, "takeRight": 19, "takeRightBytes": 19, "throw": 2, "wavesBalance": 109}

const _names_V2 = "!!=-011001000100110031011021031041040104110421043105105010511052105310610601072200201202203300303304305400401410411412420421500501502503600601602603AddressAliasAssetPairDataEntryDataTransactionaddressFromPublicKeyaddressFromStringdropRightdropRightBytesextractgetBinarygetBooleangetIntegergetStringisDefinedtakeRighttakeRightBytesthrowwavesBalance"

var _index_V2 = [...]int{0, 1, 3, 4, 5, 6, 9, 13, 17, 21, 24, 27, 30, 33, 37, 41, 45, 49, 52, 56, 60, 64, 68, 71, 75, 78, 79, 82, 85, 88, 91, 94, 97, 100, 103, 106, 109, 112, 115, 118, 121, 124, 127, 130, 133, 136, 139, 142, 145, 148, 155, 160, 169, 178, 193, 213, 230, 239, 253, 260, 269, 279, 289, 298, 307, 316, 330, 335, 347}

func functionNameV2(i int) string {
	if i < 0 || i > 67 {
		return ""
	}
	return _names_V2[_index_V2[i]:_index_V2[i+1]]
}
func functionV2(id int) rideFunction {
	if id < 0 || id > 67 {
		return nil
	}
	return _functions_V2[id]
}
func checkFunctionV2(name string) (uint16, bool) {
	for i := 0; i <= 67; i++ {
		if _names_V2[_index_V2[i]:_index_V2[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}
func costV2(id int) int {
	if id < 0 || id > 67 {
		return -1
	}
	return _catalogue_V2[id]
}

var _functions_V3 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionHeightByID, assetBalanceV3, assetInfoV3, blockInfoByHeight, transferByID, sub, gt, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, addressToString, fraction, pow, log, createList, bytesToUTF8String, bytesToInt, bytesToIntWithOffset, indexOfSubstring, indexOfSubstringWithOffset, splitString, parseInt, lastIndexOfSubstring, lastIndexOfSubstringWithOffset, throw, sizeBytes, takeBytes, dropBytes, concatBytes, concatStrings, takeString, dropString, sizeString, sizeList, getList, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, sigVerify, keccak256, blake2b256, sha256, rsaVerify, toBase58, fromBase58, toBase64, fromBase64, toBase16, fromBase16, checkMerkleProof, intValueFromArray, booleanValueFromArray, bytesValueFromArray, stringValueFromArray, intValueFromState, booleanValueFromState, bytesValueFromState, stringValueFromState, addressValueFromString, bytesValueFromArrayByIndex, booleanValueFromArrayByIndex, intValueFromArrayByIndex, stringValueFromArrayByIndex, address, alias, assetPair, createBuy, createCeiling, dataEntry, dataTransaction, createDown, createFloor, createHalfDown, createHalfEven, createHalfUp, createMd5, createNoAlg, scriptResult, scriptTransfer, createSell, createSha1, createSha224, createSha256, createSha3224, createSha3256, createSha3384, createSha3512, createSha384, createSha512, transferSet, unit, createUp, writeSet, addressFromPublicKey, addressFromString, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, parseIntValue, takeRightString, takeRightBytes, throw0, value, valueOrErrorMessage, wavesBalanceV3}
var _catalogue_V3 = [...]int{1, 1, 1, 1, 1, 1, 100, 100, 100, 100, 100, 1, 1, 1, 1, 10, 10, 10, 10, 1, 100, 100, 100, 100, 1, 100, 10, 1, 100, 100, 2, 20, 10, 10, 20, 20, 100, 20, 20, 20, 1, 1, 1, 1, 10, 10, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 100, 10, 10, 10, 300, 10, 10, 10, 10, 10, 10, 30, 10, 10, 10, 10, 100, 100, 100, 100, 124, 10, 10, 10, 10, 1, 1, 2, 0, 0, 2, 9, 0, 0, 0, 0, 0, 0, 0, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 82, 124, 19, 19, 13, 30, 30, 30, 30, 1, 20, 19, 19, 1, 13, 13, 109}
var CatalogueV3 = map[string]int{"!": 1, "!=": 1, "-": 1, "0": 1, "1": 1, "100": 1, "1001": 100, "1003": 100, "1004": 100, "1005": 100, "1006": 100, "101": 1, "102": 1, "103": 1, "104": 1, "1040": 10, "1041": 10, "1042": 10, "1043": 10, "105": 1, "1050": 100, "1051": 100, "1052": 100, "1053": 100, "106": 1, "1060": 100, "1061": 10, "107": 1, "108": 100, "109": 100, "1100": 2, "1200": 20, "1201": 10, "1202": 10, "1203": 20, "1204": 20, "1205": 100, "1206": 20, "1207": 20, "1208": 20, "2": 1, "200": 1, "201": 1, "202": 1, "203": 10, "300": 10, "303": 1, "304": 1, "305": 1, "400": 2, "401": 2, "410": 1, "411": 1, "412": 1, "420": 1, "421": 1, "500": 100, "501": 10, "502": 10, "503": 10, "504": 300, "600": 10, "601": 10, "602": 10, "603": 10, "604": 10, "605": 10, "700": 30, "@extrNative(1040)": 10, "@extrNative(1041)": 10, "@extrNative(1042)": 10, "@extrNative(1043)": 10, "@extrNative(1050)": 100, "@extrNative(1051)": 100, "@extrNative(1052)": 100, "@extrNative(1053)": 100, "@extrUser(addressFromString)": 124, "@extrUser(getBinary)": 10, "@extrUser(getBoolean)": 10, "@extrUser(getInteger)": 10, "@extrUser(getString)": 10, "Address": 1, "Alias": 1, "AssetPair": 2, "Buy": 0, "Ceiling": 0, "DataEntry": 2, "DataTransaction": 9, "Down": 0, "Floor": 0, "HalfDown": 0, "HalfEven": 0, "HalfUp": 0, "Md5": 0, "NoAlg": 0, "ScriptResult": 2, "ScriptTransfer": 3, "Sell": 0, "Sha1": 0, "Sha224": 0, "Sha256": 0, "Sha3224": 0, "Sha3256": 0, "Sha3384": 0, "Sha3512": 0, "Sha384": 0, "Sha512": 0, "TransferSet": 1, "Unit": 0, "Up": 0, "WriteSet": 1, "addressFromPublicKey": 82, "addressFromString": 124, "dropRight": 19, "dropRightBytes": 19, "extract": 13, "getBinary": 30, "getBoolean": 30, "getInteger": 30, "getString": 30, "isDefined": 1, "parseIntValue": 20, "takeRight": 19, "takeRightBytes": 19, "throw": 1, "value": 13, "valueOrErrorMessage": 13, "wavesBalance": 109}

const _names_V3 = "!!=-0110010011003100410051006101102103104104010411042104310510501051105210531061060106110710810911001200120112021203120412051206120712082200201202203300303304305400401410411412420421500501502503504600601602603604605700@extrNative(1040)@extrNative(1041)@extrNative(1042)@extrNative(1043)@extrNative(1050)@extrNative(1051)@extrNative(1052)@extrNative(1053)@extrUser(addressFromString)@extrUser(getBinary)@extrUser(getBoolean)@extrUser(getInteger)@extrUser(getString)AddressAliasAssetPairBuyCeilingDataEntryDataTransactionDownFloorHalfDownHalfEvenHalfUpMd5NoAlgScriptResultScriptTransferSellSha1Sha224Sha256Sha3224Sha3256Sha3384Sha3512Sha384Sha512TransferSetUnitUpWriteSetaddressFromPublicKeyaddressFromStringdropRightdropRightBytesextractgetBinarygetBooleangetIntegergetStringisDefinedparseIntValuetakeRighttakeRightBytesthrowvaluevalueOrErrorMessagewavesBalance"

var _index_V3 = [...]int{0, 1, 3, 4, 5, 6, 9, 13, 17, 21, 25, 29, 32, 35, 38, 41, 45, 49, 53, 57, 60, 64, 68, 72, 76, 79, 83, 87, 90, 93, 96, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 137, 140, 143, 146, 149, 152, 155, 158, 161, 164, 167, 170, 173, 176, 179, 182, 185, 188, 191, 194, 197, 200, 203, 206, 209, 212, 215, 218, 235, 252, 269, 286, 303, 320, 337, 354, 382, 402, 423, 444, 464, 471, 476, 485, 488, 495, 504, 519, 523, 528, 536, 544, 550, 553, 558, 570, 584, 588, 592, 598, 604, 611, 618, 625, 632, 638, 644, 655, 659, 661, 669, 689, 706, 715, 729, 736, 745, 755, 765, 774, 783, 796, 805, 819, 824, 829, 848, 860}

func functionNameV3(i int) string {
	if i < 0 || i > 127 {
		return ""
	}
	return _names_V3[_index_V3[i]:_index_V3[i+1]]
}
func functionV3(id int) rideFunction {
	if id < 0 || id > 127 {
		return nil
	}
	return _functions_V3[id]
}
func checkFunctionV3(name string) (uint16, bool) {
	for i := 0; i <= 127; i++ {
		if _names_V3[_index_V3[i]:_index_V3[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}
func costV3(id int) int {
	if id < 0 || id > 127 {
		return -1
	}
	return _catalogue_V3[id]
}

var _functions_V4 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionHeightByID, assetInfoV4, blockInfoByHeight, transferByID, wavesBalanceV4, assetBalanceV4, sub, gt, invoke, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, addressToString, addressFromString, fraction, transferFromProtobuf, pow, calculateAssetID, log, simplifiedIssue, fullIssue, limitedCreateList, appendToList, concatList, indexOfList, lastIndexOfList, bytesToUTF8String, bytesToInt, bytesToIntWithOffset, indexOfSubstring, indexOfSubstringWithOffset, splitString, parseInt, lastIndexOfSubstring, lastIndexOfSubstringWithOffset, makeString, newTuple2, newTuple3, newTuple4, newTuple5, newTuple6, newTuple7, newTuple8, newTuple9, newTuple10, newTuple11, newTuple12, newTuple13, newTuple14, newTuple15, newTuple16, newTuple17, newTuple18, newTuple19, newTuple20, newTuple21, newTuple22, throw, sizeBytes, takeBytes, dropBytes, concatBytes, bls12Groth16Verify_1, bls12Groth16Verify_2, bls12Groth16Verify_3, bls12Groth16Verify_4, bls12Groth16Verify_5, bls12Groth16Verify_6, bls12Groth16Verify_7, bls12Groth16Verify_8, bls12Groth16Verify_9, bls12Groth16Verify_10, bls12Groth16Verify_11, bls12Groth16Verify_12, bls12Groth16Verify_13, bls12Groth16Verify_14, bls12Groth16Verify_15, bn256Groth16Verify_1, bn256Groth16Verify_2, bn256Groth16Verify_3, bn256Groth16Verify_4, bn256Groth16Verify_5, bn256Groth16Verify_6, bn256Groth16Verify_7, bn256Groth16Verify_8, bn256Groth16Verify_9, bn256Groth16Verify_10, bn256Groth16Verify_11, bn256Groth16Verify_12, bn256Groth16Verify_13, bn256Groth16Verify_14, bn256Groth16Verify_15, sigVerify_8, sigVerify_16, sigVerify_32, sigVerify_64, sigVerify_128, rsaVerify_16, rsaVerify_32, rsaVerify_64, rsaVerify_128, keccak256_16, keccak256_32, keccak256_64, keccak256_128, blake2b256_16, blake2b256_32, blake2b256_64, blake2b256_128, sha256_16, sha256_32, sha256_64, sha256_128, concatStrings, takeString, dropString, sizeString, sizeList, getList, median, max, min, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, sigVerify, keccak256, blake2b256, sha256, rsaVerify, toBase58, fromBase58, toBase64, fromBase64, toBase16, fromBase16, rebuildMerkleRoot, bls12Groth16Verify, bn256Groth16Verify, ecRecover, intValueFromArray, booleanValueFromArray, bytesValueFromArray, stringValueFromArray, intValueFromState, booleanValueFromState, bytesValueFromState, stringValueFromState, addressFromString, addressValueFromString, bytesValueFromArrayByIndex, booleanValueFromArrayByIndex, intValueFromArrayByIndex, stringValueFromArrayByIndex, address, alias, assetPair, attachedPayment, checkedBytesDataEntry, checkedBooleanDataEntry, burn, createBuy, createCeiling, dataTransaction, checkedDeleteEntry, createDown, createFloor, createHalfDown, createHalfEven, createHalfUp, checkedIntDataEntry, createMd5, createNoAlg, reissue, scriptTransfer, createSell, createSha1, createSha224, createSha256, createSha3224, createSha3256, createSha3384, createSha3512, createSha384, createSha512, sponsorship, checkedStringDataEntry, unit, createUp, addressFromPublicKey, addressFromString, contains, containsElement, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, parseIntValue, takeRightString, takeRightBytes, throw0, value, valueOrElse, valueOrErrorMessage}
var _catalogue_V4 = [...]int{1, 1, 1, 1, 1, 1, 100, 15, 100, 100, 10, 10, 1, 1, 1, 1, 1, 10, 10, 10, 10, 1, 10, 10, 10, 10, 1, 5, 10, 1, 1, 5, 100, 10, 100, 1, 1, 1, 1, 4, 5, 5, 7, 1, 1, 3, 3, 75, 2, 3, 3, 30, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 6, 6, 2, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000, 2100, 2200, 2300, 2400, 2500, 2600, 800, 850, 950, 1000, 1050, 1100, 1150, 1200, 1250, 1300, 1350, 1400, 1450, 1550, 1600, 47, 57, 70, 102, 172, 500, 550, 625, 750, 10, 25, 50, 100, 10, 25, 50, 100, 10, 25, 50, 100, 20, 20, 20, 1, 2, 2, 20, 3, 3, 1, 8, 1, 1, 1, 200, 200, 200, 200, 1000, 3, 1, 35, 40, 10, 10, 30, 2700, 1650, 70, 10, 10, 10, 10, 10, 10, 10, 10, 1, 124, 10, 10, 10, 10, 1, 1, 2, 2, 2, 2, 2, 0, 0, 9, 1, 0, 0, 0, 0, 0, 2, 0, 0, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 0, 63, 124, 3, 5, 20, 6, 13, 30, 30, 30, 30, 1, 2, 20, 6, 1, 2, 2, 2}
var CatalogueV4 = map[string]int{"!": 1, "!=": 1, "-": 1, "0": 1, "1": 1, "100": 1, "1001": 100, "1004": 15, "1005": 100, "1006": 100, "1007": 10, "1008": 10, "101": 1, "102": 1, "1020": 1, "103": 1, "104": 1, "1040": 10, "1041": 10, "1042": 10, "1043": 10, "105": 1, "1050": 10, "1051": 10, "1052": 10, "1053": 10, "106": 1, "1060": 5, "1061": 10, "1062": 1, "107": 1, "1070": 5, "108": 100, "1080": 10, "109": 100, "1090": 1, "1091": 1, "1100": 1, "1101": 1, "1102": 4, "1103": 5, "1104": 5, "1200": 7, "1201": 1, "1202": 1, "1203": 3, "1204": 3, "1205": 75, "1206": 2, "1207": 3, "1208": 3, "1209": 30, "1300": 1, "1301": 1, "1302": 1, "1303": 1, "1304": 1, "1305": 1, "1306": 1, "1307": 1, "1308": 1, "1309": 1, "1310": 1, "1311": 1, "1312": 1, "1313": 1, "1314": 1, "1315": 1, "1316": 1, "1317": 1, "1318": 1, "1319": 1, "1320": 1, "2": 1, "200": 1, "201": 6, "202": 6, "203": 2, "2400": 1200, "2401": 1300, "2402": 1400, "2403": 1500, "2404": 1600, "2405": 1700, "2406": 1800, "2407": 1900, "2408": 2000, "2409": 2100, "2410": 2200, "2411": 2300, "2412": 2400, "2413": 2500, "2414": 2600, "2450": 800, "2451": 850, "2452": 950, "2453": 1000, "2454": 1050, "2455": 1100, "2456": 1150, "2457": 1200, "2458": 1250, "2459": 1300, "2460": 1350, "2461": 1400, "2462": 1450, "2463": 1550, "2464": 1600, "2500": 47, "2501": 57, "2502": 70, "2503": 102, "2504": 172, "2600": 500, "2601": 550, "2602": 625, "2603": 750, "2700": 10, "2701": 25, "2702": 50, "2703": 100, "2800": 10, "2801": 25, "2802": 50, "2803": 100, "2900": 10, "2901": 25, "2902": 50, "2903": 100, "300": 20, "303": 20, "304": 20, "305": 1, "400": 2, "401": 2, "405": 20, "406": 3, "407": 3, "410": 1, "411": 8, "412": 1, "420": 1, "421": 1, "500": 200, "501": 200, "502": 200, "503": 200, "504": 1000, "600": 3, "601": 1, "602": 35, "603": 40, "604": 10, "605": 10, "701": 30, "800": 2700, "801": 1650, "900": 70, "@extrNative(1040)": 10, "@extrNative(1041)": 10, "@extrNative(1042)": 10, "@extrNative(1043)": 10, "@extrNative(1050)": 10, "@extrNative(1051)": 10, "@extrNative(1052)": 10, "@extrNative(1053)": 10, "@extrNative(1062)": 1, "@extrUser(addressFromString)": 124, "@extrUser(getBinary)": 10, "@extrUser(getBoolean)": 10, "@extrUser(getInteger)": 10, "@extrUser(getString)": 10, "Address": 1, "Alias": 1, "AssetPair": 2, "AttachedPayment": 2, "BinaryEntry": 2, "BooleanEntry": 2, "Burn": 2, "Buy": 0, "Ceiling": 0, "DataTransaction": 9, "DeleteEntry": 1, "Down": 0, "Floor": 0, "HalfDown": 0, "HalfEven": 0, "HalfUp": 0, "IntegerEntry": 2, "Md5": 0, "NoAlg": 0, "Reissue": 3, "ScriptTransfer": 3, "Sell": 0, "Sha1": 0, "Sha224": 0, "Sha256": 0, "Sha3224": 0, "Sha3256": 0, "Sha3384": 0, "Sha3512": 0, "Sha384": 0, "Sha512": 0, "SponsorFee": 2, "StringEntry": 2, "Unit": 0, "Up": 0, "addressFromPublicKey": 63, "addressFromString": 124, "contains": 3, "containsElement": 5, "dropRight": 20, "dropRightBytes": 6, "extract": 13, "getBinary": 30, "getBoolean": 30, "getInteger": 30, "getString": 30, "isDefined": 1, "parseIntValue": 2, "takeRight": 20, "takeRightBytes": 6, "throw": 1, "value": 2, "valueOrElse": 2, "valueOrErrorMessage": 2}

const _names_V4 = "!!=-0110010011004100510061007100810110210201031041040104110421043105105010511052105310610601061106210710701081080109109010911100110111021103110412001201120212031204120512061207120812091300130113021303130413051306130713081309131013111312131313141315131613171318131913202200201202203240024012402240324042405240624072408240924102411241224132414245024512452245324542455245624572458245924602461246224632464250025012502250325042600260126022603270027012702270328002801280228032900290129022903300303304305400401405406407410411412420421500501502503504600601602603604605701800801900@extrNative(1040)@extrNative(1041)@extrNative(1042)@extrNative(1043)@extrNative(1050)@extrNative(1051)@extrNative(1052)@extrNative(1053)@extrNative(1062)@extrUser(addressFromString)@extrUser(getBinary)@extrUser(getBoolean)@extrUser(getInteger)@extrUser(getString)AddressAliasAssetPairAttachedPaymentBinaryEntryBooleanEntryBurnBuyCeilingDataTransactionDeleteEntryDownFloorHalfDownHalfEvenHalfUpIntegerEntryMd5NoAlgReissueScriptTransferSellSha1Sha224Sha256Sha3224Sha3256Sha3384Sha3512Sha384Sha512SponsorFeeStringEntryUnitUpaddressFromPublicKeyaddressFromStringcontainscontainsElementdropRightdropRightBytesextractgetBinarygetBooleangetIntegergetStringisDefinedparseIntValuetakeRighttakeRightBytesthrowvaluevalueOrElsevalueOrErrorMessage"

var _index_V4 = [...]int{0, 1, 3, 4, 5, 6, 9, 13, 17, 21, 25, 29, 33, 36, 39, 43, 46, 49, 53, 57, 61, 65, 68, 72, 76, 80, 84, 87, 91, 95, 99, 102, 106, 109, 113, 116, 120, 124, 128, 132, 136, 140, 144, 148, 152, 156, 160, 164, 168, 172, 176, 180, 184, 188, 192, 196, 200, 204, 208, 212, 216, 220, 224, 228, 232, 236, 240, 244, 248, 252, 256, 260, 264, 268, 269, 272, 275, 278, 281, 285, 289, 293, 297, 301, 305, 309, 313, 317, 321, 325, 329, 333, 337, 341, 345, 349, 353, 357, 361, 365, 369, 373, 377, 381, 385, 389, 393, 397, 401, 405, 409, 413, 417, 421, 425, 429, 433, 437, 441, 445, 449, 453, 457, 461, 465, 469, 473, 477, 481, 485, 488, 491, 494, 497, 500, 503, 506, 509, 512, 515, 518, 521, 524, 527, 530, 533, 536, 539, 542, 545, 548, 551, 554, 557, 560, 563, 566, 569, 572, 589, 606, 623, 640, 657, 674, 691, 708, 725, 753, 773, 794, 815, 835, 842, 847, 856, 871, 882, 894, 898, 901, 908, 923, 934, 938, 943, 951, 959, 965, 977, 980, 985, 992, 1006, 1010, 1014, 1020, 1026, 1033, 1040, 1047, 1054, 1060, 1066, 1076, 1087, 1091, 1093, 1113, 1130, 1138, 1153, 1162, 1176, 1183, 1192, 1202, 1212, 1221, 1230, 1243, 1252, 1266, 1271, 1276, 1287, 1306}

func functionNameV4(i int) string {
	if i < 0 || i > 225 {
		return ""
	}
	return _names_V4[_index_V4[i]:_index_V4[i+1]]
}
func functionV4(id int) rideFunction {
	if id < 0 || id > 225 {
		return nil
	}
	return _functions_V4[id]
}
func checkFunctionV4(name string) (uint16, bool) {
	for i := 0; i <= 225; i++ {
		if _names_V4[_index_V4[i]:_index_V4[i+1]] == name {
			return uint16(i), true
		}
	}
	return 0, false
}
func costV4(id int) int {
	if id < 0 || id > 225 {
		return -1
	}
	return _catalogue_V4[id]
}
