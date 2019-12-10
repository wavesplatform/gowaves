package server

import (
	"context"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetTransactions(req *g.TransactionsRequest, srv g.TransactionsApi_GetTransactionsServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetStateChanges(req *g.TransactionsRequest, srv g.TransactionsApi_GetStateChangesServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	return status.Errorf(codes.Unimplemented, "Not implemented")
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
