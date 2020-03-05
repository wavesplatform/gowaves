package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTxFilterLeases(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	assert.NoError(t, err)
	st, stCloser := stateWithCustomGenesis(t, genesisPath)
	txId, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	assert.NoError(t, err)
	txId2, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD7p2eJ93dciuBPJp3Qhx")
	assert.NoError(t, err)
	addr, err := proto.NewAddressFromString("3Fv3jiLvLS4c4N1ZvSLac3HBGUzaHDMvjN1")
	assert.NoError(t, err)
	addrBody, err := addr.Body()
	assert.NoError(t, err)
	pk, err := crypto.NewPublicKeyFromBase58("7rAoh3kPtsPQCTMVe9Bb39GKNX17bR5G57Ef66uwXfeT")
	assert.NoError(t, err)
	pk2, err := crypto.NewPublicKeyFromBase58("7rAoh3kPtsPQCTMVe8Bb39GKNX17bR5G57Ef66uwXfeT")
	assert.NoError(t, err)

	defer func() {
		stCloser()
	}()

	var tx proto.Transaction
	req := &g.TransactionsRequest{Sender: addrBody}
	filter, err := newTxFilter(scheme, req)
	assert.NoError(t, err)
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
