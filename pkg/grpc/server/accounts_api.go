package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/wrappers"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	addr, err := proto.NewAddressFromBytes(req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	if req.Key != "" {
		entry, err := s.state.RetrieveEntry(rcp, req.Key)
		if err != nil {
			return status.Errorf(codes.NotFound, err.Error())
		}
		res := &g.DataEntryResponse{Address: req.Address, Entry: entry.ToProtobuf()}
		return srv.Send(res)
	}
	entries, err := s.state.RetrieveEntries(rcp)
	if err != nil {
		return status.Errorf(codes.NotFound, err.Error())
	}
	for _, entry := range entries {
		res := &g.DataEntryResponse{Address: req.Address, Entry: entry.ToProtobuf()}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) ResolveAlias(ctx context.Context, req *wrappers.StringValue) (*wrappers.BytesValue, error) {
	alias, err := proto.NewAliasFromString(req.Value)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	addr, err := s.state.AddrByAlias(*alias)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &wrappers.BytesValue{Value: addr.Bytes()}, nil
}
