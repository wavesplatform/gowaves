package ast

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
)

type Account interface {
	Data() []proto.DataEntry
}

type Scope interface {
	Clone() Scope
	AddValue(name string, expr Expr)
	FuncByShort(int16) (Callable, bool)
	FuncByName(string) (Callable, bool)
	Value(string) (Expr, bool)
	State() mockstate.MockState
	Scheme() byte
}

type ScopeImpl struct {
	parent    Scope
	funcs     *Functions
	variables map[string]Expr
	state     mockstate.MockState
	scheme    byte
}

type Callable func(Scope, Exprs) (Expr, error)

func NewScope(scheme byte, state mockstate.MockState, f *Functions, variables map[string]Expr) *ScopeImpl {
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
	}
}

func (a *ScopeImpl) State() mockstate.MockState {
	return a.state
}

func (a *ScopeImpl) FuncByShort(id int16) (Callable, bool) {
	return a.funcs.GetByShort(id)
}

func (a *ScopeImpl) FuncByName(name string) (Callable, bool) {
	return a.funcs.GetByName(name)
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
	user   map[string]Callable
}

func EmptyFunctions() *Functions {
	return &Functions{
		native: make(map[int16]Callable),
		user:   make(map[string]Callable),
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

	native[1040] = NativeDataLongFromArray
	native[1041] = NativeDataBooleanFromArray
	native[1042] = NativeDataBinaryFromArray
	native[1043] = NativeDataStringFromArray

	native[1050] = NativeDataLongFromState
	native[1051] = NativeDataBooleanFromState
	native[1052] = NativeDataBytesFromState
	native[1053] = NativeDataStringFromState

	native[1060] = NativeAddressFromRecipient

	user := make(map[string]Callable)
	user["throw"] = UserThrow
	user["addressFromString"] = UserAddressFromString
	user["!="] = UserFunctionNeq
	user["isDefined"] = UserIsDefined
	user["extract"] = UserExtract
	user["dropRightBytes"] = UserDropRightBytes
	user["takeRightBytes"] = UserTakeRightBytes
	user["takeRight"] = UserTakeRightString
	user["dropRight"] = UserDropRightString
	user["!"] = UserUnaryNot
	user["-"] = UserUnaryMinus

	user["getInteger"] = UserDataIntegerFromArrayByIndex
	user["getBoolean"] = UserDataBooleanFromArrayByIndex
	user["getBinary"] = UserDataBinaryFromArrayByIndex
	user["getString"] = UserDataStringFromArrayByIndex

	user["addressFromPublicKey"] = UserAddressFromPublicKey
	user["wavesBalance"] = UserWavesBalance

	// type constructors
	user["Address"] = UserAddress
	user["Alias"] = UserAlias

	return &Functions{
		native: native,
		user:   user,
	}
}

func FunctionsV3() *Functions {
	s := FunctionsV2()
	s.native[108] = NativePowLong
	s.native[109] = NativeLogLong

	s.native[504] = NativeRSAVerify
	s.native[604] = NativeToBase16
	s.native[605] = NativeFromBase16
	s.native[700] = NativeCheckMerkleProof
	//TODO: native[1004] = NativeAssetInfoByID // RIDE v3
	//TODO: native[1005] = NativeBlockInfoByHeight // RIDE v3
	//TODO: native[1006] = NativeTransferTransactionByID // RIDE v3
	//TODO: native[1061] = NativeAddressToString // RIDE v3
	//TODO: native[1070] = NativeBlockHeaderFromBytes // RIDE v4
	//TODO: native[1100] = NativeCreateList // RIDE v3
	//TODO: native[1200] = NativeBytesToUTF8String // RIDE v3
	//TODO: native[1201] = NativeBytesToLong // RIDE v3
	//TODO: native[1202] = NativeBytesToLongWithOffset // RIDE v3
	//TODO: native[1203] = NativeIndexOfSubstring // RIDE v3
	//TODO: native[1204] = NativeIndexOfSubstringWithOffset // RIDE v3
	//TODO: native[1205] = NativeSplitString // RIDE v3
	//TODO: native[1206] = NativeParseInt // RIDE v3
	//TODO: native[1207] = NativeLastIndexOfSubstring // RIDE v3
	//TODO: native[1208] = NativeLastIndexOfSubstringWithOffset // RIDE v3

	// Constructors for simple types
	s.user["Ceiling"] = SimpleTypeConstructorFactory("Ceiling", CeilingExpr{})
	s.user["Floor"] = SimpleTypeConstructorFactory("Floor", FloorExpr{})
	s.user["HalfEven"] = SimpleTypeConstructorFactory("HalfEven", HalfEvenExpr{})
	s.user["Down"] = SimpleTypeConstructorFactory("Down", DownExpr{})
	s.user["Up"] = SimpleTypeConstructorFactory("Up", UpExpr{})
	s.user["HalfUp"] = SimpleTypeConstructorFactory("HalfUp", HalfUpExpr{})
	s.user["HalfDown"] = SimpleTypeConstructorFactory("HalfDown", HalfDownExpr{})

	s.user["NoAlg"] = SimpleTypeConstructorFactory("NoAlg", NoAlgExpr{})
	s.user["Md5"] = SimpleTypeConstructorFactory("Md5", MD5Expr{})
	s.user["Sha1"] = SimpleTypeConstructorFactory("Sha1", SHA1Expr{})
	s.user["Sha224"] = SimpleTypeConstructorFactory("Sha224", SHA224Expr{})
	s.user["Sha256"] = SimpleTypeConstructorFactory("Sha256", SHA256Expr{})
	s.user["Sha384"] = SimpleTypeConstructorFactory("Sha384", SHA384Expr{})
	s.user["Sha512"] = SimpleTypeConstructorFactory("Sha512", SHA512Expr{})
	s.user["Sha3224"] = SimpleTypeConstructorFactory("Sha3224", SHA3224Expr{})
	s.user["Sha3256"] = SimpleTypeConstructorFactory("Sha3256", SHA3256Expr{})
	s.user["Sha3384"] = SimpleTypeConstructorFactory("Sha3384", SHA3384Expr{})
	s.user["Sha3512"] = SimpleTypeConstructorFactory("Sha3512", SHA3512Expr{})

	return s
}

func (a *Functions) GetByShort(id int16) (Callable, bool) {
	f, ok := a.native[id]
	return f, ok
}

func (a *Functions) GetByName(name string) (Callable, bool) {
	f, ok := a.user[name]
	return f, ok
}

func (a *Functions) Clone() *Functions {
	return a
}

func VariablesV2() map[string]Expr {
	v := make(map[string]Expr)
	v["tx"] = NewObject(nil)
	v["height"] = NewLong(0)
	//TODO: add Buy and Sell predefined variables
	return v
}

func VariablesV3() map[string]Expr {
	v := VariablesV2()
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
	return v
}
