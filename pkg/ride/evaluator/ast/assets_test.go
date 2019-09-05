package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/mockstate"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

func TestNewMapAssetInfoV1(t *testing.T) {
	state := mockstate.MockStateImpl{
		AssetIsSponsored: true,
	}
	tx := byte_helpers.IssueV1.Transaction.Clone()
	rs, err := newMapAssetInfo(proto.MainNetScheme, state, byte_helpers.IssueV1.Transaction.Clone())
	require.NoError(t, err)
	require.Equal(t, NewBytes(tx.ID.Bytes()), rs["id"])
	require.Equal(t, NewLong(1000), rs["quantity"])
	require.Equal(t, NewLong(4), rs["decimals"])
	require.Equal(t, NewAddressFromProtoAddress(proto.MustAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)), rs["issuer"])
	require.Equal(t, NewBytes(tx.SenderPK.Bytes()), rs["issuerPublicKey"])
	require.Equal(t, NewBoolean(false), rs["reissuable"])
	require.Equal(t, NewBoolean(false), rs["scripted"])
	require.Equal(t, NewBoolean(true), rs["sponsored"])
}

func TestNewMapAssetInfoV2(t *testing.T) {
	state := mockstate.MockStateImpl{
		AssetIsSponsored: true,
	}
	tx := byte_helpers.IssueV2.Transaction.Clone()
	rs, err := newMapAssetInfo(proto.MainNetScheme, state, tx.Clone())
	require.NoError(t, err)
	require.Equal(t, NewBytes(tx.ID.Bytes()), rs["id"])
	require.Equal(t, NewLong(1000), rs["quantity"])
	require.Equal(t, NewLong(4), rs["decimals"])
	require.Equal(t, NewAddressFromProtoAddress(proto.MustAddressFromPublicKey(proto.MainNetScheme, tx.SenderPK)), rs["issuer"])
	require.Equal(t, NewBytes(tx.SenderPK.Bytes()), rs["issuerPublicKey"])
	require.Equal(t, NewBoolean(false), rs["reissuable"])
	require.Equal(t, NewBoolean(true), rs["scripted"])
	require.Equal(t, NewBoolean(true), rs["sponsored"])
}

func TestAssetInfoExprIsObject(t *testing.T) {
	var a Expr = NewAssetInfo(nil)
	_ = a.(Getable)
}

func TestAssetPairExprIsObject(t *testing.T) {
	var e Expr = NewAssetPair(nil, nil)
	_ = e.(Getable)
}
