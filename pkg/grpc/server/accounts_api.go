package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetBalances(req *g.BalancesRequest, srv g.AccountsApi_GetBalancesServer) error {
	var c proto.ProtobufConverter
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	for _, asset := range req.Assets {
		var res g.BalanceResponse
		if len(asset) == 0 {
			// Waves.
			balanceInfo, err := s.state.FullWavesBalance(rcp)
			if err != nil {
				return status.Errorf(codes.NotFound, err.Error())
			}
			res.Balance = &g.BalanceResponse_Waves{Waves: balanceInfo.ToProtobuf()}
		} else {
			// Asset.
			balance, err := s.state.AccountBalance(rcp, asset)
			if err != nil {
				return status.Errorf(codes.NotFound, err.Error())
			}
			res.Balance = &g.BalanceResponse_Asset{Asset: &pb.Amount{AssetId: asset, Amount: int64(balance)}}
		}
		if err := srv.Send(&res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) GetScript(ctx context.Context, req *g.AccountRequest) (*g.ScriptData, error) {
	var c proto.ProtobufConverter
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	scriptInfo, err := s.state.ScriptInfoByAccount(rcp)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return scriptInfo.ToProtobuf(), nil
}

type getActiveLeasesHandler struct {
	srv g.AccountsApi_GetActiveLeasesServer
	s   *Server
}

func (h *getActiveLeasesHandler) handle(tx proto.Transaction) error {
	res, err := h.s.transactionToTransactionResponse(tx, true)
	if err != nil {
		return errors.Wrap(err, "failed to form transaction response")
	}
	if err := h.srv.Send(res); err != nil {
		return errors.Wrap(err, "failed to send")
	}
	return nil
}

func (s *Server) GetActiveLeases(req *g.AccountRequest, srv g.AccountsApi_GetActiveLeasesServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	reqTr := &g.TransactionsRequest{Sender: req.Address}
	ftr, err := newTxFilter(s.scheme, reqTr)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, err.Error())
	}
	filter := newTxFilterLeases(ftr, s.state)
	iter, err := s.newStateIterator(ftr.getSenderRecipient())
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if iter == nil {
		// Nothing to iterate.
		return nil
	}
	handler := &getActiveLeasesHandler{srv, s}
	if err := s.iterateAndHandleTransactions(iter, filter.filter, handler.handle); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

func (s *Server) GetDataEntries(req *g.DataRequest, srv g.AccountsApi_GetDataEntriesServer) error {
	var c proto.ProtobufConverter
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	if req.Key != "" {
		entry, err := s.state.RetrieveEntry(rcp, req.Key)
		if err != nil {
			return status.Errorf(codes.NotFound, err.Error())
		}
		if entry.GetValueType() == proto.DataDelete { // Send "Not Found" if entry was removed
			return status.Errorf(codes.NotFound, "entry removed")
		}
		res := &g.DataEntryResponse{Address: req.Address, Entry: entry.ToProtobuf()}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	entries, err := s.state.RetrieveEntries(rcp)
	if err != nil {
		return status.Errorf(codes.NotFound, err.Error())
	}
	for _, entry := range entries {
		if entry.GetValueType() == proto.DataDelete {
			continue // Do not send removed entries
		}
		res := &g.DataEntryResponse{Address: req.Address, Entry: entry.ToProtobuf()}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) ResolveAlias(ctx context.Context, req *wrappers.StringValue) (*wrappers.BytesValue, error) {
	alias := proto.NewAlias(s.scheme, req.Value)
	addr, err := s.state.AddrByAlias(*alias)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	addrBody := addr.Body()
	return &wrappers.BytesValue{Value: addrBody}, nil
}
