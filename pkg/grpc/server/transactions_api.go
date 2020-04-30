package server

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type getTransactionsHandler struct {
	srv g.TransactionsApi_GetTransactionsServer
	s   *Server
}

func (h *getTransactionsHandler) handle(tx proto.Transaction, failed bool) error {
	res, err := h.s.transactionToTransactionResponse(tx, true, failed)
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

func (h *getStateChangesHandler) handle(tx proto.Transaction, failed bool) error {
	var id crypto.Digest
	switch t := tx.(type) {
	case *proto.InvokeScriptWithProofs:
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
		return errors.Wrap(err, "failed to convert ScriptResultV3 to protobuf")
	}
	txProto, err := tx.ToProtobufSigned(h.s.scheme)
	if err != nil {
		return errors.Wrap(err, "failed to convert InvokeScriptWithProofs to protobuf")
	}
	resp := &g.InvokeScriptResultResponse{
		Transaction: txProto,
		Result:      resProto,
	}
	if err := h.srv.Send(resp); err != nil {
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
	filter := newTxFilterInvoke(ftr)
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
		if _, failed, err := s.state.TransactionByIDWithStatus(id); err == nil {
			// Transaction is in state, it is confirmed.
			height, err := s.state.TransactionHeightByID(id)
			if err != nil {
				return status.Errorf(codes.Internal, err.Error())
			}
			res.Status = g.TransactionStatus_CONFIRMED
			res.Height = int64(height)
			if failed {
				res.ApplicationStatus = g.ApplicationStatus_SCRIPT_EXECUTION_FAILED
			} else {
				res.ApplicationStatus = g.ApplicationStatus_SUCCEEDED
			}
		} else if s.utx.ExistsByID(id) {
			// Transaction is in UTX.
			res.Status = g.TransactionStatus_UNCONFIRMED
			res.ApplicationStatus = g.ApplicationStatus_UNKNOWN
		} else {
			res.Status = g.TransactionStatus_NOT_EXISTS
			res.ApplicationStatus = g.ApplicationStatus_UNKNOWN
		}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

type getUnconfirmedHandler struct {
	srv g.TransactionsApi_GetUnconfirmedServer
	s   *Server
}

func (h *getUnconfirmedHandler) handle(tx proto.Transaction, failed bool) error {
	res, err := h.s.transactionToTransactionResponse(tx, false, failed)
	if err != nil {
		return errors.Wrap(err, "failed to form transaction response")
	}
	if err := h.srv.Send(res); err != nil {
		return errors.Wrap(err, "failed to send")
	}
	return nil
}

func (s *Server) GetUnconfirmed(req *g.TransactionsRequest, srv g.TransactionsApi_GetUnconfirmedServer) error {
	filter, err := newTxFilter(s.scheme, req)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, err.Error())
	}
	handler := &getUnconfirmedHandler{srv, s}
	txs := s.utx.AllTransactions()
	for _, tx := range txs {
		if !filter.filter(tx.T) {
			continue
		}
		if err := handler.handle(tx.T, false); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) Sign(ctx context.Context, req *g.SignRequest) (*pb.SignedTransaction, error) {
	var c proto.ProtobufConverter
	tx, err := c.Transaction(req.Transaction)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	pk, err := crypto.NewPublicKeyFromBytes(req.SignerPublicKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := s.wallet.SignTransactionWith(pk, tx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	txProto, err := tx.ToProtobufSigned(s.scheme)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return txProto, nil
}

func (s *Server) Broadcast(ctx context.Context, tx *pb.SignedTransaction) (*pb.SignedTransaction, error) {
	var c proto.ProtobufConverter
	t, err := c.SignedTransaction(tx)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	tBytes, err := t.MarshalBinary()
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	err = s.utx.AddWithBytes(t, tBytes)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to add transaction to UTX")
	}
	return tx, nil
}
