package ast

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
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

func TestNativeSigVerify(t *testing.T) {
	msg, err := hex.DecodeString("135212a9cf00d0a05220be7323bfa4a5ba7fc5465514007702121a9c92e46bd473062f00841af83cb7bc4b2cd58dc4d5b151244cc8293e795796835ed36822c6e09893ec991b38ada4b21a06e691afa887db4e9d7b1d2afc65ba8d2f5e6926ff53d2d44d55fa095f3fad62545c714f0f3f59e4bfe91af8")
	require.NoError(t, err)
	sig, err := hex.DecodeString("d971ec27c5bfc384804c8d8d6a2de9edc3d957b25e488e954a71ef4c4a87f5fb09cfdf6bd26cffc49d03048e8edb0c918061be158d737c2e11cc7210263efb85")
	require.NoError(t, err)
	bad, err := hex.DecodeString("44164f23a95ed2662c5b1487e8fd688be9032efa23dd2ef29b018d33f65d0043df75f3ac1d44b4bda50e8b07e0b49e2898bec80adbf7604e72ef6565bd2f8189")
	require.NoError(t, err)
	pk, err := hex.DecodeString("ba9e7203ca62efbaa49098ec408bdf8a3dfed5a7fa7c200ece40aade905e535f")
	require.NoError(t, err)

	rs, err := NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewBytes(sig), NewBytes(pk)))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs)
	rs, err = NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewBytes(bad), NewBytes(pk)))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(false), rs)

	_, err = NativeSigVerify(newEmptyScope(), nil)
	require.Error(t, err)
	_, err = NativeSigVerify(newEmptyScope(), NewExprs(NewString("BAD"), NewBytes(sig), NewBytes(pk)))
	require.Error(t, err)
	_, err = NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewString("BAD"), NewBytes(pk)))
	require.Error(t, err)
	_, err = NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewBytes(sig), NewString("BAD")))
	require.Error(t, err)
	_, err = NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewBytes(pk), NewBytes(pk)))
	require.Error(t, err)
	_, err = NativeSigVerify(newEmptyScope(), NewExprs(NewBytes(msg), NewBytes(sig), NewBytes(pk[:10])))
	require.Error(t, err)
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
	if err != nil {
		assert.Equal(t, "mess", err.Error())
	} else {
		assert.Fail(t, "No error")
	}
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

var testBytes = []byte{0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x61, 0x20, 0x73, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x74, 0x65, 0x73, 0x74}

func TestNativeToBase58(t *testing.T) {
	rs1, err := NativeToBase58(newEmptyScope(), Params(NewBytes(testBytes)))
	require.NoError(t, err)
	assert.Equal(t, NewString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM"), rs1)
}

func TestNativeFromBase58(t *testing.T) {
	rs1, err := NativeFromBase58(newEmptyScope(), Params(NewString("6gVbAXCUdsa14xdsSk2SKaNBXs271V3Mo4zjb2cvCrsM")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes(testBytes), rs1)
}

func TestNativeToBase64(t *testing.T) {
	rs1, err := NativeToBase64(newEmptyScope(), Params(NewBytes(testBytes)))
	require.NoError(t, err)
	assert.Equal(t, NewString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q="), rs1)
}

func TestNativeFromBase64(t *testing.T) {
	rs1, err := NativeFromBase64(newEmptyScope(), Params(NewString("VGhpcyBpcyBhIHNpbXBsZSBzdHJpbmcgZm9yIHRlc3Q=")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes(testBytes), rs1)
}

func TestNativeToBase16(t *testing.T) {
	rs1, err := NativeToBase16(newEmptyScope(), Params(NewBytes(testBytes)))
	require.NoError(t, err)
	assert.Equal(t, NewString("5468697320697320612073696d706c6520737472696e6720666f722074657374"), rs1)
}

func TestNativeFromBase16(t *testing.T) {
	rs1, err := NativeFromBase16(newEmptyScope(), Params(NewString("5468697320697320612073696d706c6520737472696e6720666f722074657374")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes(testBytes), rs1)
}

func TestNativeFromBase64String(t *testing.T) {
	rs1, err := NativeFromBase64(newEmptyScope(), Params(NewString("AQa3b8tH")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]uint8{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47}), rs1)
}

func TestNativeToBse64String(t *testing.T) {
	rs1, err := NativeToBase64(newEmptyScope(), Params(NewBytes([]uint8{0x1, 0x6, 0xb7, 0x6f, 0xcb, 0x47})))
	require.NoError(t, err)
	assert.Equal(t, NewString("AQa3b8tH"), rs1)
}

func TestNativeAssetBalance_FromAddress(t *testing.T) {
	s := mockstate.MockStateImpl{
		AccountsBalance: 5,
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
	s := mockstate.MockStateImpl{
		AccountsBalance: 5,
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

	t.Run("integer", func(t *testing.T) {
		s := mockstate.MockStateImpl{
			DataEntry: &proto.IntegerDataEntry{
				Key:   "integer",
				Value: 100500,
			},
		}

		rs1, err := NativeDataLongFromState(newScopeWithState(s), Params(addr, NewString("integer")))
		require.NoError(t, err)
		assert.Equal(t, NewLong(100500), rs1)
	})

	t.Run("boolean", func(t *testing.T) {

		s := mockstate.MockStateImpl{
			DataEntry: &proto.BooleanDataEntry{
				Key:   "boolean",
				Value: true,
			},
		}

		rs2, err := NativeDataBooleanFromState(newScopeWithState(s), Params(addr, NewString("boolean")))
		require.NoError(t, err)
		assert.Equal(t, NewBoolean(true), rs2)

	})
	t.Run("binary", func(t *testing.T) {

		s := mockstate.MockStateImpl{
			DataEntry: &proto.BinaryDataEntry{
				Key:   "binary",
				Value: []byte("hello"),
			},
		}

		rs3, err := NativeDataBytesFromState(newScopeWithState(s), Params(addr, NewString("binary")))
		require.NoError(t, err)
		assert.Equal(t, NewBytes([]byte("hello")), rs3)
	})
	t.Run("string", func(t *testing.T) {

		s := mockstate.MockStateImpl{
			DataEntry: &proto.StringDataEntry{
				Key:   "string",
				Value: "world",
			},
		}

		rs4, err := NativeDataStringFromState(newScopeWithState(s), Params(addr, NewString("string")))
		require.NoError(t, err)
		assert.Equal(t, NewString("world"), rs4)
	})

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

	s := mockstate.MockStateImpl{}

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

func TestNativePowLong(t *testing.T) {
	r, err := NativePowLong(newEmptyScope(), NewExprs(NewLong(12), NewLong(1), NewLong(3456), NewLong(3), NewLong(2), DownExpr{}))
	require.NoError(t, err)
	assert.Equal(t, NewLong(187), r)

	r, err = NativePowLong(newEmptyScope(), NewExprs(NewLong(12), NewLong(1), NewLong(3456), NewLong(3), NewLong(2), UpExpr{}))
	require.NoError(t, err)
	assert.Equal(t, NewLong(188), r)

	// overflow
	_, err = NativeFractionLong(newEmptyScope(), NewExprs(NewLong(math.MaxInt64), NewLong(0), NewLong(100), NewLong(0), NewLong(0), UpExpr{}))
	require.Error(t, err)
}

func TestNativeLogLong(t *testing.T) {
	r, err := NativeLogLong(newEmptyScope(), NewExprs(NewLong(16), NewLong(0), NewLong(2), NewLong(0), NewLong(0), UpExpr{}))
	require.NoError(t, err)
	assert.Equal(t, NewLong(4), r)

	r, err = NativeLogLong(newEmptyScope(), NewExprs(NewLong(100), NewLong(0), NewLong(10), NewLong(0), NewLong(0), UpExpr{}))
	require.NoError(t, err)
	assert.Equal(t, NewLong(2), r)
}

func TestNativeRSAVerify(t *testing.T) {
	pk, err := base64.StdEncoding.DecodeString("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkDg8m0bCDX7fTbBlHZm+BZIHVOfC2I4klRbjSqwFi/eCdfhGjYRYvu/frpSO0LIm0beKOUvwat6DY4dEhNt2PW3UeQvT2udRQ9VBcpwaJlLreCr837sn4fa9UG9FQFaGofSww1O9eBBjwMXeZr1jOzR9RBIwoL1TQkIkZGaDXRltEaMxtNnzotPfF3vGIZZuZX4CjiitHaSC0zlmQrEL3BDqqoLwo3jq8U3Zz8XUMyQElwufGRbZqdFCeiIs/EoHiJm8q8CVExRoxB0H/vE2uDFK/OXLGTgfwnDlrCa/qGt9Zsb8raUSz9IIHx72XB+kOXTt/GOuW7x2dJvTJIqKTwIDAQAB")
	require.NoError(t, err)
	for i, test := range []struct {
		msg string
		alg Expr
		sig string
		ok  bool
	}{
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", NoAlgExpr{}, "SjNvKuuJ8AnBjX8dIx3ums231M5AsVTIPrdonwvcH2lWqAOip8Bv3+hoYjt5jxPwtHxYylEJpJVXyL7q/uaxO8TATok1n/5gPd7ZzvuhuIpABe8Ot/MjcGmeI1Xdz6R6Mb+9QtSugXmy5zHqcqs4kpqQQfGSOwENktxPXqHZFKps9aR5rX945vjGbUV62EKeo76ItOdXMV+ZCN8M1denJTpEtl+Q29uEjaaCvsdwNPIR4JYqb56IjevhAt8kTXpfIypTvEKaeoMpbZaZDbIxtii2Qu+/6+HX4Mog4Bvid/FSj3qSIoPWs6UgqKnNLpMLoc3S2Foh7ZhedSDUvIH4eg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", MD5Expr{}, "Ab0sqqZApwpKOr/remFI5YxSpYEQfowygO31vDdlfCyFqPVg9zxgR6Vh0dMlZodD5cejEP91Jo1yPM4pB4BdyhAVe5EtbmT+ofDy5O2X3LGJbpGOMRyRL7Y2yr4kjDfJ3E7I+55OrThYgsv3taIliAgMV+3ZIqW9QGy4uxSLJaYbvSiLs5t26RHsm1f8pafT2QGZHDfn1KKRhCeYqtEcJIYbO92mXLUQQqFe4OCy4EayqhzEQblibAYJ14CHLfSrnabbRhvacy1RWkcchzYY3nJvyHznyNyBaYiGPgjVgeE2ZgPcIFwHEsCF7zLBzpS3gdbHk0OmhgI7LX9N5f2G0A==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA1Expr{}, "IzCKTx0UY7t1+GZovIdDKRxe3NUvobJ7fRzcnC5rVrUdY6hZaL5Djg5M7tKG1C19BjmgzgQEZc4oSMXU1BbNJUsggXZ7XWNSi8QAZ3bvXoN2qzF2DsFoxqb6lb6nAU2Vh+oazE0tXSfVjEiN3i7q6LoZPfSdsY8Cc6WdvIQqTqYRB1H25AWVO7I3IniR/qG+5S66yD3fzIRwo/XsFLuHIkoT4Yhj2VXwnrogXvoIG1opNAGtO/ddWxSb3Ac7zJlmLdSPMZjr6SUYH+g+eKM8H3d8fU8hLuLd/0R3JKvClbGyRZI+IfszLoMlowyt3A4hjfP8EXXDXhVX0VyBNHryoA==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA224Expr{}, "d6m/5WuSQbU2vOlFf74AS9zNRZEyuYBJ+CrLBSuqjIQdj74ewZtB32lfmBJxGQtABrPIl8cdlRE4sTugSc6Jcd8IpwNouNVeCRrwH90IlASOxt+3GlnNwSY2OTB7JOfn7zjLF2wbSMzBq0/qT+VmmpDFkcw7ibRAR8fYmBIQjHL9vH7WWILRJ+sF/JF2SUUkm1+dEEjq6Z6Xi0STDHcyTmBbq0ZFVOt8QRqxUVmIXq27laYjYpwtn+yQok7CT9ci3AyWYUbL4U+G+tMHEIlwBp13ItGOpkxprNKYnozdsuJvJM1XKSCN4fGKFJIlRgpRy06O6kZrIxAkQj2lDLLz+Q==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA256Expr{}, "ajH3CIH9T/nfrtwK3OPlPqz4CG6cz/cZXxQ/EIYJSUYVsGFft7edg/VhWC/vvIINFeJXues5z5VoRkw79p9akFnd8yjLv1O2X4tkp4v4l0raQZmVwJ/+Be8GfFkNi0vMcYCRBZqHaVMAeEdiXfOS3df20SZyN4IAOyOZhY4JB2phAPZDFjqK/wU1hDL1JXl1v7xAkUeMSk+Sbpmw9XqaI/ntZ4t+VDwWAqs+aVKs65X5OKXMDLSNZZLocR6uul55n74DrmHn7VojYy4LQGDKMCAu9N/nome2vvZRmETXOZUX9zHGXuuQGGNuG+r+BiMDRTHVRIogGbjfMzWQMBwLgw==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA384Expr{}, "UvoU7qoOFUmKB1P+mX2ddbPILfY0+9eLk3wtahkCPrWsnI4Bwf9yihi88erJNKyWbdhlYP7dVCcYBHOxCyDVuyoLSERimLrwoRD7aFKcwQdtQqIFInbxCenPOMS1QofjVAE0x1Vy+r6n9uh8hzKsDAP59zX2QE53BVZm0hXtRYykKxrm1hWxZdsQ90nncZ4gxb9Gp9M2TRiw1NFaRWungbbV5py64akqC9bJlLKBm5OXWkIrmoEubNJpJORo5IYS5c0Mi4f6nVn9l3UTCKP0lbjTc9LJPt8/UTASiQseaN8KfJTvRwHJkOOVIT0FFk96nBfo+lH1nCO8UW7m8n9Xvg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA512Expr{}, "cSV8v78EUUxnV9Z69jmsffjGfmtY5xVQt2W5i2MHZSIM9MQWhPdPTRGT4FmgfeyJZLn2AFNfBA61eR40PeSOyuSLGgrUuUERZEoYxdyl/9KQ7D9NT5K8JRBTtHowm5/zD7qhCPR+bJ4NiD9pRxTZb7MvmBRdJ0jeKRZYTBXTS6FULjxaEGB09Xr/gPQ7i0yGWjqYj52LzkLOErnTTPzTvhQssOmFU1mrQxFOqPFo++YYd48OLIMP0p4q3Swbxx+Em1PpisDRKW5i58UhIEPdveGyGgd3BDgTBAQ8rSkUIPQFgVtgDpgLJaTFvuT1E6v5xNzhS52mi7PhhMgeX1KIVg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA3224Expr{}, "L3jASa1P4HJ1XpnpZ3+ZfGxUEA20ApIXiiBWBUU9AoBkJIDx9WP1IjEQOR+4nkguqSvw7SXggH4YYzePwyOxiE1kZLM7U20tXZp/oJ/TqZVrcaMtiHpxWZBYZvTHCnTRjktflXy6Mxr6HVDuaVJaXVLrX6tPcqdw8/e/Cs7vcPdZdVCBGY4/LlQ46HUZQrEOApdCwcER8l3Bz2v7toTLjAnIGEbINuJ7+ye4zksw42WZG8eK2EvjOO8EylPbtWNmoqsED9O81y/HvDAY8419U9XUd/HOd7weKGNOYGZ+S3Rh0bPr7GvKQS5GvGWSxFPq3zmKyzBF7rXqBvv5vzBQ6Q==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA3256Expr{}, "GsYnxcmQOOAZthDfPjKvU1z+F7SUKGRfpNiWNpjoj6Vf6vdbP8fk9votEvVyXWd13lHZgv2lgaPG5Bd/I8Yt+/H8GPhcr/M7H0/eiZ/1yWag7O0SDdQnOAYINVGaogjuI9GdmSt33BkrPaXWjt+Li1UggT4Zgj8M2uEFvkwkpM1XDHXZChM8wHi8RNHOOfbqcPomm9qai2B1kSlw6eVjaZEEJ3SKuMdvzcsEP1P/P3pOz3/7j4uSXR9T87U0nlY8n1QXBkfMc5LggnoX5XlEvTF7jT8vsSNBYXgpBcQ3farQxdAt+qXhnFj4dttZjPMvFQHDCUxgW4zcubLmcB8/rg==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA3384Expr{}, "X7rTLJTY/ohbnOLG8hF9QqAPDzi5KCNxn1J3vQrslvTSCaNsQeI/CsVvmlusCfOqx5dI+X9cqQWLedHpxiMCbY3d+8OHKuIBd1Bs6oQuTNCnCVs9p/cyxiP2ZTbdZo5nACMW0F6DGnkLXGA1IPEBpKHTFCjhwZY+KHIwadLbtYOjqH0FfAuXytEA21IDgZIRvh0GdgbDQmzt88EPwcoUxSv+UQ99/5FMsedhrgS/fMmupmAG+DnX82xKGSRNtFe73gokrPEsXK0ldWsJhnIcTUCHXalFvYQo4HrYE8g3XBpuLC7iqHtngtk5dIZyv7nA7oT/H79OsXYXxCp8bMMs4A==", true},
		{"dj06WSU3LW53OGM9flBEODZTcSZFcXxVdiB3Y0lxUyUufmdVLnxwYXEkPXF6WH5EKCd1Kw==", SHA3512Expr{}, "bEJA5Ktjst5WLugaWh81QG31PzpJkpFkLkguiAkhEZKFWS/QRsK9Um6MHliLYqzVc3w/EvKVZkfCqLuANwHai2nuYplUwQYyBTdmIb/LuxIvuW0fL3ehajblDyQ2WhQrMBbiPgmgl6DeyeTFPqBSJSkIgT63A/J2yEUWN8iBXeqy80I8ulpHAT6NBfY/ThqSlpJbLuSN761LOkJhM3s2YxUg2O2ZZ/6DT4EnVN51vqioHfPqRxtWHCiTSV+/vXHD7UdiSwYsQC9432FtDpgsN5Fn0ndASUaMpsrpg5EgUk+rak4WwfgG3SZ1MRwBuE4iG9dk4w6tek48L32+sgqSpQ==", true},

		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", NoAlgExpr{}, "UpuBmo7cfUhjIcaN4A0kwciZWZCp3dYqsxLT4uZkJ8t+yqxDkr5BIiGTG7lbSHEqGZd6aIYWgpoOfvGUt5bgISYWysriFjMHI6FH0ObNPjj+ORyrPAzT1KTPzq5UkwC18VhmK1ZwTGtPfVPTjUagH5YRYHFD0c8uztt4QUIU3GB78l3ScjvYNpdiCsZAxcNFFF/wTfhALMr6KQwYGiWYAQCqzfErK70uqV6F9tZYs1JsZpN3y3OCAboZBzg1QvwBfzhttVwhmGNQrgYaMZmHFwyxzDz5abD/w3bpn2N7OGRApFQPXZLd74nI5H3xJS/9zW45cyv+qdPnMC5sP64epQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", MD5Expr{}, "hXYw1IaK6N0WVIOtzBOpZzaEQi/GW6CQaLW7mDYd1B1EnclE7Yd2wCVvmBs/DYQl+qtL4K4EnR0eQoI54L7S7m/0obN7tRz16f0ObLpGmra5JNlTifJRwLfz8ABoqecm271YOD1cDOScGcoEjC9ZTNJnBMCkHuAxsosk4WrxuOwrQ8cmBIpKq0rG88oHVMNlC8jT/d9ThIE5xxoLZF7Wek6mOhiB8vXhawXtd47SS4JSnAZg5oCuW+CHrlUy/CVy/IS7fvwAa/U/Sodg4pbHX/UKPSPBUCTeUIUDfiYyOBMbcL9WdgcdGFrHh7lnmzd+9reBDRk0aStl4klpe1WFDQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA1Expr{}, "KcGWAnsvh2uZbmeedd4dsq+MznQwmZEQ3VO3/HyW4+RMGfBemv0LYCjxMHqs6ztag7aJm/7kL+Rq+9YUol9KsnTx8HuwdDeBtzPBf4HrVKfcvxO20KRDmufq1B6Xy6QLN9dWSDnxjTxl0TFO9s/kbG9fdat84LP5Tl3EfEVA2Nm+lz97dt+foocz8iWWYVnd7g8yVkTB8iW8LPveW/mJvG1q5Agb4mfZIkqkptWtsbsfENBW7je3e/X1b4weJVGTuGN7CYImgMCzUpWpuhHcHs67EqMdFlc01i4w26oDD6WhxwTl+zgu7nA6/cjW+9qFhgPwDJFZM8hg7s4QtpsX1Q==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA224Expr{}, "g9n05IPksL54sQVEx1hd29mhFQ/Qb4ecNIZtm7Sk1c4O8CI0CwXfRRNL59Gk+V9oYk/14jCmgpdC0QdUjpjlEjgV7c7SjgIw7AEWP22+sLlBXpNI5uZ9stGep8aKm3fAeBjmEV3xmfvSuxxvJNC2gy0I4jtkGugrVpxul/euEhzFwWqUbbSG8fRizEsn8rUBLdsHMC9sH0rNq28UmuREgHiljNYK3G+PFMYOsgD/2u8YvgDy1vu59LOKX/2gNDmxELaPv4GZie0OmitEP4y5oufF0O4MZMtWEK1FACQvZoaZVVOPhZPwOaswauvGO7SIFSRzPLGQjORlsr+G4ZuTIg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA256Expr{}, "DLJzXp0uFISoTFT5n8h914YXEHhAsqSv5UAP52YOOWueJugYchwMFXFP+joFE2KetmF5D7htnEZFAI6j1UShuaTlZzmsrybmLOVOSvk2TYApbj1Rus1ZkRIchjDepvrNFOz6K3uE4PZ47uF6zjX1K5kDN+bD8nWULDZvx5p4P4xWGAF4Y2Aczce6yZRDq6cwTrMA4xCJr21cDlVS7URpdfemDweLIXY9NXU0PcbKc6tkL1LD0ZDRtA/3DAzqy/ae1ObyPE14yC+6+++aXYR5qIOE6CqFb8sd+WLJgbIazgfJ5unIM6kcMMl4UzpeNHhekD4gjfw/r/XWCMsjSq/Hlg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA384Expr{}, "VvgkOdf62PI9YTUT2VydTdYu9JUdag4UXJiG0gjg3z1xQf+651quqLINB0FC2yN4WL+xZhdlsSuoOVcug0FtM2wxMdWfSpMfSpGerG/u1nsc13MxRdyZLQkNOi8enxowxZGvmNFdOgppQaq9LD9a2ni2rWrQ1Fl+PWAfBIlv23PtQqM9uPJdw+IZTW/5N74TOPWhYMf0sa3oFuTjKr6S76pDKLPxfOzwrXu0oBH1g+CG8wIhxAt13khr2mtJIpu06biEaR/rKp7nBtdAiyDFB1CyNnPozAd0UcJEwXfL1k3+bR4hOknJ6D8BaqRNovICAl1knjf+ZWmt65rVeJX+CQ==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA512Expr{}, "GW4XYgmm8H8e+GWRpTcjouTD2l2oub43iT78fCkraobK+/tzWDAE8nxI05U2/9GHXHC68qLG2SdLyauXJA9YmAQBL/2Yh285YgBa5uSsBaswxuHxf82IxQ73nOj9Ek4zhi8Z32BSc46V2Kn9HFQdI3xMnbAQ1Cz+/uwfA1FeEyH3Q3sVNaE9IZheqFVopIVRV+jcma43fAPNls6ZCavQfv8MAdFsY+8SfhiifjeF+yH3vYKDWX5aG3qfFG15RTavUX2fV66OCLYhG0sGdyuirsn5cpbhVs2G+Pt1hkbs5hF59vTbLiDC+fU2gayTA4odImuyaKl35H5NO8t2h0JnoA==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA3224Expr{}, "PjKn0VMEyzqXwf/zTgbZfrCR3XSfHABwEY+cK+EURwScWxdnsQ8B/KFrmg6U1a5vj5DfdI7x2luHTUi3/UAhKvZiHAxCE+AT4o4QtIXKXn425fikQz8RyrFVAYcMEJIHOzGzaclVQaAuNKMQM44peIHxFVlGRL1ZuFzdlwPWSXTDT/LMIFxrH3IOSNiDnXPZxzjLIoC0TVVZLgNVJmypdLd9TYM5FB6mg/loBd9EuIbOLDVhZXrUuJfAk28ojhdYZWM+CLFh09UbByTZhYLT/6vs8xakA45+84GjAT5VZQOzLK7uR4OAzzMLYXpUTkZHa7x7+nnWjEn2zzVjrBV9SA==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA3256Expr{}, "OXVKJwtSoenRmwizPtpjh3sCNmOpU1tnXUnyzl+PEI1P9Rx20GkxkIXlysFT2WdbPn/HsfGMwGJW7YhrVkDXy4uAQxUxSgQouvfZoqGSPp1NtM8iVJOGyKiepgB3GxRzQsev2G8Ik47eNkEDVQa47ct9j198Wvnkf88yjSkK0KxR057MWAi20ipNLirW4ZHDAf1giv68mniKfKxsPWahOA/7JYkv18sxcsISQqRXM8nGI1UuSLt9ER7kIzyAk2mgPCiVlj0hoPGUytmbiUqvEM4QaJfCpR0wVO4f/fob6jwKkGT6wbtia+5xCD7bESIHH8ISDrdexZ01QyNP2r4enw==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA3384Expr{}, "BagFS/QgaVFTKGKpI+eMh+nMXCpI33y8jmatR6ap4fVPHtWY5+63vku3Q9uzr+4XPDclhNK3rtf+r6duZ0y4GU6M9bJuiYWPEYsq/M/M2BQ0pZVqBzYbCps2vDucaehOWS6ivU4Y9tfq+q1VOgZDZYzh9XiWfBL6pL1eIuPk/RMB11tcD91gpa0hKCD5yRzcHxmF+OVqdnyr9RT79TnR8yQ8Zf7qwBws/bPqMwvEmQsssK67wA+3vTrx8Gqgq1RfYqvIjY2llqrkeohld3O75wHAtbUFMXu8HbI4+fq1Jp3Jr/riVCScIQNv2TyPnPcWO0yfqCj+D86LGYoHoEOXrg==", true},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA3512Expr{}, "cSsxjrYkwfagdcwmA+5emRGspA6132BE/zU/QiG0pXOcaJCFE/DQaz0zPFUv/+D4BBdTx/7T/fUKFA4b3oU9KQ3RvUWaUGruwURsQ10rbmVleQdh8eODSuW38r9Vf2n/qq6VvE/2LBTM8Kamd3/czE/5RAJyCcywFmOKMKkkV96asZlb/bBeBtRSz8ZDpbyGbjm2k/cC5sxuEYgR6X1veH0wmANIsrM04+Dj6AZ4LtpUfG7hNCDUpiONmeO5KpBGvN+3bHwxuNXz311CtpJZcsr5ONvtD4l7vPv7ggQB+C1x9VvZXuJaieyk8Gm5F4oGXXfgmKsve6vAlfonpl4pmg==", true},

		{"Z0gxI00zZzNkMmkjYFxCXDg0K2Yhek9Se0hTRnR3cypSMjQmWUc=", NoAlgExpr{}, "SjNvKuuJ8AnBjX8dIx3ums231M5AsVTIPrdonwvcH2lWqAOip8Bv3+hoYjt5jxPwtHxYylEJpJVXyL7q/uaxO8TATok1n/5gPd7ZzvuhuIpABe8Ot/MjcGmeI1Xdz6R6Mb+9QtSugXmy5zHqcqs4kpqQQfGSOwENktxPXqHZFKps9aR5rX945vjGbUV62EKeo76ItOdXMV+ZCN8M1denJTpEtl+Q29uEjaaCvsdwNPIR4JYqb56IjevhAt8kTXpfIypTvEKaeoMpbZaZDbIxtii2Qu+/6+HX4Mog4Bvid/FSj3qSIoPWs6UgqKnNLpMLoc3S2Foh7ZhedSDUvIH4eg==", false},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA3512Expr{}, "hXYw1IaK6N0WVIOtzBOpZzaEQi/GW6CQaLW7mDYd1B1EnclE7Yd2wCVvmBs/DYQl+qtL4K4EnR0eQoI54L7S7m/0obN7tRz16f0ObLpGmra5JNlTifJRwLfz8ABoqecm271YOD1cDOScGcoEjC9ZTNJnBMCkHuAxsosk4WrxuOwrQ8cmBIpKq0rG88oHVMNlC8jT/d9ThIE5xxoLZF7Wek6mOhiB8vXhawXtd47SS4JSnAZg5oCuW+CHrlUy/CVy/IS7fvwAa/U/Sodg4pbHX/UKPSPBUCTeUIUDfiYyOBMbcL9WdgcdGFrHh7lnmzd+9reBDRk0aStl4klpe1WFDQ==", false},
		{"REIiN2hDQUxIJVQzdk1zQSpXclRRelExVWd+YGQoOyx0KHduPzFmcU8zUWosWiA7aFloOWplclAxPCU=", SHA512Expr{}, "cSV8v78EUUxnV9Z69jmsffjGfmtY5xVQt2W5i2MHZSIM9MQWhPdPTRGT4FmgfeyJZLn2AFNfBA61eR40PeSOyuSLGgrUuUERZEoYxdyl/9KQ7D9NT5K8JRBTtHowm5/zD7qhCPR+bJ4NiD9pRxTZb7MvmBRdJ0jeKRZYTBXTS6FULjxaEGB09Xr/gPQ7i0yGWjqYj52LzkLOErnTTPzTvhQssOmFU1mrQxFOqPFo++YYd48OLIMP0p4q3Swbxx+Em1PpisDRKW5i58UhIEPdveGyGgd3BDgTBAQ8rSkUIPQFgVtgDpgLJaTFvuT1E6v5xNzhS52mi7PhhMgeX1KIVg==", false},
	} {
		msg, err := base64.StdEncoding.DecodeString(test.msg)
		require.NoError(t, err)
		sig, err := base64.StdEncoding.DecodeString(test.sig)
		require.NoErrorf(t, err, "#%d", i)
		r, err := NativeRSAVerify(newEmptyScope(), NewExprs(test.alg, NewBytes(msg), NewBytes(sig), NewBytes(pk)))
		require.NoErrorf(t, err, "#%d", i)
		assert.Equalf(t, NewBoolean(test.ok), r, "#%d", i)
	}
}

func TestNativeCheckMerkleProof(t *testing.T) {
	for _, test := range []struct {
		root   string
		proof  string
		leaf   string
		result bool
	}{
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACCP8jyg8Rv62mE4IMD4FGATnUXEIoCIK0LMoQCjAGpl5AEg16lhBiAz+xB8hwUs8U7dTJeGmJQyWVfXmHqzA+b2YuUBICJEors9RDiMZNeWp2yIlJrpf/a4rZxTvI7yIx3D5pihACAaVrwYIveDbOb3uE+Hj1w+Tl0vornHqPT9pCja/TmfPgAgxGoHWeIYY3RDkfAyYD99LA6OXdiXaB9a86EifTMS728AINbkCaDKCXEc5i61+c3ewBPFoCCYMCyvIrDbmHAThKt4ACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAdIQ==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACDdSC04SpOqrUb7PbWs5NaLSSm/k6d1eG0MgFwTDEeJXAAg0iC2Dfqsu4tJUQt+xiDjvHyxUVu664rKruVL8zs6c60AIKLhp/AFQkokTe/NMQnKFL5eTMvDlFejApmJxPY6Rp8XACAWrdgB8DwvPA8D04E9HgUjhKghAn5aqtZnuKcmpLHztQAgd2OG15WYz90r1WipgXwjdq9WhvMIAtvGlm6E3WYY12oAIJXPPVIdbwOTdUJvCgMI4iape2gvR55vsrO2OmJJtZUNASAya23YyBl+EpKytL9+7cPdkeMMWSjk0Bc0GNnqIisofQ==", "AAAc6w==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ASADLSXbJGHQ7MMNaAqIfuLAwkvd7pQNnSQKcRnd3TYA0gAgNqksHYDS1xq5mKOpcWhxdM9KtzAJwVlJ8RECYsm9PMkAIEYOaapf0SZM4wZS8nZ95byib0SgjBLy1XG676X6lvoAASBOVhj3XzjWhqziBwKr/2M6v9VYF026vuWwXieZWMUdSwEgPqfL+ywsEjtOpywTh+k4zz23LGD2KGWHqfJvD8/9WdgBICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAc+w==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACBlQ+wlERW7AiK0dPotu7wLCCaMcH+X2D9XEU+D8TSNbwEgld8vUreEqWpiFo0nMwUsiP6LPhi8XWpV6Gge/3edo5MBIFCGuyg86lVn9ga7hNacZPBNd6T5gtMk+5OWpO8HthAmASDPIhoSPwQ9YL5aa+S6MjaLNe74dY3/Mq/OrpP7C46/8wAg1FSDEXwBdMgQkmK245kByRV39HfsgpmTdbbYd85GqI0BICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAIVw==", true},
		{"eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=", "ACBlQ+wlERW7AiK0dPotu7wLCCaMcH+X2D9XEU+D8TSNbwEgld8vUreEqWpiFo0nMwUsiP6LPhi8XWpV6Gge/3edo5MBIFCGuyg86lVn9ga7hNacZPBNd6T5gtMk+5OWpO8HthAmASDPIhoSPwQ9YL5aa+S6MjaLNe74dY3/Mq/OrpP7C46/8wAg1FSDEXwBdMgQkmK245kByRV39HfsgpmTdbbYd85GqI0BICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw==", "AAAdIQ==", false},
		{"AYzKgOs9ARx/ulwB5wBMAAsB//8Aj381Wv8lvRA8gMR/owBwlU8BsQD//7jAnABQ", "ACCtPAMekYsdrprYYtydmNgluQzuW4v8vw2V96ufptzLRAEgkZVHs/yAFKm+dzB6zGol3RqipV9n8J5tkgiA/xGxfIUBIIWgSXngwWlUvpTBVbUM9D2zGEcaLio1PlZNAgkUcpgtASBIvie1RD4kOXIEWFHyWKxGyXR+NAr1r/GX5huq/HOV+gAgHdWZ4xwPTlrgQjIL1M0aOephVd9bOEK4nO08qmyR54oAIJFT7UAb6kacEYQPYORHoMEUwF6hhVbuI3RBPcsMyg9SASCNjzIYs57ugoE56TuTjnSbtkKnJL2c0qxZ/NxEfVAf4w==", "AAASIA==", false},
		{"", "ACBx7RO4K2tuSrrQ+OG3jn8uAT2qKUlxAR1bEz/ucQEsWgAgLFOaa1LHOwhqzFou9Tece3AUeC0izlUraXyfAxnyLGMBIG/cdbO2OvahmTl/38TlRqUKZEhygqlov1KuxYPDLnPhACBUIRPanY7B4wSCGIQr8rifqw1PYIUwJB9Xj/ZFWpSRzwAgTzGXR+KVcknm5jJzJxZocqdtF14Hd8nJliISmI8lrLsAIDwdXWHBoJDzVc31XmVUOPJjgf4oezXhydg8W5nPU5NgACCVh+rJdfzMBUxlzl5N+EJ07X6/REWE8jmB4v319R0L9Q==", "AAAkig==", false},
	} {
		root, err := base64.StdEncoding.DecodeString(test.root)
		require.NoError(t, err)
		proof, err := base64.StdEncoding.DecodeString(test.proof)
		require.NoError(t, err)
		leaf, err := base64.StdEncoding.DecodeString(test.leaf)
		require.NoError(t, err)
		r, err := NativeCheckMerkleProof(newEmptyScope(), NewExprs(NewBytes(root), NewBytes(proof), NewBytes(leaf)))
		require.NoError(t, err)
		assert.Equal(t, NewBoolean(test.result), r)
	}
}

func TestNativeAddressToString(t *testing.T) {
	addr, err := proto.NewAddressFromString("3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb")
	require.NoError(t, err)
	for _, test := range []struct {
		expressions Exprs
		str         string
		error       bool
		result      bool
	}{
		{NewExprs(AddressExpr(addr)), "3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb", false, true},
		{NewExprs(AddressExpr(addr)), "3N2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb", false, false},
		{NewExprs(NewString("3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb")), "3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb", true, false},
		{NewExprs(), "3P2HNUd5VUPLMQkJmctTPEeeHumiPN2GkTb", true, false},
	} {
		r, err := NativeAddressToString(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		s, ok := r.(*StringExpr)
		assert.True(t, ok)
		assert.Equal(t, test.result, test.str == s.Value)
	}
}

func TestNativeBytesToUTF8String(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		err         bool
		result      Expr
	}{
		//{NewExprs(NewBytes([]byte("blah-blah-blah"))), false, NewString("blah-blah-blah")},
		//{NewExprs(NewBytes([]byte("blah-blah-blah")), NewString("a-a-a-a")), true, NewString("blah-blah-blah")},
		{NewExprs(NewString("blah-blah-blah")), true, NewString("blah-blah-blah")},
	} {
		r, err := NativeBytesToUTF8String(newEmptyScope(), test.expressions)
		if test.err {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func b(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func TestNativeBytesToLong(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewBytes(b(123456))), false, NewLong(123456)},
		{NewExprs(NewBytes(b(-123456))), false, NewLong(-123456)},
		{NewExprs(NewBytes(b(math.MaxInt64))), false, NewLong(math.MaxInt64)},
		{NewExprs(NewBytes(b(math.MinInt64))), false, NewLong(math.MinInt64)},
		{NewExprs(NewBytes(append(b(0), []byte{1, 2, 3, 4, 5}...))), false, NewLong(0)},
		{NewExprs(), true, NewLong(0)},
		{NewExprs(NewBytes(b(12345)), NewString("blah")), true, NewLong(0)},
		{NewExprs(NewBytes([]byte{0, 1, 2, 3, 4, 5})), true, NewLong(0)},
	} {
		r, err := NativeBytesToLong(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func in(a, b []byte, p int) []byte {
	r := make([]byte, len(a))
	copy(r, a)
	copy(r[p:], b)
	return r
}

func TestNativeBytesToLongWithOffset(t *testing.T) {
	arr := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	b := func(v int64) []byte {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v))
		return buf
	}
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewBytes(b(123456)), NewLong(0)), false, NewLong(123456)},
		{NewExprs(NewBytes(b(-123456)), NewLong(0)), false, NewLong(-123456)},
		{NewExprs(NewBytes(in(arr, b(math.MaxInt64), 3)), NewLong(3)), false, NewLong(math.MaxInt64)},
		{NewExprs(NewBytes(in(arr, b(math.MinInt64), 6)), NewLong(6)), false, NewLong(math.MinInt64)},
		{NewExprs(), true, NewLong(0)},
		{NewExprs(NewBytes(b(12345)), NewString("blah")), true, NewLong(0)},
		{NewExprs(NewBytes([]byte{0, 1, 2, 3, 4, 5})), true, NewLong(0)},
		{NewExprs(NewBytes(in(arr, b(math.MinInt64), 6)), NewLong(16)), true, NewLong(0)},
	} {
		r, err := NativeBytesToLongWithOffset(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeIndexOfSubstring(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("brown")), false, NewLong(6)},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("cafe")), false, NewUnit()},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
	} {
		r, err := NativeIndexOfSubstring(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeIndexOfSubstringWithOffset(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("brown"), NewLong(0)), false, NewLong(6)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(10)), false, NewLong(25)},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("brown"), NewLong(10)), false, NewUnit()},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("fox"), NewLong(1000)), false, NewUnit()},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1), NewString("xxx")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewString("xxx"), NewString("0")), true, NewUnit()},
	} {
		r, err := NativeIndexOfSubstringWithOffset(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeSplitString(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("abcdefg"), NewString("")), false, NewExprs(NewString("a"), NewString("b"), NewString("c"), NewString("d"), NewString("e"), NewString("f"), NewString("g"))},
		{NewExprs(NewString("one two three four"), NewString(" ")), false, NewExprs(NewString("one"), NewString("two"), NewString("three"), NewString("four"))},
		{NewExprs(), true, NewExprs()},
		{NewExprs(NewString("blah-blah-blah")), true, NewExprs()},
		{NewExprs(NewLong(0), NewString("one two three four")), true, NewExprs()},
		{NewExprs(NewString("one two three four"), NewLong(0)), true, NewExprs()},
	} {
		r, err := NativeSplitString(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeParseInt(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("123345")), false, NewLong(123345)},
		{NewExprs(NewString("0")), false, NewLong(0)},
		{NewExprs(NewString(fmt.Sprint(math.MaxInt64))), false, NewLong(math.MaxInt64)},
		{NewExprs(NewString(fmt.Sprint(math.MinInt64))), false, NewLong(math.MinInt64)},
		{NewExprs(NewString("")), false, NewUnit()},
		{NewExprs(NewString("abcd")), false, NewUnit()},
		{NewExprs(NewString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")), false, NewUnit()},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
		{NewExprs(NewLong(1)), true, NewUnit()},
	} {
		r, err := NativeParseInt(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestUserParseIntValue(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("123345")), false, NewLong(123345)},
		{NewExprs(NewString("0")), false, NewLong(0)},
		{NewExprs(NewString(fmt.Sprint(math.MaxInt64))), false, NewLong(math.MaxInt64)},
		{NewExprs(NewString(fmt.Sprint(math.MinInt64))), false, NewLong(math.MinInt64)},
		{NewExprs(NewString("")), true, NewUnit()},
		{NewExprs(NewString("abcd")), true, NewUnit()},
		{NewExprs(NewString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")), true, NewUnit()},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
		{NewExprs(NewLong(1)), true, NewUnit()},
	} {
		r, err := UserParseIntValue(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeLastIndexOfSubstring(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe")), false, NewLong(25)},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("cafe")), false, NewUnit()},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
	} {
		r, err := NativeLastIndexOfSubstring(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestNativeLastIndexOfSubstringWithOffset(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		result      Expr
	}{
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(30)), false, NewLong(25)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(25)), false, NewLong(25)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(10)), false, NewLong(5)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(5)), false, NewLong(5)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(4)), false, NewUnit()},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(0)), false, NewUnit()},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("bebe"), NewLong(-2)), false, NewUnit()},
		{NewExprs(NewString("aaa"), NewString("a"), NewLong(0)), false, NewLong(0)},
		{NewExprs(NewString("aaa"), NewString("b"), NewLong(0)), false, NewUnit()},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("dead"), NewLong(11)), false, NewLong(10)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("dead"), NewLong(10)), false, NewLong(10)},
		{NewExprs(NewString("cafe bebe dead beef cafe bebe"), NewString("dead"), NewLong(9)), false, NewUnit()},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("brown"), NewLong(12)), false, NewLong(6)},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("fox"), NewLong(14)), false, NewLong(12)},
		{NewExprs(NewString("quick brown fox jumps over the lazy dog"), NewString("fox"), NewLong(13)), false, NewLong(12)},
		{NewExprs(), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1), NewString("xxx")), true, NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewString("xxx"), NewString("0")), true, NewUnit()},
	} {
		r, err := NativeLastIndexOfSubstringWithOffset(newEmptyScope(), test.expressions)
		if test.error {
			assert.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestUserValue(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		message     string
		result      Expr
	}{
		{NewExprs(NewString("123345")), false, "", NewString("123345")},
		{NewExprs(NewLong(1)), false, "", NewLong(1)},
		{NewExprs(NewUnit()), true, "Explicit script termination", NewUnit()},
		{NewExprs(), true, "UserValue: invalid number of parameters, expected 1, received 0", NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewLong(1)), true, "UserValue: invalid number of parameters, expected 1, received 2", NewUnit()},
	} {
		r, err := UserValue(newEmptyScope(), test.expressions)
		if test.error {
			assert.EqualError(t, err, test.message)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestUserValueOrErrorMessage(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		message     string
		result      Expr
	}{
		{NewExprs(NewString("123345"), NewString("ALARM!!!")), false, "", NewString("123345")},
		{NewExprs(NewLong(1), NewString("ALARM!!!")), false, "", NewLong(1)},
		{NewExprs(NewUnit(), NewString("ALARM!!!")), true, "ALARM!!!", NewUnit()},
		{NewExprs(), true, "UserValueOrErrorMessage: invalid number of parameters, expected 2, received 0", NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewString("ALARM!!!"), NewLong(1)), true, "UserValueOrErrorMessage: invalid number of parameters, expected 2, received 3", NewUnit()},
	} {
		r, err := UserValueOrErrorMessage(newEmptyScope(), test.expressions)
		if test.error {
			assert.EqualError(t, err, test.message)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}
