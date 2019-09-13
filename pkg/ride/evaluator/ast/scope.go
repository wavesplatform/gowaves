package ast

import "github.com/wavesplatform/gowaves/pkg/types"

type Scope interface {
	Clone() Scope
	AddValue(name string, expr Expr)
	Value(string) (Expr, bool)
	State() types.SmartState
	Scheme() byte
	Initial() Scope
}

type ScopeImpl struct {
	parent    Scope
	variables map[string]Expr
	state     types.SmartState
	scheme    byte
}

type Callable func(Scope, Exprs) (Expr, error)

func NewScope(scheme byte, state types.SmartState, variables map[string]Expr) *ScopeImpl {
	return &ScopeImpl{
		variables: variables,
		state:     state,
		scheme:    scheme,
	}
}

func (a *ScopeImpl) Clone() Scope {
	return &ScopeImpl{
		parent: a,
		state:  a.state,
		scheme: a.scheme,
	}
}

// clone scope with only predefined variables
func (a *ScopeImpl) Initial() Scope {
	if a.parent != nil {
		return a.parent.Initial()
	}
	return a.Clone()
}

func (a *ScopeImpl) State() types.SmartState {
	return a.state
}

func (a *ScopeImpl) AddValue(name string, value Expr) {
	if a.variables == nil {
		a.variables = make(map[string]Expr)
	}
	a.variables[name] = value
}

func (a *ScopeImpl) Value(name string) (Expr, bool) {
	// first look in current scope
	if a.variables != nil {
		if v, ok := a.variables[name]; ok {
			return v, true
		}
	}

	// try find in parent
	if a.parent != nil {
		return a.parent.Value(name)
	} else {
		return nil, false
	}
}

func (a *ScopeImpl) Scheme() byte {
	return a.scheme
}

type Functions map[string]Expr

func EmptyFunctions() Functions {
	return Functions{}
}

func FunctionsV2() Functions {
	fns := make(map[string]Expr)

	fns["0"] = FunctionFromPredefined(NativeEq, 2)
	fns["1"] = FunctionFromPredefined(NativeIsInstanceOf, 2)
	fns["2"] = FunctionFromPredefined(NativeThrow, 1)

	fns["100"] = FunctionFromPredefined(NativeSumLong, 2)
	fns["101"] = FunctionFromPredefined(NativeSubLong, 2)
	fns["102"] = FunctionFromPredefined(NativeGtLong, 2)
	fns["103"] = FunctionFromPredefined(NativeGeLong, 2)
	fns["104"] = FunctionFromPredefined(NativeMulLong, 2)
	fns["105"] = FunctionFromPredefined(NativeDivLong, 2)
	fns["106"] = FunctionFromPredefined(NativeModLong, 2)
	fns["107"] = FunctionFromPredefined(NativeFractionLong, 3)

	fns["200"] = FunctionFromPredefined(NativeSizeBytes, 1)
	fns["201"] = FunctionFromPredefined(NativeTakeBytes, 2)
	fns["202"] = FunctionFromPredefined(NativeDropBytes, 2)
	fns["203"] = FunctionFromPredefined(NativeConcatBytes, 2)

	fns["300"] = FunctionFromPredefined(NativeConcatStrings, 2)
	fns["303"] = FunctionFromPredefined(NativeTakeStrings, 2)
	fns["304"] = FunctionFromPredefined(NativeDropStrings, 2)
	fns["305"] = FunctionFromPredefined(NativeSizeString, 1)

	fns["400"] = FunctionFromPredefined(NativeSizeList, 1)
	fns["401"] = FunctionFromPredefined(NativeGetList, 2)
	fns["410"] = FunctionFromPredefined(NativeLongToBytes, 1)
	fns["411"] = FunctionFromPredefined(NativeStringToBytes, 1)
	fns["412"] = FunctionFromPredefined(NativeBooleanToBytes, 1)
	fns["420"] = FunctionFromPredefined(NativeLongToString, 1)
	fns["421"] = FunctionFromPredefined(NativeBooleanToString, 1)

	fns["500"] = FunctionFromPredefined(NativeSigVerify, 3)
	fns["501"] = FunctionFromPredefined(NativeKeccak256, 1)
	fns["502"] = FunctionFromPredefined(NativeBlake2b256, 1)
	fns["503"] = FunctionFromPredefined(NativeSha256, 1)

	fns["600"] = FunctionFromPredefined(NativeToBase58, 1)
	fns["601"] = FunctionFromPredefined(NativeFromBase58, 1)
	fns["602"] = FunctionFromPredefined(NativeToBase64, 1)
	fns["603"] = FunctionFromPredefined(NativeFromBase64, 1)

	fns["1000"] = FunctionFromPredefined(NativeTransactionByID, 1)
	fns["1001"] = FunctionFromPredefined(NativeTransactionHeightByID, 1)
	fns["1003"] = FunctionFromPredefined(NativeAssetBalance, 2)

	fns["1040"] = FunctionFromPredefined(NativeDataIntegerFromArray, 2)
	fns["1041"] = FunctionFromPredefined(NativeDataBooleanFromArray, 2)
	fns["1042"] = FunctionFromPredefined(NativeDataBinaryFromArray, 2)
	fns["1043"] = FunctionFromPredefined(NativeDataStringFromArray, 2)

	fns["1050"] = FunctionFromPredefined(NativeDataIntegerFromState, 2)
	fns["1051"] = FunctionFromPredefined(NativeDataBooleanFromState, 2)
	fns["1052"] = FunctionFromPredefined(NativeDataBinaryFromState, 2)
	fns["1053"] = FunctionFromPredefined(NativeDataStringFromState, 2)

	fns["1060"] = FunctionFromPredefined(NativeAddressFromRecipient, 1)

	// user functions
	fns["throw"] = FunctionFromPredefined(UserThrow, 0)
	fns["addressFromString"] = FunctionFromPredefined(UserAddressFromString, 1)
	fns["!="] = FunctionFromPredefined(UserFunctionNeq, 2)
	fns["isDefined"] = FunctionFromPredefined(UserIsDefined, 1)
	fns["extract"] = FunctionFromPredefined(UserExtract, 1)
	fns["dropRightBytes"] = FunctionFromPredefined(UserDropRightBytes, 2)
	fns["takeRightBytes"] = FunctionFromPredefined(UserTakeRightBytes, 2)
	fns["takeRight"] = FunctionFromPredefined(UserTakeRightString, 2)
	fns["dropRight"] = FunctionFromPredefined(UserDropRightString, 2)
	fns["!"] = FunctionFromPredefined(UserUnaryNot, 1)
	fns["-"] = FunctionFromPredefined(UserUnaryMinus, 1)

	fns["getInteger"] = FunctionFromPredefined(UserDataIntegerFromArrayByIndex, 2)
	fns["getBoolean"] = FunctionFromPredefined(UserDataBooleanFromArrayByIndex, 2)
	fns["getBinary"] = FunctionFromPredefined(UserDataBinaryFromArrayByIndex, 2)
	fns["getString"] = FunctionFromPredefined(UserDataStringFromArrayByIndex, 2)

	fns["addressFromPublicKey"] = FunctionFromPredefined(UserAddressFromPublicKey, 1)
	fns["wavesBalance"] = FunctionFromPredefined(UserWavesBalance, 1)

	// type constructors
	fns["Address"] = FunctionFromPredefined(UserAddress, 1)
	fns["Alias"] = FunctionFromPredefined(UserAlias, 1)
	fns["DataEntry"] = FunctionFromPredefined(DataEntry, 2)

	return fns
}

var VarFunctionsV2 = FunctionsV2()

func FunctionsV3() Functions {
	s := FunctionsV2()
	s["108"] = FunctionFromPredefined(NativePowLong, 6)
	s["109"] = FunctionFromPredefined(NativeLogLong, 6)

	s["504"] = FunctionFromPredefined(NativeRSAVerify, 4)
	s["604"] = FunctionFromPredefined(NativeToBase16, 1)
	s["605"] = FunctionFromPredefined(NativeFromBase16, 1)
	s["700"] = FunctionFromPredefined(NativeCheckMerkleProof, 3)
	delete(s, "1000") // Native function transactionByID was disabled since v3
	s["1004"] = FunctionFromPredefined(NativeAssetInfo, 1)
	s["1005"] = FunctionFromPredefined(NativeBlockInfoByHeight, 1)
	s["1006"] = FunctionFromPredefined(NativeTransferTransactionByID, 1)
	s["1061"] = FunctionFromPredefined(NativeAddressToString, 1)
	s["1070"] = FunctionFromPredefined(NativeParseBlockHeader, 1) // RIDE v4
	s["1100"] = FunctionFromPredefined(NativeCreateList, 2)
	s["1200"] = FunctionFromPredefined(NativeBytesToUTF8String, 1)
	s["1201"] = FunctionFromPredefined(NativeBytesToLong, 1)
	s["1202"] = FunctionFromPredefined(NativeBytesToLongWithOffset, 2)
	s["1203"] = FunctionFromPredefined(NativeIndexOfSubstring, 2)
	s["1204"] = FunctionFromPredefined(NativeIndexOfSubstringWithOffset, 3)
	s["1205"] = FunctionFromPredefined(NativeSplitString, 2)
	s["1206"] = FunctionFromPredefined(NativeParseInt, 1)
	s["1207"] = FunctionFromPredefined(NativeLastIndexOfSubstring, 2)
	s["1208"] = FunctionFromPredefined(NativeLastIndexOfSubstringWithOffset, 3)

	// Constructors for simple types
	s["Ceiling"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Ceiling", CeilingExpr{}), 0)
	s["Floor"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Floor", FloorExpr{}), 0)
	s["HalfEven"] = FunctionFromPredefined(SimpleTypeConstructorFactory("HalfEven", HalfEvenExpr{}), 0)
	s["Down"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Down", DownExpr{}), 0)
	s["Up"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Up", UpExpr{}), 0)
	s["HalfUp"] = FunctionFromPredefined(SimpleTypeConstructorFactory("HalfUp", HalfUpExpr{}), 0)
	s["HalfDown"] = FunctionFromPredefined(SimpleTypeConstructorFactory("HalfDown", HalfDownExpr{}), 0)

	s["NoAlg"] = FunctionFromPredefined(SimpleTypeConstructorFactory("NoAlg", NoAlgExpr{}), 0)
	s["Md5"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Md5", MD5Expr{}), 0)
	s["Sha1"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha1", SHA1Expr{}), 0)
	s["Sha224"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha224", SHA224Expr{}), 0)
	s["Sha256"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha256", SHA256Expr{}), 0)
	s["Sha384"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha384", SHA384Expr{}), 0)
	s["Sha512"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha512", SHA512Expr{}), 0)
	s["Sha3224"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha3224", SHA3224Expr{}), 0)
	s["Sha3256"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha3256", SHA3256Expr{}), 0)
	s["Sha3384"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha3384", SHA3384Expr{}), 0)
	s["Sha3512"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Sha3512", SHA3512Expr{}), 0)

	s["Unit"] = FunctionFromPredefined(SimpleTypeConstructorFactory("Unit", Unit{}), 0)

	// New user functions
	s["@extrNative(1050)"] = FunctionFromPredefined(wrapWithExtract(NativeDataIntegerFromState, "UserDataIntegerValueFromState"), 2)
	s["@extrNative(1051)"] = FunctionFromPredefined(wrapWithExtract(NativeDataBooleanFromState, "UserDataBooleanValueFromState"), 2)
	s["@extrNative(1052)"] = FunctionFromPredefined(wrapWithExtract(NativeDataBinaryFromState, "UserDataBinaryValueFromState"), 2)
	s["@extrNative(1053)"] = FunctionFromPredefined(wrapWithExtract(NativeDataStringFromState, "UserDataStringValueFromState"), 2)
	s["@extrNative(1040)"] = FunctionFromPredefined(wrapWithExtract(NativeDataIntegerFromArray, "UserDataIntegerValueFromArray"), 2)
	s["@extrNative(1041)"] = FunctionFromPredefined(wrapWithExtract(NativeDataBooleanFromArray, "UserDataBooleanValueFromArray"), 2)
	s["@extrNative(1042)"] = FunctionFromPredefined(wrapWithExtract(NativeDataBinaryFromArray, "UserDataBinaryValueFromArray"), 2)
	s["@extrNative(1043)"] = FunctionFromPredefined(wrapWithExtract(NativeDataStringFromArray, "UserDataStringValueFromArray"), 2)
	s["@extrUser(getInteger)"] = FunctionFromPredefined(wrapWithExtract(UserDataIntegerFromArrayByIndex, "UserDataIntegerValueFromArrayByIndex"), 2)
	s["@extrUser(getBoolean)"] = FunctionFromPredefined(wrapWithExtract(UserDataBooleanFromArrayByIndex, "UserDataBooleanValueFromArrayByIndex"), 2)
	s["@extrUser(getBinary)"] = FunctionFromPredefined(wrapWithExtract(UserDataBinaryFromArrayByIndex, "UserDataBinaryValueFromArrayByIndex"), 2)
	s["@extrUser(getString)"] = FunctionFromPredefined(wrapWithExtract(UserDataStringFromArrayByIndex, "UserDataStringValueFromArrayByIndex"), 2)
	s["@extrUser(addressFromString)"] = FunctionFromPredefined(wrapWithExtract(UserAddressFromString, "UserAddressValueFromString"), 1)
	s["parseIntValue"] = FunctionFromPredefined(wrapWithExtract(NativeParseInt, "UserParseIntValue"), 1)
	s["value"] = FunctionFromPredefined(UserValue, 1)
	s["valueOrErrorMessage"] = FunctionFromPredefined(UserValueOrErrorMessage, 2)
	return s
}

func (a *Functions) Clone() *Functions {
	return a
}

func VariablesV2(tx map[string]Expr, height uint64) map[string]Expr {
	v := make(map[string]Expr)
	v["tx"] = NewObject(tx)
	v["height"] = NewLong(int64(height))
	v["Sell"] = NewSell()
	v["Buy"] = NewBuy()
	return v
}

func VariablesV3(tx map[string]Expr, height uint64) map[string]Expr {
	v := VariablesV2(tx, height)
	v["CEILING"] = CeilingExpr{}
	v["FLOOR"] = FloorExpr{}
	v["HALFEVEN"] = HalfEvenExpr{}
	v["DOWN"] = DownExpr{}
	v["UP"] = UpExpr{}
	v["HALFUP"] = HalfUpExpr{}
	v["HALFDOWN"] = HalfDownExpr{}

	v["NOALG"] = NoAlgExpr{}
	v["MD5"] = MD5Expr{}
	v["SHA1"] = SHA1Expr{}
	v["SHA224"] = SHA224Expr{}
	v["SHA256"] = SHA256Expr{}
	v["SHA384"] = SHA384Expr{}
	v["SHA512"] = SHA512Expr{}
	v["SHA3224"] = SHA3224Expr{}
	v["SHA3256"] = SHA3256Expr{}
	v["SHA3384"] = SHA3384Expr{}
	v["SHA3512"] = SHA3512Expr{}

	v["nil"] = Exprs(nil)
	v["unit"] = NewUnit()
	return v
}
