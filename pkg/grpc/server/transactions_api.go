package server

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type getTransactionsHandler struct {
	srv g.TransactionsApi_GetTransactionsServer
	s   *Server
}

func (h *getTransactionsHandler) handle(tx proto.Transaction) error {
	res, err := h.s.transactionToTransactionResponse(tx)
	if err != nil {
		return errors.Wrap(err, "failed to form transaction response")
	}
	if err := h.srv.Send(res); err != nil {
		return errors.Wrap(err, "failed to send")
	}
	return nil
}

func (s *Server) GetTransactions(req *g.TransactionsRequest, srv g.TransactionsApi_GetTransactionsServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	filter, err := newTxFilter(s.scheme, req)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, err.Error())
	}
	iter, err := s.newStateIterator(filter.getSenderRecipient())
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if iter == nil {
		// Nothing to iterate.
		return nil
	}
	handler := &getTransactionsHandler{srv, s}
	if err := s.iterateAndHandleTransactions(iter, filter.filter, handler.handle); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

type getStateChangesHandler struct {
	srv g.TransactionsApi_GetStateChangesServer
	s   *Server
}

func (h *getStateChangesHandler) handle(tx proto.Transaction) error {
	var id crypto.Digest
	switch t := tx.(type) {
	case *proto.InvokeScriptV1:
		id = *t.ID
	default:
		return errors.New("bad transaction type")
	}
	res, err := h.s.state.InvokeResultByID(id)
	if err != nil {
		return errors.Wrap(err, "InvokeResultByID() failed")
	}
	resProto, err := res.ToProtobuf()
	if err != nil {
		return errors.Wrap(err, "failed to convert ScriptResult to protobuf")
	}
	if err := h.srv.Send(resProto); err != nil {
		return errors.Wrap(err, "failed to send")
	}
	return nil
}

func (s *Server) GetStateChanges(req *g.TransactionsRequest, srv g.TransactionsApi_GetStateChangesServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	ftr, err := newTxFilter(s.scheme, req)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, err.Error())
	}
	filter := newTxFilterInvoke(ftr, s.state)
	iter, err := s.newStateIterator(ftr.getSenderRecipient())
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if iter == nil {
		// Nothing to iterate.
		return nil
	}
	handler := &getStateChangesHandler{srv, s}
	if err := s.iterateAndHandleTransactions(iter, filter.filter, handler.handle); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

func (s *Server) GetStatuses(req *g.TransactionsByIdRequest, srv g.TransactionsApi_GetStatusesServer) error {
	for _, id := range req.TransactionIds {
		res := &g.TransactionStatus{Id: id}
		if _, err := s.state.TransactionByID(id); err == nil {
			// Transaction is in state, it is confirmed.
			height, err := s.state.TransactionHeightByID(id)
			if err != nil {
				return status.Errorf(codes.Internal, err.Error())
			}
			res.Status = g.TransactionStatus_CONFIRMED
			res.Height = int64(height)
		} else if s.utx.TransactionExists(id) {
			// Transaction is in UTX.
			res.Status = g.TransactionStatus_UNCONFIRMED
		} else {
			res.Status = g.TransactionStatus_NOT_EXISTS
		}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) GetUnconfirmed(req *g.TransactionsRequest, srv g.TransactionsApi_GetUnconfirmedServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) Sign(ctx context.Context, req *g.SignRequest) (*g.SignedTransaction, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) Broadcast(ctx context.Context, tx *g.SignedTransaction) (*g.SignedTransaction, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}
