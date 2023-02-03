package server

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/iterators"
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
		if len(req.TransactionIds) > 0 {
			handler := &getTransactionsHandler{srv, s}
			for _, bts := range req.TransactionIds {
				tx, failed, err := s.state.TransactionByIDWithStatus(bts)
				if err != nil {
					continue
				}
				err = handler.handle(tx, failed)
				if err != nil {
					continue
				}
			}
		}
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
	var iter state.TransactionIterator
	if len(req.TransactionIds) > 0 {
		iter = iterators.NewTxByIdIterator(s.state, req.TransactionIds)
	} else {
		iter, err = s.newStateIterator(ftr.getSenderRecipient())
	}
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
	return g.UnimplementedTransactionsApiServer{}.Sign(ctx, req)
}

func (s *Server) Broadcast(ctx context.Context, tx *pb.SignedTransaction) (out *pb.SignedTransaction, err error) {
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	t, err := c.SignedTransaction(tx)
	if err != nil {
		return nil, apiError(err)
	}
	t, err = t.Validate(s.scheme)
	if err != nil {
		return nil, apiError(err)
	}
	err = broadcast(ctx, s.services.InternalChannel, t)
	if err != nil {
		return nil, apiError(err)
	}
	return tx, nil
}

func apiError(err error) error {
	err = errors.Cause(err)
	switch e := err.(type) {
	case *errs.NonPositiveAmount:
		return status.Errorf(codes.InvalidArgument, "non-positive amount "+err.Error())
	case *errs.TooBigArray:
		return status.Errorf(codes.InvalidArgument, "Too big sequences requested: "+err.Error())
	case *errs.InvalidName:
		return status.Errorf(codes.InvalidArgument, "invalid name: %q", err)
	case *errs.AccountBalanceError:
		return status.Errorf(codes.InvalidArgument, "Accounts balance errors: %q", err)
	case *errs.ToSelf:
		return status.Errorf(codes.InvalidArgument, "Transaction to yourself: %q", err)
	case *errs.TxValidationError:
		return status.Errorf(codes.InvalidArgument, err.Error())
	case *errs.AssetIsNotReissuable:
		return status.Errorf(codes.InvalidArgument, "Asset is not reissuable: %s", err)
	case *errs.AliasTaken:
		return status.Errorf(codes.InvalidArgument, "Alias already claimed: %s", err)
	case *errs.Mistiming:
		return status.Errorf(codes.InvalidArgument, err.Error())
	case *errs.EmptyDataKey:
		return status.Errorf(codes.InvalidArgument, "Empty key found: %s", err)
	case *errs.DuplicatedDataKeys:
		return status.Errorf(codes.InvalidArgument, "Duplicated keys found: %s", err)
	case *errs.UnknownAsset:
		return status.Errorf(codes.InvalidArgument, "Referenced assetId not found: %s", err)
	case *errs.AssetIssuedByOtherAddress:
		return status.Errorf(codes.InvalidArgument, "Asset was issued by other address: %s", err)
	case *errs.FeeValidation:
		return status.Errorf(codes.InvalidArgument, err.Error())
	case *errs.AssetUpdateInterval:
		return status.Errorf(codes.InvalidArgument, err.Error())
	case *errs.TransactionNotAllowedByScript:
		if e.IsAssetScript() {
			return status.Errorf(codes.InvalidArgument, "Transaction is not allowed by token-script: %s: Transaction is not allowed by script of the asset", err)
		}
		return status.Errorf(codes.InvalidArgument, "Transaction is not allowed by account-script")
	default:
		return status.Errorf(codes.Internal, err.Error())
	}
}

func broadcast(ctx context.Context, ch chan messages.InternalMessage, tx proto.Transaction) error {
	respCh := make(chan error, 1)
	select {
	case ch <- messages.NewBroadcastTransaction(respCh, tx):
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		return errors.New("timeout waiting request to internal")
	}
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "ctx cancelled from client")
	case <-time.After(5 * time.Second):
		return errors.New("timeout waiting response from internal")
	case err := <-respCh:
		return err
	}
}
