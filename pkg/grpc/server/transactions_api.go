package server

import (
	"context"

	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetTransactions(req *g.TransactionsRequest, srv g.TransactionsApi_GetTransactionsServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetStateChanges(req *g.TransactionsRequest, srv g.TransactionsApi_GetStateChangesServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetStatuses(req *g.TransactionsByIdRequest, srv g.TransactionsApi_GetStatusesServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
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
