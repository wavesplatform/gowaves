package ast

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"math"
	"testing"
	"time"
)

func TestNativeSumLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeSumLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(9), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeSumLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeSubLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeSubLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(1), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeSubLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeMulLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(2)}
	rs, err := NativeMulLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(10), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeMulLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeGeLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(5)}
	rs, err := NativeGeLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeGeLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeGtLong(t *testing.T) {
	params1 := Exprs{NewLong(5), NewLong(4)}
	rs, err := NativeGtLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeGtLong(newEmptyScope(), params2)
	require.Error(t, err)
}

func TestNativeDivLong(t *testing.T) {
	params1 := Exprs{NewLong(9), NewLong(2)}
	rs, err := NativeDivLong(newEmptyScope(), params1)
	require.NoError(t, err)
	assert.Equal(t, NewLong(4), rs)

	params2 := Exprs{NewLong(5), NewBoolean(true)}
	_, err = NativeDivLong(newEmptyScope(), params2)
	require.Error(t, err)

	// zero division
	params3 := Exprs{NewLong(9), NewLong(0)}
	_, err = NativeDivLong(newEmptyScope(), params3)
	require.Error(t, err)
}

func TestUserAddressFromString(t *testing.T) {
	params1 := NewString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	rs, err := UserAddressFromString(newEmptyScope(), NewExprs(params1))
	require.NoError(t, err)
	addr, _ := NewAddressFromString("3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3")
	assert.Equal(t, addr, rs)
}

func TestNativeKeccak256(t *testing.T) {
	str := "64617461"
	data, err := hex.DecodeString(str)
	require.NoError(t, err)
	result := "8f54f1c2d0eb5771cd5bf67a6689fcd6eed9444d91a39e5ef32a9b4ae5ca14ff"
	rs, err := NativeKeccak256(newEmptyScope(), NewExprs(NewBytes(data)))
	require.NoError(t, err)

	expected, err := hex.DecodeString(result)
	require.NoError(t, err)
	assert.Equal(t, NewBytes(expected), rs)
}

func TestNativeBlake2b256(t *testing.T) {
	str := "64617461"
	data, err := hex.DecodeString(str)
	require.NoError(t, err)
	result := "a035872d6af8639ede962dfe7536b0c150b590f3234a922fb7064cd11971b58e"
	rs, err := NativeBlake2b256(newEmptyScope(), NewExprs(NewBytes(data)))
	require.NoError(t, err)

	expected, err := hex.DecodeString(result)
	require.NoError(t, err)
	assert.Equal(t, NewBytes(expected), rs)
}

func TestNativeSha256(t *testing.T) {
	data := "123"
	result := "A665A45920422F9D417E4867EFDC4FB8A04A1F3FFF1FA07E998E86F7F7A27AE3"
	rs, err := NativeSha256(newEmptyScope(), NewExprs(NewBytes([]byte(data))))
	require.NoError(t, err)

	expected, err := hex.DecodeString(result)
	require.NoError(t, err)
	assert.Equal(t, NewBytes(expected), rs)
}

func TestNativeTransactionHeightByID(t *testing.T) {
	sign, err := crypto.NewSignatureFromBase58("hVTTxvgCuezXDsZgh3rDreHzf4AULe5LB1J7zveRbBD4nz3Bzb9yJ2aXKchD4Ls3y2fvYAxnpHXx54S9ZghRx67")
	require.NoError(t, err)

	scope := newScopeWithState(&mockstate.MockStateImpl{
		TransactionsHeightByID: map[string]uint64{sign.String(): 15},
	})

	rs, err :=
		NativeTransactionHeightByID(scope, NewExprs(NewBytes(sign.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(15), rs)
}

func TestNativeTransactionByID(t *testing.T) {
	sign, err := crypto.NewSignatureFromBase58("hVTTxvgCuezXDsZgh3rDreHzf4AULe5LB1J7zveRbBD4nz3Bzb9yJ2aXKchD4Ls3y2fvYAxnpHXx54S9ZghRx67")
	require.NoError(t, err)

	seed := "abcde"
	secret, public, err := crypto.GenerateKeyPair([]byte(seed))
	require.NoError(t, err)
	sender, _ := proto.NewAddressFromPublicKey(proto.MainNetScheme, public)

	transferV1 := proto.NewUnsignedTransferV1(
		public,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		uint64(time.Now().Unix()),
		1,
		10000,
		proto.NewRecipientFromAddress(sender),
		"",
	)
	require.NoError(t, err)
	require.NoError(t, transferV1.Sign(secret))

	scope := newScopeWithState(&mockstate.MockStateImpl{
		TransactionsByID: map[string]proto.Transaction{sign.String(): transferV1},
	})

	tx, err := NativeTransactionByID(scope, NewExprs(NewBytes(sign.Bytes())))
	require.NoError(t, err)
	switch v := tx.(type) {
	case *ObjectExpr:
		addr, _ := v.Get("sender")
		expected := NewAddressFromProtoAddress(sender)
		assert.Equal(t, expected, addr)
	default:
		t.Fail()
	}
}

func TestNativeSizeBytes(t *testing.T) {
	rs, err := NativeSizeBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abc"))))
	require.NoError(t, err)
	assert.Equal(t, NewLong(3), rs)
}

func TestNativeTake(t *testing.T) {
	rs, err := NativeTakeBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abc")), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("ab")), rs)

	_, err = NativeTakeBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abc")), NewLong(4)))
	require.Error(t, err)
}

func TestNativeDropBytes(t *testing.T) {
	rs, err := NativeDropBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abcdef")), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("cdef")), rs)

	_, err = NativeDropBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abc")), NewLong(4)))
	require.Error(t, err)
}

func TestNativeConcatBytes(t *testing.T) {
	rs, err := NativeConcatBytes(newEmptyScope(), NewExprs(NewBytes([]byte("abc")), NewBytes([]byte("def"))))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("abcdef")), rs)
}

func TestNativeConcatStrings(t *testing.T) {
	rs, err := NativeConcatStrings(newEmptyScope(), NewExprs(NewString("abc"), NewString("def")))
	require.NoError(t, err)
	assert.Equal(t, NewString("abcdef"), rs)
}

func TestNativeTakeStrings(t *testing.T) {
	rs, err := NativeTakeStrings(newEmptyScope(), NewExprs(NewString("abcdef"), NewLong(3)))
	require.NoError(t, err)
	assert.Equal(t, NewString("abc"), rs)

	rs2, err := NativeTakeStrings(newEmptyScope(), NewExprs(NewString("привет"), NewLong(3)))
	require.NoError(t, err)
	assert.Equal(t, NewString("при"), rs2)
}

func TestNativeDropStrings(t *testing.T) {
	rs, err := NativeDropStrings(newEmptyScope(), NewExprs(NewString("abcdef"), NewLong(4)))
	require.NoError(t, err)
	assert.Equal(t, NewString("ef"), rs)

	rs2, err := NativeDropStrings(newEmptyScope(), NewExprs(NewString("привет"), NewLong(4)))
	require.NoError(t, err)
	assert.Equal(t, NewString("ет"), rs2)
}

func TestNativeSizeString(t *testing.T) {
	rs2, err := NativeSizeString(newEmptyScope(), NewExprs(NewString("привет")))
	require.NoError(t, err)
	assert.Equal(t, NewLong(6), rs2)
}

func TestNativeSizeList(t *testing.T) {
	rs, err := NativeSizeList(newEmptyScope(), Params(NewExprs(NewLong(1))))
	require.NoError(t, err)
	assert.Equal(t, NewLong(1), rs)
}

func TestNativeLongToBytes(t *testing.T) {
	rs, err := NativeLongToBytes(newEmptyScope(), Params(NewLong(1)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}), rs)
}

func TestNativeThrow(t *testing.T) {
	rs, err := NativeThrow(newEmptyScope(), Params(NewString("mess")))
	require.Nil(t, rs)
	assert.Equal(t, "mess", err.Error())
}

func TestNativeModLong(t *testing.T) {
	rs, err := NativeModLong(newEmptyScope(), Params(NewLong(-10), NewLong(6)))
	require.NoError(t, err)
	assert.Equal(t, NewLong(2), rs)
}

func TestModDivision(t *testing.T) {
	assert.EqualValues(t, 4, modDivision(10, 6))
	assert.EqualValues(t, 2, modDivision(-10, 6))
	assert.EqualValues(t, -2, modDivision(10, -6))
	assert.EqualValues(t, -4, modDivision(-10, -6))
}

func TestNativeFractionLong(t *testing.T) {
	// works with big integers
	rs1, err := NativeFractionLong(newEmptyScope(), Params(NewLong(math.MaxInt64), NewLong(4), NewLong(6)))
	require.NoError(t, err)
	assert.Equal(t, NewLong(6148914691236517204), rs1)

	// and works with usual integers
	rs2, err := NativeFractionLong(newEmptyScope(), NewExprs(NewLong(8), NewLong(4), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewLong(16), rs2)

	// overflow
	_, err = NativeFractionLong(newEmptyScope(), NewExprs(NewLong(math.MaxInt64), NewLong(4), NewLong(1)))
	require.Error(t, err)

	// zero division
	_, err = NativeFractionLong(newEmptyScope(), NewExprs(NewLong(math.MaxInt64), NewLong(4), NewLong(0)))
	require.Error(t, err)
}

func TestNativeStringToBytes(t *testing.T) {
	rs, err := NativeStringToBytes(newEmptyScope(), NewExprs(NewString("привет")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("привет")), rs)
}

func TestNativeBooleanToBytes(t *testing.T) {
	rs1, err := NativeBooleanToBytes(newEmptyScope(), Params(NewBoolean(true)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte{1}), rs1)
	rs2, err := NativeBooleanToBytes(newEmptyScope(), Params(NewBoolean(false)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte{0}), rs2)
}

func TestNativeLongToString(t *testing.T) {
	rs1, err := NativeLongToString(newEmptyScope(), Params(NewLong(100500)))
	require.NoError(t, err)
	assert.Equal(t, NewString("100500"), rs1)
}

func TestNativeBooleanToString(t *testing.T) {
	rs1, err := NativeBooleanToString(newEmptyScope(), Params(NewBoolean(true)))
	require.NoError(t, err)
	assert.Equal(t, NewString("true"), rs1)

	rs2, err := NativeBooleanToString(newEmptyScope(), Params(NewBoolean(false)))
	require.NoError(t, err)
	assert.Equal(t, NewString("false"), rs2)
}

func TestNativeToBase58(t *testing.T) {
	rs1, err := NativeToBase58(newEmptyScope(), Params(NewBytes([]byte("hello"))))
	require.NoError(t, err)
	assert.Equal(t, NewString("Cn8eVZg"), rs1)
}

func TestNativeFromBase58(t *testing.T) {
	rs1, err := NativeFromBase58(newEmptyScope(), Params(NewString("abcde")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]uint8{0x16, 0xa9, 0x5c, 0x99}), rs1)
}

func TestNativeFromBase64String(t *testing.T) {
	rs1, err := NativeFromBase64String(newEmptyScope(), Params(NewString("AQa3b8tH")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]uint8{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}), rs1)
}

func TestNativeToBse64String(t *testing.T) {
	rs1, err := NativeToBse64String(newEmptyScope(), Params(NewBytes([]uint8{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47})))
	require.NoError(t, err)
	assert.Equal(t, NewString("AQa3b8tH"), rs1)
}

func TestNativeAssetBalance_FromAddress(t *testing.T) {
	am := mockstate.MockAccount{
		Assets: map[string]uint64{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD": 5},
	}

	s := mockstate.MockStateImpl{
		Accounts: map[string]mockstate.Account{"3N2YHKSnQTUmka4pocTt71HwSSAiUWBcojK": &am},
	}

	addr, err := proto.NewAddressFromString("3N2YHKSnQTUmka4pocTt71HwSSAiUWBcojK")
	require.NoError(t, err)

	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	rs, err := NativeAssetBalance(newScopeWithState(s), Params(NewAddressFromProtoAddress(addr), NewBytes(d.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(5), rs)
}

func TestNativeAssetBalance_FromAlias(t *testing.T) {
	am := mockstate.MockAccount{
		Assets: map[string]uint64{"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD": 5},
	}

	s := mockstate.MockStateImpl{
		Accounts: map[string]mockstate.Account{"alias:W:test": &am},
	}

	scope := newScopeWithState(s)

	alias := proto.NewAlias(scope.Scheme(), "test")

	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	rs, err := NativeAssetBalance(scope, Params(NewAliasFromProtoAlias(*alias), NewBytes(d.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(5), rs)
}

func TestNativeDataFromArray(t *testing.T) {

	var dataEntries []proto.DataEntry
	dataEntries = append(dataEntries, &proto.IntegerDataEntry{
		Key:   "integer",
		Value: 100500,
	})
	dataEntries = append(dataEntries, &proto.BooleanDataEntry{
		Key:   "boolean",
		Value: true,
	})
	dataEntries = append(dataEntries, &proto.BinaryDataEntry{
		Key:   "binary",
		Value: []byte("hello"),
	})
	dataEntries = append(dataEntries, &proto.StringDataEntry{
		Key:   "string",
		Value: "world",
	})

	rs1, err := NativeDataLongFromArray(newEmptyScope(), Params(NewDataEntryList(dataEntries), NewString("integer")))
	require.NoError(t, err)
	assert.Equal(t, NewLong(100500), rs1)

	rs2, err := NativeDataBooleanFromArray(newEmptyScope(), Params(NewDataEntryList(dataEntries), NewString("boolean")))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs2)

	rs3, err := NativeDataStringFromArray(newEmptyScope(), Params(NewDataEntryList(dataEntries), NewString("string")))
	require.NoError(t, err)
	assert.Equal(t, NewString("world"), rs3)

	rs4, err := NativeDataBinaryFromArray(newEmptyScope(), Params(NewDataEntryList(dataEntries), NewString("binary")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("hello")), rs4)

	// test no value
	rs5, err := NativeDataBinaryFromArray(newEmptyScope(), Params(NewDataEntryList(dataEntries), NewString("unknown")))
	require.NoError(t, err)
	assert.Equal(t, Unit{}, rs5)
}

func TestNativeDataFromState(t *testing.T) {
	saddr := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := NewAddressFromString(saddr)
	require.NoError(t, err)

	var dataEntries []proto.DataEntry
	dataEntries = append(dataEntries, &proto.IntegerDataEntry{
		Key:   "integer",
		Value: 100500,
	})
	dataEntries = append(dataEntries, &proto.BooleanDataEntry{
		Key:   "boolean",
		Value: true,
	})
	dataEntries = append(dataEntries, &proto.BinaryDataEntry{
		Key:   "binary",
		Value: []byte("hello"),
	})
	dataEntries = append(dataEntries, &proto.StringDataEntry{
		Key:   "string",
		Value: "world",
	})

	am := mockstate.MockAccount{
		DataEntries: dataEntries,
	}

	s := mockstate.MockStateImpl{
		Accounts: map[string]mockstate.Account{saddr: &am},
	}

	rs1, err := NativeDataLongFromState(newScopeWithState(s), Params(addr, NewString("integer")))
	require.NoError(t, err)
	assert.Equal(t, NewLong(100500), rs1)

	rs2, err := NativeDataBooleanFromState(newScopeWithState(s), Params(addr, NewString("boolean")))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs2)

	rs3, err := NativeDataBytesFromState(newScopeWithState(s), Params(addr, NewString("binary")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("hello")), rs3)

	rs4, err := NativeDataStringFromState(newScopeWithState(s), Params(addr, NewString("string")))
	require.NoError(t, err)
	assert.Equal(t, NewString("world"), rs4)
}

func TestUserIsDefined(t *testing.T) {
	rs1, err := UserIsDefined(newEmptyScope(), Params(NewString("")))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs1)

	rs2, err := UserIsDefined(newEmptyScope(), Params(NewUnit()))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(false), rs2)
}

func TestUserExtract(t *testing.T) {
	rs1, err := UserExtract(newEmptyScope(), Params(NewString("")))
	require.NoError(t, err)
	assert.Equal(t, NewString(""), rs1)

	_, err = UserExtract(newEmptyScope(), Params(NewUnit()))
	require.EqualError(t, err, "extract() called on unit value")
}

func TestUserDropRightBytes(t *testing.T) {
	rs1, err := UserDropRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("hel")), rs1)

	_, err = UserDropRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(10)))
	require.Error(t, err)

	_, err = UserDropRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(5)))
	require.NoError(t, err)
}

func TestUserTakeRight(t *testing.T) {
	rs1, err := UserTakeRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("lo")), rs1)

	_, err = UserTakeRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(10)))
	require.Error(t, err)

	_, err = UserTakeRightBytes(newEmptyScope(), Params(NewBytes([]byte("hello")), NewLong(5)))
	require.NoError(t, err)
}

func TestUserTakeRightString(t *testing.T) {
	rs1, err := UserTakeRightString(newEmptyScope(), Params(NewString("hello"), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewString("lo"), rs1)

	_, err = UserTakeRightString(newEmptyScope(), Params(NewString("hello"), NewLong(20)))
	require.Error(t, err)
}

func TestUserDropRightString(t *testing.T) {
	rs1, err := UserDropRightString(newEmptyScope(), Params(NewString("hello"), NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewString("hel"), rs1)

	_, err = UserDropRightString(newEmptyScope(), Params(NewString("hello"), NewLong(20)))
	require.Error(t, err)
}

func TestUserUnaryMinus(t *testing.T) {
	rs1, err := UserUnaryMinus(newEmptyScope(), Params(NewLong(2)))
	require.NoError(t, err)
	assert.Equal(t, NewLong(-2), rs1)
}

func TestUserUnaryNot(t *testing.T) {
	rs1, err := UserUnaryNot(newEmptyScope(), Params(NewBoolean(true)))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(false), rs1)
}

func TestUserAddressFromPublicKey(t *testing.T) {
	s := "14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY"
	pub, err := crypto.NewPublicKeyFromBase58(s)
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	require.NoError(t, err)

	rs, err := UserAddressFromPublicKey(newEmptyScope(), Params(NewBytes(pub.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs)
}

func TestNativeAddressFromRecipient(t *testing.T) {
	a := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := proto.NewAddressFromString(a)
	require.NoError(t, err)

	r := proto.NewRecipientFromAddress(addr)

	acc := mockstate.MockAccount{
		AddressField: addr,
	}

	s := mockstate.MockStateImpl{
		Accounts: map[string]mockstate.Account{r.String(): &acc},
	}

	rs, err := NativeAddressFromRecipient(newScopeWithState(s), Params(NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(addr))))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs)
}

func TestUserAddress(t *testing.T) {
	s := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := proto.NewAddressFromString(s)
	require.NoError(t, err)

	rs1, err := UserAddress(newEmptyScope(), Params(NewBytes(addr.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs1)
}

func TestUserAlias(t *testing.T) {
	s := "alias:T:testme"
	alias, err := proto.NewAliasFromString(s)
	require.NoError(t, err)

	rs1, err := UserAlias(newEmptyScope(), Params(NewString(s)))
	require.NoError(t, err)
	assert.Equal(t, NewAliasFromProtoAlias(*alias), rs1)
}
