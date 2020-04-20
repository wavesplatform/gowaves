package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
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
	addrBody := addr.Body()
	pk2, err := crypto.NewPublicKeyFromBase58(pkStr2)
	assert.NoError(t, err)
	addr2, err := proto.NewAddressFromPublicKey(scheme, pk2)
	assert.NoError(t, err)
	rcp2 := proto.NewRecipientFromAddress(addr2)
	addr2Body := addr2.Body()

	// Test sender only.
	req := &g.TransactionsRequest{Sender: addrBody}
	filter, err := newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.Payment{SenderPK: pk}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.IssueWithSig{Issue: proto.Issue{SenderPK: pk}}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.Genesis{}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk2}}
	assert.Equal(t, false, filter.filter(tx))

	// Test sender and recipient.
	req = &g.TransactionsRequest{
		Sender:    addrBody,
		Recipient: &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: addr2Body}},
	}
	filter, err = newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp2}}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk2, Recipient: rcp2}}
	assert.Equal(t, false, filter.filter(tx))

	// Test sender, recipient and IDs.
	id, err := crypto.NewDigestFromBase58(idStr)
	assert.NoError(t, err)
	id2, err := crypto.NewDigestFromBase58(idStr2)
	assert.NoError(t, err)
	req = &g.TransactionsRequest{
		Sender:         addrBody,
		Recipient:      &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: addrBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	filter, err = newTxFilter(scheme, req)
	assert.NoError(t, err)
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}, ID: &id}
	assert.Equal(t, true, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk2, Recipient: rcp}, ID: &id}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp2}, ID: &id}
	assert.Equal(t, false, filter.filter(tx))
	tx = &proto.TransferWithSig{Transfer: proto.Transfer{SenderPK: pk, Recipient: rcp}, ID: &id2}
	assert.Equal(t, false, filter.filter(tx))
}
