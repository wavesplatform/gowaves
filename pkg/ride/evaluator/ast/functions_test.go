package ast

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestNativeTransactionHeightByID(t *testing.T) {
	sign, err := crypto.NewSignatureFromBase58("hVTTxvgCuezXDsZgh3rDreHzf4AULe5LB1J7zveRbBD4nz3Bzb9yJ2aXKchD4Ls3y2fvYAxnpHXx54S9ZghRx67")
	require.NoError(t, err)

	scope := newScopeWithState(&mockstate.State{
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

	transferWithSig := proto.NewUnsignedTransferWithSig(
		public,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		uint64(time.Now().Unix()),
		1,
		10000,
		proto.NewRecipientFromAddress(sender),
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, transferWithSig.Sign(proto.MainNetScheme, secret))

	scope := newScopeWithState(&mockstate.State{
		TransactionsByID: map[string]proto.Transaction{sign.String(): transferWithSig},
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

func TestNativeTransferTransactionByID(t *testing.T) {
	t.Run("transfer v1", func(t *testing.T) {
		scope := newScopeWithState(&mockstate.State{
			TransactionsByID: map[string]proto.Transaction{
				byte_helpers.TransferWithSig.Transaction.ID.String(): byte_helpers.TransferWithSig.Transaction.Clone(),
			},
		})

		rs, err := NativeTransferTransactionByID(scope, Params(NewBytes(byte_helpers.TransferWithSig.Transaction.ID.Bytes())))
		require.NoError(t, err)
		require.Equal(t, "TransferTransaction", rs.InstanceOf())
	})
	t.Run("transfer v2", func(t *testing.T) {
		scope := newScopeWithState(&mockstate.State{
			TransactionsByID: map[string]proto.Transaction{
				byte_helpers.TransferWithProofs.Transaction.ID.String(): byte_helpers.TransferWithProofs.Transaction.Clone(),
			},
		})

		rs, err := NativeTransferTransactionByID(scope, Params(NewBytes(byte_helpers.TransferWithProofs.Transaction.ID.Bytes())))
		require.NoError(t, err)
		require.Equal(t, "TransferTransaction", rs.InstanceOf())
	})
	t.Run("not found", func(t *testing.T) {
		scope := newScopeWithState(&mockstate.State{})
		rs, err := NativeTransferTransactionByID(scope, Params(NewBytes(byte_helpers.TransferWithProofs.Transaction.ID.Bytes())))
		require.NoError(t, err)
		require.Equal(t, NewUnit(), rs)
	})
}

func TestNativeSizeList(t *testing.T) {
	rs, err := NativeSizeList(newEmptyScopeV1(), Params(NewExprs(NewLong(1))))
	require.NoError(t, err)
	assert.Equal(t, NewLong(1), rs)
}

func TestNativeThrow(t *testing.T) {
	rs, err := NativeThrow(newEmptyScopeV1(), Params(NewString("mess")))
	require.Nil(t, rs)
	if err != nil {
		assert.Equal(t, "mess", err.Error())
	} else {
		assert.Fail(t, "No error")
	}
}

func TestNativeAssetBalance_FromAddress(t *testing.T) {
	addr, err := proto.NewAddressFromString("3N2YHKSnQTUmka4pocTt71HwSSAiUWBcojK")
	require.NoError(t, err)

	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	s := mockstate.State{
		AssetsBalances: map[crypto.Digest]uint64{d: 5},
	}

	rs, err := NativeAssetBalanceV3(newScopeWithState(s), Params(NewAddressFromProtoAddress(addr), NewBytes(d.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(5), rs)
}

func TestNativeAssetBalance_FromAlias(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	s := mockstate.State{
		AssetsBalances: map[crypto.Digest]uint64{d: 5},
	}
	scope := newScopeWithState(s)

	alias := proto.NewAlias(scope.Scheme(), "test")

	rs, err := NativeAssetBalanceV3(scope, Params(NewAliasFromProtoAlias(*alias), NewBytes(d.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(5), rs)
}

func TestNativeAssetBalanceV4(t *testing.T) {
	d, err := crypto.NewDigestFromBase58("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)

	s := mockstate.State{
		AssetsBalances: map[crypto.Digest]uint64{d: 5},
	}
	scope := NewScope(4, proto.MainNetScheme, s)

	alias := proto.NewAlias(scope.Scheme(), "test")

	rs, err := NativeAssetBalanceV4(scope, Params(NewAliasFromProtoAlias(*alias), NewBytes(d.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewLong(5), rs)

	_, err = NativeAssetBalanceV4(scope, Params(NewAliasFromProtoAlias(*alias), NewUnit()))
	assert.Error(t, err)
}

func TestUserWavesBalance(t *testing.T) {
	addr, err := proto.NewAddressFromString("3N2YHKSnQTUmka4pocTt71HwSSAiUWBcojK")
	require.NoError(t, err)
	s := mockstate.State{
		FullWavesBalance: proto.FullWavesBalance{
			Regular:    1,
			Generating: 2,
			Available:  3,
			Effective:  4,
			LeaseIn:    5,
			LeaseOut:   6,
		},
		WavesBalance: 123456,
	}
	scope3 := NewScope(3, proto.TestNetScheme, s)
	scope4 := NewScope(4, proto.TestNetScheme, s)

	rs, err := UserWavesBalanceV3(scope3, Params(NewAddressFromProtoAddress(addr)))
	assert.NoError(t, err)
	v3, ok := rs.(*LongExpr)
	assert.True(t, ok)
	assert.Equal(t, 123456, int(v3.Value))

	rs, err = UserWavesBalanceV4(scope4, Params(NewAddressFromProtoAddress(addr)))
	assert.NoError(t, err)
	v4, ok := rs.(*BalanceDetailsExpr)
	assert.True(t, ok)
	rv, err := v4.Get("regular")
	assert.NoError(t, err)
	rb, ok := rv.(*LongExpr)
	assert.True(t, ok)
	assert.Equal(t, 1, int(rb.Value))
	gv, err := v4.Get("generating")
	assert.NoError(t, err)
	gb, ok := gv.(*LongExpr)
	assert.True(t, ok)
	assert.Equal(t, 2, int(gb.Value))
	av, err := v4.Get("available")
	assert.NoError(t, err)
	ab, ok := av.(*LongExpr)
	assert.True(t, ok)
	assert.Equal(t, 3, int(ab.Value))
	ev, err := v4.Get("effective")
	assert.NoError(t, err)
	eb, ok := ev.(*LongExpr)
	assert.True(t, ok)
	assert.Equal(t, 4, int(eb.Value))
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

	rs1, err := NativeDataIntegerFromArray(newEmptyScopeV1(), Params(NewDataEntryList(dataEntries), NewString("integer")))
	require.NoError(t, err)
	assert.Equal(t, NewLong(100500), rs1)

	rs2, err := NativeDataBooleanFromArray(newEmptyScopeV1(), Params(NewDataEntryList(dataEntries), NewString("boolean")))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs2)

	rs3, err := NativeDataStringFromArray(newEmptyScopeV1(), Params(NewDataEntryList(dataEntries), NewString("string")))
	require.NoError(t, err)
	assert.Equal(t, NewString("world"), rs3)

	rs4, err := NativeDataBinaryFromArray(newEmptyScopeV1(), Params(NewDataEntryList(dataEntries), NewString("binary")))
	require.NoError(t, err)
	assert.Equal(t, NewBytes([]byte("hello")), rs4)

	// test no value
	rs5, err := NativeDataBinaryFromArray(newEmptyScopeV1(), Params(NewDataEntryList(dataEntries), NewString("unknown")))
	require.NoError(t, err)
	assert.Equal(t, &Unit{}, rs5)
}

func TestNativeDataFromState(t *testing.T) {
	a := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := NewAddressFromString(a)
	require.NoError(t, err)

	t.Run("integer", func(t *testing.T) {
		s := mockstate.State{
			DataEntries: map[string]proto.DataEntry{"integer": &proto.IntegerDataEntry{Key: "integer", Value: 100500}},
		}
		rs1, err := NativeDataIntegerFromState(newScopeWithState(s), Params(addr, NewString("integer")))
		require.NoError(t, err)
		assert.Equal(t, NewLong(100500), rs1)
	})

	t.Run("boolean", func(t *testing.T) {
		s := mockstate.State{
			DataEntries: map[string]proto.DataEntry{"boolean": &proto.BooleanDataEntry{Key: "boolean", Value: true}},
		}
		rs2, err := NativeDataBooleanFromState(newScopeWithState(s), Params(addr, NewString("boolean")))
		require.NoError(t, err)
		assert.Equal(t, NewBoolean(true), rs2)

	})

	t.Run("binary", func(t *testing.T) {
		s := mockstate.State{
			DataEntries: map[string]proto.DataEntry{"binary": &proto.BinaryDataEntry{Key: "binary", Value: []byte("hello")}},
		}
		rs3, err := NativeDataBinaryFromState(newScopeWithState(s), Params(addr, NewString("binary")))
		require.NoError(t, err)
		assert.Equal(t, NewBytes([]byte("hello")), rs3)
	})

	t.Run("string", func(t *testing.T) {
		s := mockstate.State{
			DataEntries: map[string]proto.DataEntry{"string": &proto.StringDataEntry{Key: "string", Value: "world"}},
		}
		rs4, err := NativeDataStringFromState(newScopeWithState(s), Params(addr, NewString("string")))
		require.NoError(t, err)
		assert.Equal(t, NewString("world"), rs4)
	})
}

func TestUserIsDefined(t *testing.T) {
	rs1, err := UserIsDefined(newEmptyScopeV1(), Params(NewString("")))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(true), rs1)

	rs2, err := UserIsDefined(newEmptyScopeV1(), Params(NewUnit()))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(false), rs2)
}

func TestUserExtract(t *testing.T) {
	rs1, err := UserExtract(newEmptyScopeV1(), Params(NewString("")))
	require.NoError(t, err)
	assert.Equal(t, NewString(""), rs1)

	_, err = UserExtract(newEmptyScopeV1(), Params(NewUnit()))
	require.EqualError(t, err, "extract() called on unit value")
}

func TestUserUnaryNot(t *testing.T) {
	rs1, err := UserUnaryNot(newEmptyScopeV1(), Params(NewBoolean(true)))
	require.NoError(t, err)
	assert.Equal(t, NewBoolean(false), rs1)
}

func TestUserAddressFromPublicKey(t *testing.T) {
	s := "14ovLL9a6xbBfftyxGNLKMdbnzGgnaFQjmgUJGdho6nY"
	pub, err := crypto.NewPublicKeyFromBase58(s)
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	require.NoError(t, err)

	rs, err := UserAddressFromPublicKey(newEmptyScopeV1(), Params(NewBytes(pub.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs)
}

func TestNativeAddressFromRecipient(t *testing.T) {
	a := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := proto.NewAddressFromString(a)
	require.NoError(t, err)

	s := mockstate.State{}

	rs, err := NativeAddressFromRecipient(newScopeWithState(s), Params(NewRecipientFromProtoRecipient(proto.NewRecipientFromAddress(addr))))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs)
}

func TestUserAddress(t *testing.T) {
	s := "3N9WtaPoD1tMrDZRG26wA142Byd35tLhnLU"
	addr, err := proto.NewAddressFromString(s)
	require.NoError(t, err)

	rs1, err := UserAddress(newEmptyScopeV1(), Params(NewBytes(addr.Bytes())))
	require.NoError(t, err)
	assert.Equal(t, NewAddressFromProtoAddress(addr), rs1)
}

func TestUserAlias(t *testing.T) {
	s := "alias:W:testme"
	alias, err := proto.NewAliasFromString(s)
	require.NoError(t, err)

	rs1, err := UserAlias(newEmptyScopeV1(), Params(NewString("testme")))
	require.NoError(t, err)
	assert.Equal(t, NewAliasFromProtoAlias(*alias), rs1)
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
		r, err := UserValue(newEmptyScopeV1(), test.expressions)
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
		r, err := UserValueOrErrorMessage(newEmptyScopeV1(), test.expressions)
		if test.error {
			assert.EqualError(t, err, test.message)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func ok(e Expr, err error) Expr {
	if err != nil {
		panic("value not found")
	}
	return e
}

func TestNativeBlockInfoByHeight(t *testing.T) {
	_, publicKey, _ := crypto.GenerateKeyPair([]byte("test"))
	parentSig := crypto.MustSignatureFromBase58("4sukfbjbbkBnFevQrGN7VvpBSwvufsuqvq5fmfiMdp1pBDMF5TanbFejRHhsiUQSWPkvWRdagwWD3oxnX3eEqzvM")
	addr := proto.MustAddressFromPublicKey(proto.MainNetScheme, publicKey)
	signa := crypto.MustSignatureFromBase58("5X76YVeG8T6iTxFmD5WNSaR13hxtsgJPQ2oELeZUsrQfZWSXtnUbq1kRqqMjfBngPvaEKVVV2FSujdTXm3hTW172")
	gensig := crypto.MustBytesFromBase58("6a1hWT8QNGw8wnacXQ8vT2YEFLuxRxVpEuaaSf6AbSvU")
	parent := proto.NewBlockIDFromSignature(parentSig)
	h := proto.BlockHeader{
		Version:       3,
		Timestamp:     1567506205718,
		Parent:        parent,
		FeaturesCount: 2,
		Features:      []int16{7, 99},
		NxtConsensus: proto.NxtConsensus{
			BaseTarget:   1310,
			GenSignature: gensig,
		},
		TransactionCount: 12,
		GenPublicKey:     publicKey,
		BlockSignature:   signa,
		Height:           659687,
	}
	state := mockstate.State{
		BlockHeaderByHeight: &h,
	}
	s := newScopeWithState(state)

	rs, err := NativeBlockInfoByHeight(s, Params(NewLong(10)))
	b := rs.(Getable)
	require.NoError(t, err)
	require.Equal(t, NewLong(1567506205718), ok(b.Get("timestamp")))
	require.Equal(t, NewLong(10), ok(b.Get("height")))
	require.Equal(t, NewLong(1310), ok(b.Get("baseTarget")))
	require.Equal(t, NewBytes(gensig), ok(b.Get("generationSignature")))
	require.Equal(t, NewAddressFromProtoAddress(addr), ok(b.Get("generator")))
	require.Equal(t, NewBytes(publicKey.Bytes()), ok(b.Get("generatorPublicKey")))
}

func TestNativeAssetInfoV3(t *testing.T) {
	info := proto.AssetInfo{
		ID: crypto.MustDigestFromBase58("6a1hWT8QNGw8wnacXQ8vT2YEFLuxRxVpEuaaSf6AbSvU"),
	}
	s := mockstate.State{
		Assets: map[crypto.Digest]proto.AssetInfo{info.ID: info},
	}
	rs, err := NativeAssetInfoV3(newScopeWithState(s), Params(NewBytes(info.ID.Bytes())))
	require.NoError(t, err)
	v := rs.(Getable)
	require.Equal(t, NewBytes(info.ID.Bytes()), ok(v.Get("id")))

	wID, err := base58.Decode("WAVES")
	require.NoError(t, err)
	rs2, err := NativeAssetInfoV3(newScopeWithState(s), Params(NewBytes(wID)))
	require.NoError(t, err)
	assert.Equal(t, NewUnit(), rs2)
}

func TestNativeAssetInfoV4(t *testing.T) {
	info := proto.FullAssetInfo{
		AssetInfo: proto.AssetInfo{ID: crypto.MustDigestFromBase58("6a1hWT8QNGw8wnacXQ8vT2YEFLuxRxVpEuaaSf6AbSvU")},
	}
	s := mockstate.State{
		FullAssets: map[crypto.Digest]proto.FullAssetInfo{info.ID: info},
	}
	rs, err := NativeAssetInfoV4(newScopeWithState(s), Params(NewBytes(info.ID.Bytes())))
	require.NoError(t, err)
	v := rs.(Getable)
	require.Equal(t, NewBytes(info.ID.Bytes()), ok(v.Get("id")))

	wID, err := base58.Decode("WAVES")
	require.NoError(t, err)
	rs2, err := NativeAssetInfoV4(newScopeWithState(s), Params(NewBytes(wID)))
	require.NoError(t, err)
	assert.Equal(t, NewUnit(), rs2)
}

func TestNativeList(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		error       bool
		message     string
		result      Expr
	}{
		{NewExprs(NewExprs(NewLong(1), NewLong(2), NewLong(3)), NewLong(0)), false, "", NewLong(1)},
		{NewExprs(NewExprs(NewString("A"), NewString("B"), NewString("C")), NewLong(2)), false, "", NewString("C")},
		{NewExprs(NewExprs(NewString("A")), NewLong(1)), true, "NativeGetList: invalid index 1, len 1", NewUnit()},
		{NewExprs(NewExprs(), NewLong(0)), true, "NativeGetList: invalid index 0, len 0", NewUnit()},
		{NewExprs(), true, "NativeGetList: invalid params, expected 2, passed 0", NewUnit()},
		{NewExprs(NewString("blah-blah-blah"), NewString("ALARM!!!"), NewLong(1)), true, "NativeGetList: invalid params, expected 2, passed 3", NewUnit()},
	} {
		r, err := NativeGetList(newEmptyScopeV1(), test.expressions)
		if test.error {
			assert.EqualError(t, err, test.message)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestDataTransaction(t *testing.T) {
	addr, err := NewAddressFromString("3NAJMMGLfxUF91apoYJQnwY4RQrf5gSfynu")
	require.NoError(t, err)
	for _, test := range []struct {
		expressions Exprs
		error       bool
		message     string
		result      Expr
	}{
		{NewExprs(NewExprs(NewBytes(nil)), NewBytes(nil), NewLong(0), NewLong(0), NewLong(0), addr, NewBytes(nil), NewBytes(nil), NewExprs(NewBytes(nil))), false, "", NewObject(map[string]Expr{"$instance": NewString("DataTransaction"), "bodyBytes": NewBytes(nil), "data": NewExprs(NewBytes(nil)), "fee": NewLong(0), "id": NewBytes(nil), "proofs": NewExprs(NewBytes(nil)), "sender": addr, "senderPublicKey": NewBytes(nil), "timestamp": NewLong(0), "version": NewLong(0)})},
		{NewExprs(), true, "DataTransaction: invalid params, expected 9, passed 0", NewUnit()},
	} {
		r, err := DataTransaction(newEmptyScopeV1(), test.expressions)
		if test.error {
			assert.EqualError(t, err, test.message)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestValueOrElse(t *testing.T) {
	for _, test := range []struct {
		expressions Exprs
		result      Expr
	}{
		{NewExprs(NewString("ride"), NewString("ide")), NewString("ride")},
		{NewExprs(NewString("string"), NewLong(12345)), NewString("string")},
		{NewExprs(NewBoolean(true), NewString("xxx")), NewBoolean(true)},
		{NewExprs(NewLong(12345), NewBoolean(true)), NewLong(12345)},
		{NewExprs(NewUnit(), NewString("ide")), NewString("ide")},
		{NewExprs(NewUnit(), NewLong(12345)), NewLong(12345)},
		{NewExprs(NewUnit(), NewString("xxx")), NewString("xxx")},
		{NewExprs(NewUnit(), NewBoolean(true)), NewBoolean(true)},
	} {
		r, err := ValueOrElse(newEmptyScopeV4(), test.expressions)
		require.NoError(t, err)
		assert.Equal(t, test.result, r)
	}
}

func TestCalculateAssetID(t *testing.T) {
	for _, test := range []struct {
		txID        string
		name        string
		description string
		decimals    int64
		quantity    int64
		reissuable  bool
		nonce       int64
	}{
		{"2K2XASvPkwdePyWaKDKpKT1X7u2uzu6FJASJ34nuTdEi", "asset", "test asset", 2, 100000, false, 0},
		{"F2fxqoTg3PvEwBshxhwKY9BrbqHvi1RZfyFJ4VmRmokZ", "somerset", "this asset is summer set", 8, 100000000000000, true, 1234567890},
		{"AafWgQtRaLm915tNf1fhFdmRr7g6Y9YxyeaJRYuhioRX", "some", "this asset is awesome", 0, 1000000000, true, 987654321},
	} {
		txID, err := crypto.NewDigestFromBase58(test.txID)
		require.NoError(t, err)
		s := newEmptyScopeV4()
		s.AddValue("txId", NewBytes(txID.Bytes()))
		r, err := CalculateAssetID(s, NewExprs(NewIssueExpr(test.name, test.description, test.quantity, test.decimals, test.reissuable, test.nonce)))
		require.NoError(t, err)
		id := proto.GenerateIssueScriptActionID(test.name, test.description, test.decimals, test.quantity, test.reissuable, test.nonce, txID)
		assert.Equal(t, NewBytes(id.Bytes()), r)
	}
}

func TestLimitedCreateList(t *testing.T) {
	for _, test := range []struct {
		expression  Expr
		repetitions int
		error       bool
	}{
		{NewString("ride"), 100, false},
		{NewString("ride"), 1001, true},
		{NewBoolean(true), 100, false},
		{NewBoolean(true), 1001, true},
		{NewLong(12345), 100, false},
		{NewLong(12345), 1001, true},
	} {
		r := NewExprs()
		var ok bool
		s := newEmptyScopeV4()
		for i := 0; i < test.repetitions-1; i++ {
			res, err := LimitedCreateList(s, NewExprs(test.expression, r))
			require.NoError(t, err)
			r, ok = res.(Exprs)
			require.True(t, ok)
		}
		res, err := LimitedCreateList(s, NewExprs(test.expression, r))
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			l, ok := res.(Exprs)
			require.True(t, ok)
			assert.Equal(t, test.repetitions, len(l))
		}
	}
}

func TestAppendToList(t *testing.T) {
	for _, test := range []struct {
		expression  Expr
		repetitions int
		error       bool
	}{
		{NewString("ride"), 100, false},
		{NewString("ride"), 1001, true},
		{NewBoolean(true), 100, false},
		{NewBoolean(true), 1001, true},
		{NewLong(12345), 100, false},
		{NewLong(12345), 1001, true},
	} {
		r := NewExprs()
		var ok bool
		s := newEmptyScopeV4()
		for i := 0; i < test.repetitions-1; i++ {
			res, err := AppendToList(s, NewExprs(r, test.expression))
			require.NoError(t, err)
			r, ok = res.(Exprs)
			require.True(t, ok)
		}
		res, err := AppendToList(s, NewExprs(r, test.expression))
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			l, ok := res.(Exprs)
			require.True(t, ok)
			assert.Equal(t, test.repetitions, len(l))
		}
	}
}

func TestConcat(t *testing.T) {
	list500 := NewExprs()
	for i := 0; i < 500; i++ {
		list500 = append(list500, NewBoolean(true))
	}
	list600 := NewExprs()
	for i := 0; i < 600; i++ {
		list600 = append(list600, NewBoolean(true))
	}
	for _, test := range []struct {
		expressions Exprs
		error       bool
		size        int
	}{
		{NewExprs(NewExprs(NewString("RIDE"), NewString("RIDE")), NewExprs(NewString("RIDE"), NewString("RIDE"))), false, 4},
		{NewExprs(NewExprs(NewString("RIDE"), NewLong(12345)), NewExprs(NewBoolean(true))), false, 3},
		{NewExprs(NewExprs(), NewExprs(NewString("RIDE"), NewString("RIDE"))), false, 2},
		{NewExprs(NewExprs(NewString("RIDE"), NewString("RIDE")), NewExprs()), false, 2},
		{NewExprs(list500, list500), false, 1000},
		{NewExprs(list600, list500), true, 0},
		{NewExprs(list500, list600), true, 0},
		{NewExprs(list600, list600), true, 0},
	} {
		res, err := Concat(newEmptyScopeV4(), test.expressions)
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			list, ok := res.(Exprs)
			require.True(t, ok)
			assert.Equal(t, test.size, len(list))
		}
	}
}

func TestMedian(t *testing.T) {
	list1000 := make([]int, 1000)
	for i := 0; i < len(list1000); i++ {
		list1000[i] = rand.Int()
	}
	for _, test := range []struct {
		items  []int
		error  bool
		median Expr
	}{
		{[]int{1, 2, 3, 4, 5}, false, NewLong(3)},
		{[]int{1, 2, 3, 4}, false, NewLong(2)},
		{[]int{1, 2}, false, NewLong(1)},
		{[]int{0, 0, 0, 0, 0, 0, 0, 0, 1}, false, NewLong(0)},
		{append(list1000, 1), true, NewUnit()},
		{[]int{1}, true, NewUnit()},
		{[]int{}, true, NewUnit()},
	} {
		e := NewExprs()
		for _, x := range test.items {
			e = append(e, NewLong(int64(x)))
		}
		res, err := Median(newEmptyScopeV4(), NewExprs(e))
		if test.error {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.median, res)
		}
	}
}

func TestIssueConstructors(t *testing.T) {
	for _, test := range []struct {
		txID        string
		name        string
		description string
		decimals    int64
		quantity    int64
		reissuable  bool
	}{
		{"2K2XASvPkwdePyWaKDKpKT1X7u2uzu6FJASJ34nuTdEi", "asset", "test asset", 2, 100000, false},
		{"F2fxqoTg3PvEwBshxhwKY9BrbqHvi1RZfyFJ4VmRmokZ", "somerset", "this asset is summer set", 8, 100000000000000, true},
		{"AafWgQtRaLm915tNf1fhFdmRr7g6Y9YxyeaJRYuhioRX", "some", "this asset is awesome", 0, 1000000000, true},
	} {
		txID, err := crypto.NewDigestFromBase58(test.txID)
		require.NoError(t, err)
		s := newEmptyScopeV4()
		s.AddValue("txId", NewBytes(txID.Bytes()))
		i1, err := Issue(s, NewExprs(NewString(test.name), NewString(test.description), NewLong(test.quantity), NewLong(test.decimals), NewBoolean(test.reissuable), NewUnit(), NewLong(0)))
		require.NoError(t, err)
		r1, err := CalculateAssetID(s, NewExprs(i1))
		require.NoError(t, err)
		id1, ok := r1.(*BytesExpr)
		require.True(t, ok)

		i2, err := SimplifiedIssue(s, NewExprs(NewString(test.name), NewString(test.description), NewLong(test.quantity), NewLong(test.decimals), NewBoolean(test.reissuable)))
		require.NoError(t, err)
		r2, err := CalculateAssetID(s, NewExprs(i2))
		require.NoError(t, err)
		id2, ok := r2.(*BytesExpr)
		require.True(t, ok)

		assert.ElementsMatch(t, id1.Value, id2.Value)
	}
}

func TestECRecover(t *testing.T) {
	scope := newEmptyScopeV4()
	sig, err := hex.DecodeString("848ffb6a07e7ce335a2bfe373f1c17573eac320f658ea8cf07426544f2203e9d52dbba4584b0b6c0ed5333d84074002878082aa938fdf68c43367946b2f615d01b")
	require.NoError(t, err)
	md, err := hex.DecodeString("ee97de2243125c58133531c3d5c6e244eb6165df38694b1724623d69fd323e6b")
	require.NoError(t, err)
	epk, err := hex.DecodeString("f80cb44734ef6eba2cff997ca17d1cfb03a85db1b0aa2101a07184e04a3cde02c0f2ecded2918ccb6b86d568cceed83e9beeb749ff8981a718e495aff30ac446")
	require.NoError(t, err)

	res, err := ECRecover(scope, NewExprs(NewBytes(md), NewBytes(sig)))
	require.NoError(t, err)

	pk, ok := res.(*BytesExpr)
	assert.True(t, ok)
	assert.ElementsMatch(t, epk, pk.Value)
}

func TestECRecoverFailures(t *testing.T) {
	scope := newEmptyScopeV4()
	for _, test := range []struct {
		sig string
		md  string
		err string
	}{
		{
			"848ffb6a07e7",
			"ee97de2243125c58133531c3d5c6e244eb6165df38694b1724623d69fd323e6b",
			"ECRecover: invalid signature size 6, expected 65 bytes",
		},
		{
			"848ffb6a07e7ce335a2bfe373f1c17573eac320f658ea8cf07426544f2203e9d52dbba4584b0b6c0ed5333d84074002878082aa938fdf68c43367946b2f615d01b",
			"ee97de224312",
			"ECRecover: invalid message digest size 6, expected 32 bytes",
		},
		{
			"0000fb6a07e7ce335a2bfe373f1c17573eac320f658ea8cf07426544f2203e9d52dbba4584b0b6c0ed5333d84074002878082aa938fdf68c43367946b2f615d01b",
			"ee97de2243125c58133531c3d5c6e244eb6165df38694b1724623d69fd323e6b",
			"ECRecover: failed to recover public key: invalid square root",
		},
	} {
		sig, err := hex.DecodeString(test.sig)
		require.NoError(t, err)
		md, err := hex.DecodeString(test.md)
		require.NoError(t, err)
		_, err = ECRecover(scope, NewExprs(NewBytes(md), NewBytes(sig)))
		assert.EqualError(t, err, test.err)
	}
}

func TestMin(t *testing.T) {
	list999 := NewExprs()
	for i := 0; i < 999; i++ {
		list999 = append(list999, NewLong(0))
	}
	list1000 := append(list999, NewLong(1))
	list1001 := append(list1000, NewLong(2))
	broken1000 := make(Exprs, 1000)
	copy(broken1000, list1000)
	broken1000[999] = NewString("XXX")
	for n, test := range []struct {
		expressions Exprs
		result      Expr
		error       string
	}{
		{NewExprs(NewLong(1), NewLong(2), NewLong(3)), NewLong(1), ""},
		{NewExprs(NewLong(-1), NewLong(-2), NewLong(-3)), NewLong(-3), ""},
		{NewExprs(NewLong(0)), NewLong(0), ""},
		{NewExprs(), nil, "Min: invalid list size 0"},
		{list1000, NewLong(0), ""},
		{broken1000, nil, "Min: list must contain only LongExpr elements"},
		{list1001, nil, "Min: invalid list size 1001"},
	} {
		r, err := Min(newEmptyScopeV4(), NewExprs(test.expressions))
		if test.result != nil {
			require.NoError(t, err, fmt.Sprintf("#%d", n))
			assert.Equal(t, test.result, r, fmt.Sprintf("#%d", n))
		} else {
			assert.EqualError(t, err, test.error, fmt.Sprintf("#%d", n))
		}
	}
}

func TestMax(t *testing.T) {
	list999 := NewExprs()
	for i := 0; i < 999; i++ {
		list999 = append(list999, NewLong(0))
	}
	list1000 := append(list999, NewLong(1))
	list1001 := append(list1000, NewLong(2))
	broken1000 := make(Exprs, 1000)
	copy(broken1000, list1000)
	broken1000[999] = NewString("XXX")
	for _, test := range []struct {
		expressions Exprs
		result      Expr
		error       string
	}{
		{NewExprs(NewLong(1), NewLong(2), NewLong(3)), NewLong(3), ""},
		{NewExprs(NewLong(-1), NewLong(-2), NewLong(-3)), NewLong(-1), ""},
		{NewExprs(NewLong(0)), NewLong(0), ""},
		{NewExprs(), nil, "Max: invalid list size 0"},
		{list1000, NewLong(1), ""},
		{broken1000, nil, "Max: list must contain only LongExpr elements"},
		{list1001, nil, "Max: invalid list size 1001"},
	} {
		r, err := Max(newEmptyScopeV4(), NewExprs(test.expressions))
		if test.result != nil {
			require.NoError(t, err)
			assert.Equal(t, test.result, r)
		} else {
			assert.EqualError(t, err, test.error)
		}
	}
}

func TestIndexOf(t *testing.T) {
	list1000 := NewExprs()
	for i := 0; i < 1000; i++ {
		list1000 = append(list1000, NewLong(0))
	}
	list1001 := append(list1000, NewLong(2))
	for _, test := range []struct {
		expressions Exprs
		result      Expr
		error       string
	}{
		{NewExprs(NewExprs(NewLong(1), NewLong(2), NewLong(3)), NewLong(3)), NewLong(2), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewLong(3)), nil, "IndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewLong(1)), NewLong(0), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewString("A")), NewLong(1), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewString("B")), nil, "IndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewBoolean(false)), nil, "IndexOf: not found"},
		{NewExprs(NewExprs(), NewBoolean(false)), nil, "IndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewString("A"), NewBoolean(true)), NewString("A")), NewLong(1), ""},
		{NewExprs(list1000, NewLong(0)), NewLong(0), ""},
		{NewExprs(list1001, NewLong(0)), nil, "IndexOf: list size can not exceed 1000 elements"},
	} {
		r, err := IndexOf(newEmptyScopeV4(), test.expressions)
		if test.result != nil {
			require.NoError(t, err)
			assert.Equal(t, test.result, r)
		} else {
			assert.EqualError(t, err, test.error)
		}
	}
}

func TestLastIndexOf(t *testing.T) {
	list1000 := NewExprs()
	for i := 0; i < 1000; i++ {
		list1000 = append(list1000, NewLong(0))
	}
	list1001 := append(list1000, NewLong(2))
	for _, test := range []struct {
		expressions Exprs
		result      Expr
		error       string
	}{
		{NewExprs(NewExprs(NewLong(1), NewLong(2), NewLong(3)), NewLong(3)), NewLong(2), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewLong(3)), nil, "LastIndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewLong(1)), NewLong(0), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewString("A")), NewLong(1), ""},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewString("B")), nil, "LastIndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewBoolean(true)), NewBoolean(false)), nil, "LastIndexOf: not found"},
		{NewExprs(NewExprs(), NewBoolean(false)), nil, "LastIndexOf: not found"},
		{NewExprs(NewExprs(NewLong(1), NewString("A"), NewString("A"), NewBoolean(true)), NewString("A")), NewLong(2), ""},
		{NewExprs(list1000, NewLong(0)), NewLong(999), ""},
		{NewExprs(list1001, NewLong(0)), nil, "LastIndexOf: list size can not exceed 1000 elements"},
	} {
		r, err := LastIndexOf(newEmptyScopeV4(), test.expressions)
		if test.result != nil {
			require.NoError(t, err)
			assert.Equal(t, test.result, r)
		} else {
			assert.EqualError(t, err, test.error)
		}
	}
}
