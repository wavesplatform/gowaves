package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTxFilterInvoke(t *testing.T) {
	var tx proto.Transaction
	pk, err := crypto.NewPublicKeyFromBase58(pkStr)
	assert.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	assert.NoError(t, err)
	addrBody, err := addr.Body()
	assert.NoError(t, err)
	pk2, err := crypto.NewPublicKeyFromBase58(pkStr2)
	assert.NoError(t, err)
	id, err := crypto.NewDigestFromBase58(idStr)
	assert.NoError(t, err)
	id2, err := crypto.NewDigestFromBase58(idStr2)
	assert.NoError(t, err)

	req := &g.TransactionsRequest{
		Sender:         addrBody,
		TransactionIds: [][]byte{id.Bytes()},
	}
	filter, err := newTxFilter(scheme, req)
	assert.NoError(t, err)
	filterInvoke := newTxFilterInvoke(filter)
	tx = &proto.InvokeScriptWithProofs{SenderPK: pk, ID: &id}
	assert.Equal(t, true, filterInvoke.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk}, ID: &id}
	assert.Equal(t, false, filterInvoke.filter(tx))
	tx = &proto.InvokeScriptWithProofs{SenderPK: pk2, ID: &id}
	assert.Equal(t, false, filterInvoke.filter(tx))
	tx = &proto.InvokeScriptWithProofs{SenderPK: pk, ID: &id2}
	assert.Equal(t, false, filterInvoke.filter(tx))
}
