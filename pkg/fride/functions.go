package fride

import (
	"strconv"

	"github.com/pkg/errors"
)

func functionsV12() map[string]rideFunction {
	m := make(map[string]rideFunction)
	m["0"] = eq
	//m["1"] = FunctionFromPredefined(NativeIsInstanceOf, 2)
	//m["2"] = FunctionFromPredefined(NativeThrow, 1)
	//
	//m["100"] = FunctionFromPredefined(NativeSumLong, 2)
	//m["101"] = FunctionFromPredefined(NativeSubLong, 2)
	//m["102"] = FunctionFromPredefined(NativeGtLong, 2)
	m["103"] = ge
	//m["104"] = FunctionFromPredefined(NativeMulLong, 2)
	//m["105"] = FunctionFromPredefined(NativeDivLong, 2)
	//m["106"] = FunctionFromPredefined(NativeModLong, 2)
	//m["107"] = FunctionFromPredefined(NativeFractionLong, 3)
	//
	//m["200"] = FunctionFromPredefined(NativeSizeBytes, 1)
	//m["201"] = FunctionFromPredefined(NativeTakeBytes, 2)
	//m["202"] = FunctionFromPredefined(NativeDropBytes, 2)
	//m["203"] = FunctionFromPredefined(NativeConcatBytes, 2)
	//
	//m["300"] = FunctionFromPredefined(NativeConcatStrings, 2)
	//m["303"] = FunctionFromPredefined(NativeTakeStrings, 2)
	//m["304"] = FunctionFromPredefined(NativeDropStrings, 2)
	//m["305"] = FunctionFromPredefined(NativeSizeString, 1)
	//
	//m["400"] = FunctionFromPredefined(NativeSizeList, 1)
	//m["401"] = FunctionFromPredefined(NativeGetList, 2)
	//m["410"] = FunctionFromPredefined(NativeLongToBytes, 1)
	//m["411"] = FunctionFromPredefined(NativeStringToBytes, 1)
	//m["412"] = FunctionFromPredefined(NativeBooleanToBytes, 1)
	m["420"] = longToString
	//m["421"] = FunctionFromPredefined(NativeBooleanToString, 1)
	//
	//m["500"] = FunctionFromPredefined(limitedSigVerify(0), 3)
	//m["501"] = FunctionFromPredefined(limitedKeccak256(0), 1)
	//m["502"] = FunctionFromPredefined(limitedBlake2b256(0), 1)
	//m["503"] = FunctionFromPredefined(limitedSha256(0), 1)
	//
	//m["600"] = FunctionFromPredefined(NativeToBase58, 1)
	//m["601"] = FunctionFromPredefined(NativeFromBase58, 1)
	//m["602"] = FunctionFromPredefined(NativeToBase64, 1)
	//m["603"] = FunctionFromPredefined(NativeFromBase64, 1)
	//
	//m["1000"] = FunctionFromPredefined(NativeTransactionByID, 1)
	//m["1001"] = FunctionFromPredefined(NativeTransactionHeightByID, 1)
	//m["1003"] = FunctionFromPredefined(NativeAssetBalanceV3, 2)
	//
	//m["1040"] = FunctionFromPredefined(NativeDataIntegerFromArray, 2)
	//m["1041"] = FunctionFromPredefined(NativeDataBooleanFromArray, 2)
	//m["1042"] = FunctionFromPredefined(NativeDataBinaryFromArray, 2)
	//m["1043"] = FunctionFromPredefined(NativeDataStringFromArray, 2)
	//
	//m["1050"] = FunctionFromPredefined(NativeDataIntegerFromState, 2)
	//m["1051"] = FunctionFromPredefined(NativeDataBooleanFromState, 2)
	//m["1052"] = FunctionFromPredefined(NativeDataBinaryFromState, 2)
	//m["1053"] = FunctionFromPredefined(NativeDataStringFromState, 2)
	//
	//m["1060"] = FunctionFromPredefined(NativeAddressFromRecipient, 1)
	//
	//user functions
	//m["throw"] = FunctionFromPredefined(UserThrow, 0)
	//m["addressFromString"] = FunctionFromPredefined(UserAddressFromString, 1)
	//m["!="] = FunctionFromPredefined(UserFunctionNeq, 2)
	//m["isDefined"] = FunctionFromPredefined(UserIsDefined, 1)
	//m["extract"] = FunctionFromPredefined(UserExtract, 1)
	//m["dropRightBytes"] = FunctionFromPredefined(UserDropRightBytes, 2)
	//m["takeRightBytes"] = FunctionFromPredefined(UserTakeRightBytes, 2)
	//m["takeRight"] = FunctionFromPredefined(UserTakeRightString, 2)
	//m["dropRight"] = FunctionFromPredefined(UserDropRightString, 2)
	//m["!"] = FunctionFromPredefined(UserUnaryNot, 1)
	//m["-"] = FunctionFromPredefined(UserUnaryMinus, 1)
	//
	//m["getInteger"] = FunctionFromPredefined(UserDataIntegerFromArrayByIndex, 2)
	//m["getBoolean"] = FunctionFromPredefined(UserDataBooleanFromArrayByIndex, 2)
	//m["getBinary"] = FunctionFromPredefined(UserDataBinaryFromArrayByIndex, 2)
	//m["getString"] = FunctionFromPredefined(UserDataStringFromArrayByIndex, 2)
	//
	//m["addressFromPublicKey"] = FunctionFromPredefined(UserAddressFromPublicKey, 1)
	//m["wavesBalance"] = FunctionFromPredefined(UserWavesBalanceV3, 1)
	//
	// type constructors
	//m["Address"] = FunctionFromPredefined(UserAddress, 1)
	//m["Alias"] = FunctionFromPredefined(UserAlias, 1)
	//m["DataEntry"] = FunctionFromPredefined(DataEntry, 2)
	//m["AssetPair"] = FunctionFromPredefined(AssetPair, 2)
	//m["DataTransaction"] = FunctionFromPredefined(DataTransaction, 9)
	return m
}

func functionsV3() map[string]rideFunction {
	m := functionsV12()
	//TODO: implement
	return m
}

func functionsV4() map[string]rideFunction {
	m := functionsV3()
	//TODO: implement
	return m
}

func eq(args ...rideType) (rideType, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("eq: %d is invalid number of arguments, expected 2", len(args))
	}
	return rideBoolean(args[0].eq(args[1])), nil
}

func ge(args ...rideType) (rideType, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("ge: %d is invalid number of arguments, expected 2", len(args))
	}
	return rideBoolean(args[0].ge(args[1])), nil
}

func longToString(args ...rideType) (rideType, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("longToString: %d is invalid number of arguments, expected 1", len(args))
	}
	lv, ok := args[0].(rideLong)
	if !ok {
		return nil, errors.Errorf("longToString: first argument is not a long value but '%v' of type '%T'", args[0], args[0])
	}
	return rideString(strconv.Itoa(int(lv))), nil
}
