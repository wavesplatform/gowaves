package server

import (
	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func (s *Server) transactionToTransactionResponse(tx proto.Transaction, setHeight bool) (*g.TransactionResponse, error) {
	id, err := tx.GetID(s.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx ID")
	}
	txProto, err := tx.ToProtobufSigned(s.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert transaction to Protobuf")
	}
	res := &g.TransactionResponse{Id: id, Transaction: txProto}
	if !setHeight {
		return res, nil
	}
	height, err := s.state.TransactionHeightByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx height by ID")
	}
	res.Height = int64(height)
	return res, nil
}

func (s *Server) newStateIterator(sender, recipient *proto.Address) (state.TransactionIterator, error) {
	if sender != nil {
		return s.state.NewAddrTransactionsIterator(*sender)
	} else if recipient != nil {
		return s.state.NewAddrTransactionsIterator(*recipient)
	}
	return nil, nil
}

type filterFunc = func(tx proto.Transaction) bool
type handleFunc = func(tx proto.Transaction) error

func (s *Server) iterateAndHandleTransactions(iter state.TransactionIterator, filter filterFunc, handle handleFunc) error {
	for iter.Next() {
		// Get and send transactions one-by-one.
		tx, err := iter.Transaction()
		if err != nil {
			return errors.Wrap(err, "iterator.Transaction() failed")
		}
		if !filter(tx) {
			continue
		}
		if err := handle(tx); err != nil {
			return errors.Wrap(err, "handle() failed")
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return errors.Wrap(err, "iterator error")
	}
	return nil
}
