package ast

import "github.com/wavesplatform/gowaves/pkg/types"

type Scope interface {
	Clone() Scope
	AddValue(name string, expr Expr)
	FuncByShort(int16) (Callable, bool)
	Value(string) (Expr, bool)
	State() types.SmartState
	Scheme() byte
	Initial() Scope
}

type ScopeImpl struct {
	parent    Scope
	funcs     *Functions
	variables map[string]Expr
	state     types.SmartState
	scheme    byte
}

type Callable func(Scope, Exprs) (Expr, error)

func NewScope(scheme byte, state types.SmartState, f *Functions, variables map[string]Expr) *ScopeImpl {
	return &ScopeImpl{
		funcs:     f,
		variables: variables,
		state:     state,
		scheme:    scheme,
	}
}

func (a *ScopeImpl) Clone() Scope {
	return &ScopeImpl{
		funcs:  a.funcs.Clone(),
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

func (a *ScopeImpl) FuncByShort(id int16) (Callable, bool) {
	return a.funcs.GetByShort(id)
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

type Functions struct {
	native map[int16]Callable
	user   map[string]Expr
}

func EmptyFunctions() *Functions {
	return &Functions{
		native: make(map[int16]Callable),
		user:   make(map[string]Expr),
	}
}

func FunctionsV2() *Functions {
	native := make(map[int16]Callable)

	native[0] = NativeEq
	native[1] = NativeIsInstanceOf
	native[2] = NativeThrow

	native[100] = NativeSumLong
	native[101] = NativeSubLong
	native[102] = NativeGtLong
	native[103] = NativeGeLong
	native[104] = NativeMulLong
	native[105] = NativeDivLong
	native[106] = NativeModLong
	native[107] = NativeFractionLong

	native[200] = NativeSizeBytes
	native[201] = NativeTakeBytes
	native[202] = NativeDropBytes
	native[203] = NativeConcatBytes

	native[300] = NativeConcatStrings
	native[303] = NativeTakeStrings
	native[304] = NativeDropStrings
	native[305] = NativeSizeString

	native[400] = NativeSizeList
	native[401] = NativeGetList
	native[410] = NativeLongToBytes
	native[411] = NativeStringToBytes
	native[412] = NativeBooleanToBytes
	native[420] = NativeLongToString
	native[421] = NativeBooleanToString

	native[500] = NativeSigVerify
	native[501] = NativeKeccak256
	native[502] = NativeBlake2b256
	native[503] = NativeSha256

	native[600] = NativeToBase58
	native[601] = NativeFromBase58
	native[602] = NativeToBase64
	native[603] = NativeFromBase64

	native[1000] = NativeTransactionByID
	native[1001] = NativeTransactionHeightByID
	native[1003] = NativeAssetBalance

	native[1040] = NativeDataIntegerFromArray
	native[1041] = NativeDataBooleanFromArray
	native[1042] = NativeDataBinaryFromArray
	native[1043] = NativeDataStringFromArray

	native[1050] = NativeDataIntegerFromState
	native[1051] = NativeDataBooleanFromState
	native[1052] = NativeDataBinaryFromState
	native[1053] = NativeDataStringFromState

	native[1060] = NativeAddressFromRecipient

	user := make(map[string]Expr)
	user["throw"] = UserFunctionFromPredefined(UserThrow, 0)
	user["addressFromString"] = UserFunctionFromPredefined(UserAddressFromString, 1)
	user["!="] = UserFunctionFromPredefined(UserFunctionNeq, 2)
	user["isDefined"] = UserFunctionFromPredefined(UserIsDefined, 1)
	user["extract"] = UserFunctionFromPredefined(UserExtract, 1)
	user["dropRightBytes"] = UserFunctionFromPredefined(UserDropRightBytes, 2)
	user["takeRightBytes"] = UserFunctionFromPredefined(UserTakeRightBytes, 2)
	user["takeRight"] = UserFunctionFromPredefined(UserTakeRightString, 2)
	user["dropRight"] = UserFunctionFromPredefined(UserDropRightString, 2)
	user["!"] = UserFunctionFromPredefined(UserUnaryNot, 1)
	user["-"] = UserFunctionFromPredefined(UserUnaryMinus, 1)

	user["getInteger"] = UserFunctionFromPredefined(UserDataIntegerFromArrayByIndex, 2)
	user["getBoolean"] = UserFunctionFromPredefined(UserDataBooleanFromArrayByIndex, 2)
	user["getBinary"] = UserFunctionFromPredefined(UserDataBinaryFromArrayByIndex, 2)
	user["getString"] = UserFunctionFromPredefined(UserDataStringFromArrayByIndex, 2)

	user["addressFromPublicKey"] = UserFunctionFromPredefined(UserAddressFromPublicKey, 1)
	user["wavesBalance"] = UserFunctionFromPredefined(UserWavesBalance, 1)

	// type constructors
	user["Address"] = UserFunctionFromPredefined(UserAddress, 1)
	user["Alias"] = UserFunctionFromPredefined(UserAlias, 1)
	user["DataEntry"] = UserFunctionFromPredefined(DataEntry, 2)

	return &Functions{
		native: native,
		user:   user,
	}
}

var VarFunctionsV2 = FunctionsV2()

func FunctionsV3() *Functions {
	s := FunctionsV2()
	s.native[108] = NativePowLong
	s.native[109] = NativeLogLong

	s.native[504] = NativeRSAVerify
	s.native[604] = NativeToBase16
	s.native[605] = NativeFromBase16
	s.native[700] = NativeCheckMerkleProof
	s.native[1004] = NativeAssetInfo
	s.native[1005] = NativeBlockInfoByHeight
	s.native[1006] = NativeTransferTransactionByID
	s.native[1061] = NativeAddressToString
	s.native[1070] = NativeParseBlockHeader // RIDE v4
	s.native[1100] = NativeCreateList
	s.native[1200] = NativeBytesToUTF8String
	s.native[1201] = NativeBytesToLong
	s.native[1202] = NativeBytesToLongWithOffset
	s.native[1203] = NativeIndexOfSubstring
	s.native[1204] = NativeIndexOfSubstringWithOffset
	s.native[1205] = NativeSplitString
	s.native[1206] = NativeParseInt
	s.native[1207] = NativeLastIndexOfSubstring
	s.native[1208] = NativeLastIndexOfSubstringWithOffset

	// Constructors for simple types
	s.user["Ceiling"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Ceiling", CeilingExpr{}), 0)
	s.user["Floor"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Floor", FloorExpr{}), 0)
	s.user["HalfEven"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("HalfEven", HalfEvenExpr{}), 0)
	s.user["Down"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Down", DownExpr{}), 0)
	s.user["Up"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Up", UpExpr{}), 0)
	s.user["HalfUp"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("HalfUp", HalfUpExpr{}), 0)
	s.user["HalfDown"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("HalfDown", HalfDownExpr{}), 0)

	s.user["NoAlg"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("NoAlg", NoAlgExpr{}), 0)
	s.user["Md5"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Md5", MD5Expr{}), 0)
	s.user["Sha1"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha1", SHA1Expr{}), 0)
	s.user["Sha224"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha224", SHA224Expr{}), 0)
	s.user["Sha256"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha256", SHA256Expr{}), 0)
	s.user["Sha384"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha384", SHA384Expr{}), 0)
	s.user["Sha512"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha512", SHA512Expr{}), 0)
	s.user["Sha3224"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha3224", SHA3224Expr{}), 0)
	s.user["Sha3256"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha3256", SHA3256Expr{}), 0)
	s.user["Sha3384"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha3384", SHA3384Expr{}), 0)
	s.user["Sha3512"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Sha3512", SHA3512Expr{}), 0)

	s.user["Unit"] = UserFunctionFromPredefined(SimpleTypeConstructorFactory("Unit", Unit{}), 0)

	// New user functions
	s.user["@extrNative(1050)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataIntegerFromState, "UserDataIntegerValueFromState"), 2)
	s.user["@extrNative(1051)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataBooleanFromState, "UserDataBooleanValueFromState"), 2)
	s.user["@extrNative(1052)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataBinaryFromState, "UserDataBinaryValueFromState"), 2)
	s.user["@extrNative(1053)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataStringFromState, "UserDataStringValueFromState"), 2)
	s.user["@extrNative(1040)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataIntegerFromArray, "UserDataIntegerValueFromArray"), 2)
	s.user["@extrNative(1041)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataBooleanFromArray, "UserDataBooleanValueFromArray"), 2)
	s.user["@extrNative(1042)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataBinaryFromArray, "UserDataBinaryValueFromArray"), 2)
	s.user["@extrNative(1043)"] = UserFunctionFromPredefined(wrapWithExtract(NativeDataStringFromArray, "UserDataStringValueFromArray"), 2)
	s.user["@extrUser(getInteger)"] = UserFunctionFromPredefined(wrapWithExtract(UserDataIntegerFromArrayByIndex, "UserDataIntegerValueFromArrayByIndex"), 2)
	s.user["@extrUser(getBoolean)"] = UserFunctionFromPredefined(wrapWithExtract(UserDataBooleanFromArrayByIndex, "UserDataBooleanValueFromArrayByIndex"), 2)
	s.user["@extrUser(getBinary)"] = UserFunctionFromPredefined(wrapWithExtract(UserDataBinaryFromArrayByIndex, "UserDataBinaryValueFromArrayByIndex"), 2)
	s.user["@extrUser(getString)"] = UserFunctionFromPredefined(wrapWithExtract(UserDataStringFromArrayByIndex, "UserDataStringValueFromArrayByIndex"), 2)
	s.user["@extrUser(addressFromString)"] = UserFunctionFromPredefined(wrapWithExtract(UserAddressFromString, "UserAddressValueFromString"), 1)
	s.user["parseIntValue"] = UserFunctionFromPredefined(wrapWithExtract(NativeParseInt, "UserParseIntValue"), 1)
	s.user["value"] = UserFunctionFromPredefined(UserValue, 1)
	s.user["valueOrErrorMessage"] = UserFunctionFromPredefined(UserValueOrErrorMessage, 2)
	return s
}

func (a *Functions) GetByShort(id int16) (Callable, bool) {
	f, ok := a.native[id]
	return f, ok
}

func (a *Functions) UserFunctions() map[string]Expr {
	return a.user
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
