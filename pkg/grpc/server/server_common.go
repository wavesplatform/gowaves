package server

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func (s *Server) transactionToTransactionResponse(
	tx proto.Transaction,
	confirmed bool,
	status proto.TransactionStatus,
) (*g.TransactionResponse, error) {
	id, err := tx.GetID(s.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx ID")
	}
	txProto, err := tx.ToProtobufSigned(s.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert transaction to Protobuf")
	}
	res := &g.TransactionResponse{Id: id, Transaction: txProto}
	res.ApplicationStatus = g.ApplicationStatus_UNKNOWN
	if !confirmed {
		return res, nil
	}
	switch status {
	case proto.TransactionSucceeded:
		res.ApplicationStatus = g.ApplicationStatus_SUCCEEDED
	case proto.TransactionFailed:
		res.ApplicationStatus = g.ApplicationStatus_SCRIPT_EXECUTION_FAILED
	case proto.TransactionElided:
		res.ApplicationStatus = g.ApplicationStatus_ELIDED
	default:
		return nil, errors.Errorf("invalid tx status (%d)", status)
	}
	height, err := s.state.TransactionHeightByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx height by ID")
	}
	res.Height = int64(height)
	return res, nil
}

func (s *Server) newStateIterator(sender, recipient *proto.WavesAddress) (state.TransactionIterator, error) {
	if sender != nil {
		return s.state.NewAddrTransactionsIterator(*sender)
	} else if recipient != nil {
		return s.state.NewAddrTransactionsIterator(*recipient)
	}
	return nil, nil
}

type filterFunc = func(tx proto.Transaction) bool
type handleFunc = func(tx proto.Transaction, status proto.TransactionStatus) error

func (s *Server) iterateAndHandleTransactions(iter state.TransactionIterator, filter filterFunc, handle handleFunc) error {
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()
	for iter.Next() {
		// Get and send transactions one-by-one.
		tx, status, err := iter.Transaction()
		if err != nil {
			return errors.Wrap(err, "iterator.Transaction() failed")
		}
		if !filter(tx) {
			continue
		}
		if hErr := handle(tx, status); hErr != nil {
			return errors.Wrap(hErr, "handle() failed")
		}
	}
	if err := iter.Error(); err != nil {
		return errors.Wrap(err, "iterator error")
	}
	return nil
}
