package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/wrappers"
	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetBalances(req *g.BalancesRequest, srv g.AccountsApi_GetBalancesServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetScript(ctx context.Context, req *g.AccountRequest) (*g.ScriptData, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetActiveLeases(req *g.AccountRequest, srv g.AccountsApi_GetActiveLeasesServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetDataEntries(req *g.DataRequest, srv g.AccountsApi_GetDataEntriesServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) ResolveAlias(ctx context.Context, req *wrappers.StringValue) (*wrappers.BytesValue, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}
