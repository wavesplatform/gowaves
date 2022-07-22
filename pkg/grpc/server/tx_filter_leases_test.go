package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTxFilterLeases(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	require.NoError(t, err)
	st := stateWithCustomGenesis(t, genesisPath)
	txId, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	require.NoError(t, err)
	txId2, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD7p2eJ93dciuBPJp3Qhx")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromString("3Fv3jiLvLS4c4N1ZvSLac3HBGUzaHDMvjN1")
	require.NoError(t, err)
	addrBody := addr.Body()
	pk, err := crypto.NewPublicKeyFromBase58("7rAoh3kPtsPQCTMVe9Bb39GKNX17bR5G57Ef66uwXfeT")
	require.NoError(t, err)
	pk2, err := crypto.NewPublicKeyFromBase58("7rAoh3kPtsPQCTMVe8Bb39GKNX17bR5G57Ef66uwXfeT")
	require.NoError(t, err)

	var tx proto.Transaction
	req := &g.TransactionsRequest{Sender: addrBody}
	filter, err := newTxFilter(scheme, req)
	require.NoError(t, err)
	filterLeases := newTxFilterLeases(filter, st)
	tx = &proto.LeaseWithSig{Lease: proto.Lease{SenderPK: pk}, ID: &txId}
	assert.Equal(t, true, filterLeases.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk}, ID: &txId}
	assert.Equal(t, false, filterLeases.filter(tx))
	tx = &proto.LeaseWithSig{Lease: proto.Lease{SenderPK: pk2}, ID: &txId}
	assert.Equal(t, false, filterLeases.filter(tx))
	tx = &proto.LeaseWithSig{Lease: proto.Lease{SenderPK: pk}, ID: &txId2}
	assert.Equal(t, false, filterLeases.filter(tx))
}
