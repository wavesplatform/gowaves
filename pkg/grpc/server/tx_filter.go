package server

import (
	"bytes"

	"github.com/wavesplatform/gowaves/pkg/grpc/client"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type txFilter struct {
	sender    proto.Address
	recipient proto.Recipient
	ids       map[string]bool
	scheme    byte

	hasSender, hasRecipient, hasIds bool
}

func newTxFilter(scheme byte, req *g.TransactionsRequest) (*txFilter, error) {
	res := &txFilter{}
	res.scheme = scheme
	if req.Sender != nil {
		sender, err := proto.NewAddressFromBytes(req.Sender)
		if err != nil {
			return nil, err
		}
		res.sender = sender
		res.hasSender = true
	}
	if req.Recipient != nil {
		var c client.SafeConverter
		res.recipient = c.Recipient(scheme, req.Recipient)
		res.hasRecipient = true
	}
	if req.TransactionIds != nil {
		ids := make(map[string]bool)
		for _, id := range req.TransactionIds {
			ids[string(id)] = true
		}
		res.ids = ids
		res.hasIds = true
	}
	return res, nil
}

func (f *txFilter) filterSender(tx proto.Transaction) bool {
	if !f.hasSender {
		return true
	}
	senderAddr, err := proto.NewAddressFromPublicKey(f.scheme, tx.GetSenderPK())
	if err != nil {
		return false
	}
	return f.sender == senderAddr
}

func (f *txFilter) filterRecipient(tx proto.Transaction) bool {
	if !f.hasRecipient {
		return true
	}
	switch t := tx.(type) {
	case *proto.TransferV1:
		return t.Recipient == f.recipient
	case *proto.TransferV2:
		return t.Recipient == f.recipient
	case *proto.LeaseV1:
		return t.Recipient == f.recipient
	case *proto.LeaseV2:
		return t.Recipient == f.recipient
	case *proto.MassTransferV1:
		return t.HasRecipient(f.recipient)
	default:
		if f.recipient.Address == nil {
			return false
		}
		senderAddr, err := proto.NewAddressFromPublicKey(f.scheme, tx.GetSenderPK())
		if err != nil {
			return false
		}
		return bytes.Equal(f.recipient.Address[:], senderAddr[:])
	}
}

func (f *txFilter) filterId(tx proto.Transaction) bool {
	if !f.hasIds {
		return true
	}
	id, err := tx.GetID()
	if err != nil {
		return false
	}
	_, containsId := f.ids[string(id)]
	return containsId
}

func (f *txFilter) filter(tx proto.Transaction) bool {
	return f.filterSender(tx) && f.filterRecipient(tx) && f.filterId(tx)
}

func (f *txFilter) getSenderRecipient() (*proto.Address, *proto.Address) {
	var sender, recipient *proto.Address
	if f.hasSender {
		sender = &f.sender
	}
	if f.hasRecipient {
		recipient = f.recipient.Address
	}
	return sender, recipient
}
