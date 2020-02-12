package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	idStr  = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
	idStr2 = "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
	pkStr  = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	pkStr2 = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7"
	scheme = byte('W')
)

func TestTxFilter(t *testing.T) {
	var tx proto.Transaction
	pk, err := crypto.NewPublicKeyFromBase58(pkStr)
	assert.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	assert.NoError(t, err)
	rcp := proto.NewRecipientFromAddress(addr)
	pk2, err := crypto.NewPublicKeyFromBase58(pkStr2)
	assert.NoError(t, err)
	addr2, err := proto.NewAddressFromPublicKey(scheme, pk2)
	assert.NoError(t, err)
	rcp2 := proto.NewRecipientFromAddress(addr2)

	// Test sender only.
	req := &g.TransactionsRequest{Sender: addr.Body()}
	filter, err := newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.Payment{SenderPK: pk}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.IssueV1{Issue: proto.Issue{SenderPK: pk}}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.Genesis{}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk2}}
	assert.Equal(t, false, filter.filter(tx))

	// Test sender and recipient.
	req = &g.TransactionsRequest{
		Sender:    addr.Body(),
		Recipient: &g.Recipient{Recipient: &g.Recipient_Address{Address: addr2.Body()}},
	}
	filter, err = newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp2}}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk2, Recipient: rcp2}}
	assert.Equal(t, false, filter.filter(tx))

	// Test sender, recipient and IDs.
	id, err := crypto.NewDigestFromBase58(idStr)
	assert.NoError(t, err)
	id2, err := crypto.NewDigestFromBase58(idStr2)
	assert.NoError(t, err)
	req = &g.TransactionsRequest{
		Sender:         addr.Body(),
		Recipient:      &g.Recipient{Recipient: &g.Recipient_Address{Address: addr.Body()}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	filter, err = newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}, ID: &id}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk2, Recipient: rcp}, ID: &id}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp2}, ID: &id}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferV1{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}, ID: &id2}
	assert.Equal(t, false, filter.filter(tx))
}
