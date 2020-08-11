// Code generated by fride/generate/main.go. DO NOT EDIT.

package fride

var _functions_V2 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionByID, transactionHeightByID, assetBalanceV3, sub, gt, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, fraction, throw, sizeBytes, takeBytes, dropBytes, concatBytes, concatStrings, takeStrings, dropStrings, sizeStrings, sizeList, getList, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, unlimitedSigVerify, unlimitedKeccak256, unlimitedBlake2b256, unlimitedSha256, toBase58, fromBase58, toBase64, fromBase64, address, alias, assetPair, dataEntry, dataTransaction, addressFromPublicKey, addressFromString, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, takeRightString, takeRightBytes, throw0, wavesBalanceV3}
var _catalogue_V2 = [...]int{11, 26, 9, 1, 1, 1, 100, 100, 100, 1, 1, 1, 1, 10, 10, 10, 10, 1, 100, 100, 100, 100, 1, 100, 1, 1, 1, 1, 1, 10, 10, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 100, 10, 10, 10, 10, 10, 10, 10, 1, 1, 2, 2, 9, 82, 124, 19, 19, 13, 10, 10, 10, 10, 35, 19, 19, 2, 109}

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
func checkFunctionV2(name string) (byte, bool) {
	for i := 0; i <= 67; i++ {
		if _names_V2[_index_V2[i]:_index_V2[i+1]] == name {
			return byte(i), true
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

var _functions_V3 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionHeightByID, assetBalanceV3, assetInfoV3, blockInfoByHeight, transferByID, sub, gt, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, addressToString, fraction, pow, log, createList, bytesToUTF8String, bytesToLong, bytesToLongWithOffset, indexOfSubstring, indexOfSubstringWithOffset, splitString, parseInt, lastIndexOfSubstring, lastIndexOfSubstringWithOffset, throw, sizeBytes, takeBytes, dropBytes, concatBytes, concatStrings, takeStrings, dropStrings, sizeStrings, sizeList, getList, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, unlimitedSigVerify, unlimitedKeccak256, unlimitedBlake2b256, unlimitedSha256, unlimitedRSAVerify, toBase58, fromBase58, toBase64, fromBase64, toBase16, fromBase16, checkMerkleProof, intValueFromArray, booleanValueFromArray, bytesValueFromArray, stringValueFromArray, intValueFromState, booleanValueFromState, bytesValueFromState, stringValueFromState, addressFromString, bytesValueFromArrayByIndex, booleanValueFromArrayByIndex, intValueFromArrayByIndex, stringValueFromArrayByIndex, address, alias, assetPair, createBuy, createCeiling, dataEntry, dataTransaction, createDown, createFloor, createHalfDown, createHalfEven, createHalfUp, createMd5, createNoAlg, scriptResult, scriptTransfer, createSell, createSha1, createSha224, createSha256, createSha3224, createSha3256, createSha3384, createSha3512, createSha384, createSha512, transferSet, unit, createUp, writeSet, addressFromPublicKey, addressFromString, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, parseIntValue, takeRightString, takeRightBytes, throw0, value, valueOrErrorMessage, wavesBalanceV3}
var _catalogue_V3 = [...]int{1, 1, 1, 1, 1, 1, 100, 100, 100, 100, 100, 1, 1, 1, 1, 10, 10, 10, 10, 1, 100, 100, 100, 100, 1, 100, 10, 1, 100, 100, 2, 20, 10, 10, 20, 20, 100, 20, 20, 20, 1, 1, 1, 1, 10, 10, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 100, 10, 10, 10, 300, 10, 10, 10, 10, 10, 10, 30, 10, 10, 10, 10, 100, 100, 100, 100, 124, 10, 10, 10, 10, 1, 1, 2, 0, 0, 2, 9, 0, 0, 0, 0, 0, 0, 0, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 82, 124, 19, 19, 13, 10, 10, 10, 10, 1, 20, 19, 19, 1, 13, 13, 109}

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
func checkFunctionV3(name string) (byte, bool) {
	for i := 0; i <= 127; i++ {
		if _names_V3[_index_V3[i]:_index_V3[i+1]] == name {
			return byte(i), true
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

var _functions_V4 = [...]rideFunction{unaryNot, neq, unaryMinus, eq, instanceOf, sum, transactionHeightByID, assetBalanceV4, assetInfoV4, blockInfoByHeight, transferByID, sub, gt, ge, mul, intFromArray, booleanFromArray, bytesFromArray, stringFromArray, div, intFromState, booleanFromState, bytesFromState, stringFromState, mod, addressFromRecipient, addressToString, fraction, transferFromProtobuf, pow, calculateAssetID, log, simplifiedIssue, fullIssue, limitedCreateList, appendToList, concatList, indexOfList, lastIndexOfList, bytesToUTF8String, bytesToLong, bytesToLongWithOffset, indexOfSubstring, indexOfSubstringWithOffset, splitString, parseInt, lastIndexOfSubstring, lastIndexOfSubstringWithOffset, throw, sizeBytes, takeBytes, dropBytes, concatBytes, limitedGroth16Verify_1, limitedGroth16Verify_2, limitedGroth16Verify_3, limitedGroth16Verify_4, limitedGroth16Verify_5, limitedGroth16Verify_6, limitedGroth16Verify_7, limitedGroth16Verify_8, limitedGroth16Verify_9, limitedGroth16Verify_10, limitedGroth16Verify_11, limitedGroth16Verify_12, limitedGroth16Verify_13, limitedGroth16Verify_14, limitedGroth16Verify_15, sigVerify_16, sigVerify_32, sigVerify_64, sigVerify_128, rsaVerify_16, rsaVerify_32, rsaVerify_64, rsaVerify_128, keccak256_16, keccak256_32, keccak256_64, keccak256_128, blake2b256_16, blake2b256_32, blake2b256_64, blake2b256_128, sha256_16, sha256_32, sha256_64, sha256_128, concatStrings, takeStrings, dropStrings, sizeStrings, sizeList, getList, median, max, min, intToBytes, stringToBytes, booleanToBytes, intToString, booleanToString, unlimitedSigVerify, unlimitedKeccak256, unlimitedBlake2b256, unlimitedSha256, unlimitedRSAVerify, toBase58, fromBase58, toBase64, fromBase64, toBase16, fromBase16, rebuildMerkleRoot, unlimitedGroth16Verify, ecRecover, intValueFromArray, booleanValueFromArray, bytesValueFromArray, stringValueFromArray, intValueFromState, booleanValueFromState, bytesValueFromState, stringValueFromState, addressFromString, bytesValueFromArrayByIndex, booleanValueFromArrayByIndex, intValueFromArrayByIndex, stringValueFromArrayByIndex, address, alias, assetPair, checkedBytesDataEntry, checkedBooleanDataEntry, burn, createBuy, createCeiling, dataTransaction, checkedDeleteEntry, createDown, createFloor, createHalfDown, createHalfEven, createHalfUp, checkedIntDataEntry, issue, createMd5, createNoAlg, reissue, scriptTransfer, createSell, createSha1, createSha224, createSha256, createSha3224, createSha3256, createSha3384, createSha3512, createSha384, createSha512, sponsorship, checkedStringDataEntry, unit, createUp, addressFromPublicKey, addressFromString, contains, dropRightString, dropRightBytes, extract, bytesFromArrayByIndex, booleanFromArrayByIndex, intFromArrayByIndex, stringFromArrayByIndex, isDefined, parseIntValue, takeRightString, takeRightBytes, throw0, value, valueOrElse, valueOrErrorMessage, wavesBalanceV4}
var _catalogue_V4 = [...]int{1, 1, 1, 1, 1, 1, 100, 100, 100, 100, 100, 1, 1, 1, 1, 10, 10, 10, 10, 1, 100, 100, 100, 100, 1, 100, 10, 1, 5, 100, 10, 100, 0, 0, 2, 3, 10, 5, 5, 20, 10, 10, 20, 20, 100, 20, 20, 20, 1, 1, 1, 1, 10, 1900, 2000, 2150, 2300, 2450, 2550, 2700, 2900, 3000, 3150, 3250, 3400, 3500, 3650, 3750, 100, 110, 125, 150, 100, 500, 550, 625, 10, 25, 50, 100, 10, 25, 50, 100, 10, 25, 50, 100, 10, 1, 1, 1, 2, 2, 10, 3, 3, 1, 1, 1, 1, 1, 100, 10, 10, 10, 300, 10, 10, 10, 10, 10, 10, 30, 3900, 70, 10, 10, 10, 10, 100, 100, 100, 100, 124, 10, 10, 10, 10, 1, 1, 2, 2, 2, 2, 0, 0, 9, 1, 0, 0, 0, 0, 0, 2, 7, 0, 0, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 82, 124, 20, 19, 19, 13, 10, 10, 10, 10, 1, 20, 19, 19, 1, 13, 13, 13, 109}

const _names_V4 = "!!=-01100100110031004100510061011021031041040104110421043105105010511052105310610601061107107010810801091090109111001101110211031104120012011202120312041205120612071208220020120220324002401240224032404240524062407240824092410241124122413241425002501250225032600260126022603270027012702270328002801280228032900290129022903300303304305400401405406407410411412420421500501502503504600601602603604605701800900@extrNative(1040)@extrNative(1041)@extrNative(1042)@extrNative(1043)@extrNative(1050)@extrNative(1051)@extrNative(1052)@extrNative(1053)@extrUser(addressFromString)@extrUser(getBinary)@extrUser(getBoolean)@extrUser(getInteger)@extrUser(getString)AddressAliasAssetPairBinaryEntryBooleanEntryBurnBuyCeilingDataTransactionDeleteEntryDownFloorHalfDownHalfEvenHalfUpIntegerEntryIssueMd5NoAlgReissueScriptTransferSellSha1Sha224Sha256Sha3224Sha3256Sha3384Sha3512Sha384Sha512SponsorFeeStringEntryUnitUpaddressFromPublicKeyaddressFromStringcontainsdropRightdropRightBytesextractgetBinarygetBooleangetIntegergetStringisDefinedparseIntValuetakeRighttakeRightBytesthrowvaluevalueOrElsevalueOrErrorMessagewavesBalance"

var _index_V4 = [...]int{0, 1, 3, 4, 5, 6, 9, 13, 17, 21, 25, 29, 32, 35, 38, 41, 45, 49, 53, 57, 60, 64, 68, 72, 76, 79, 83, 87, 90, 94, 97, 101, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144, 148, 152, 156, 160, 164, 168, 169, 172, 175, 178, 181, 185, 189, 193, 197, 201, 205, 209, 213, 217, 221, 225, 229, 233, 237, 241, 245, 249, 253, 257, 261, 265, 269, 273, 277, 281, 285, 289, 293, 297, 301, 305, 309, 313, 317, 321, 324, 327, 330, 333, 336, 339, 342, 345, 348, 351, 354, 357, 360, 363, 366, 369, 372, 375, 378, 381, 384, 387, 390, 393, 396, 399, 402, 405, 422, 439, 456, 473, 490, 507, 524, 541, 569, 589, 610, 631, 651, 658, 663, 672, 683, 695, 699, 702, 709, 724, 735, 739, 744, 752, 760, 766, 778, 783, 786, 791, 798, 812, 816, 820, 826, 832, 839, 846, 853, 860, 866, 872, 882, 893, 897, 899, 919, 936, 944, 953, 967, 974, 983, 993, 1003, 1012, 1021, 1034, 1043, 1057, 1062, 1067, 1078, 1097, 1109}

func functionNameV4(i int) string {
	if i < 0 || i > 182 {
		return ""
	}
	return _names_V4[_index_V4[i]:_index_V4[i+1]]
}
func functionV4(id int) rideFunction {
	if id < 0 || id > 182 {
		return nil
	}
	return _functions_V4[id]
}
func checkFunctionV4(name string) (byte, bool) {
	for i := 0; i <= 182; i++ {
		if _names_V4[_index_V4[i]:_index_V4[i+1]] == name {
			return byte(i), true
		}
	}
	return 0, false
}
func costV4(id int) int {
	if id < 0 || id > 182 {
		return -1
	}
	return _catalogue_V4[id]
}
