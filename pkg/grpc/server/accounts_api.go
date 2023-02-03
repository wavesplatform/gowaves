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
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *Server) GetBalances(req *g.BalancesRequest, srv g.AccountsApi_GetBalancesServer) error {
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	if len(req.Assets) == 0 {
		// TODO(nickeskov): send waves balance AND all assets balances (portfolio)
		//  by the given address according to the scala node implementation
		if err := s.sendWavesBalance(rcp, srv); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	for _, asset := range req.Assets {
		if len(asset) == 0 {
			if err := s.sendWavesBalance(rcp, srv); err != nil {
				return status.Errorf(codes.Internal, err.Error())
			}
		} else {
			// Asset.
			fullAssetID, err := crypto.NewDigestFromBytes(asset)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, err.Error())
			}
			balance, err := s.state.AssetBalance(rcp, proto.AssetIDFromDigest(fullAssetID))
			if err != nil {
				return status.Errorf(codes.NotFound, err.Error())
			}
			var res g.BalanceResponse
			res.Balance = &g.BalanceResponse_Asset{
				Asset: &pb.Amount{
					AssetId: fullAssetID.Bytes(),
					Amount:  int64(balance),
				},
			}
			if err := srv.Send(&res); err != nil {
				return status.Errorf(codes.Internal, err.Error())
			}
		}
	}
	return nil
}

func (s *Server) GetScript(_ context.Context, req *g.AccountRequest) (*g.ScriptData, error) {
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	scriptInfo, _ := s.state.ScriptInfoByAccount(rcp)
	return scriptInfo.ToProtobuf(), nil
}

func (s *Server) GetActiveLeases(req *g.AccountRequest, srv g.AccountsApi_GetActiveLeasesServer) error {
	extendedApi, err := s.state.ProvidesExtendedApi()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if !extendedApi {
		return status.Errorf(codes.FailedPrecondition, "Node's state does not have information required for extended API")
	}
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}

	filterFn := func(tx proto.Transaction) bool {
		switch t := tx.(type) {
		case *proto.LeaseWithSig:
			ok, _ := s.state.IsActiveLeasing(*t.ID)
			return ok
		case *proto.LeaseWithProofs:
			ok, _ := s.state.IsActiveLeasing(*t.ID)
			return ok
		default:
			return false
		}
	}
	iter, err := s.state.NewAddrTransactionsIterator(addr)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if iter == nil {
		// Nothing to iterate.
		return nil
	}
	handler := &getActiveLeasesHandler{srv, s}
	if err := s.iterateAndHandleTransactions(iter, filterFn, handler.handle); err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	return nil
}

func (s *Server) GetDataEntries(req *g.DataRequest, srv g.AccountsApi_GetDataEntriesServer) error {
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	rcp := proto.NewRecipientFromAddress(addr)
	if req.Key != "" {
		entry, err := s.state.RetrieveEntry(rcp, req.Key)
		if err != nil {
			if err.Error() == "not found" {
				return nil
			}
			return status.Errorf(codes.NotFound, err.Error())
		}
		if entry.GetValueType() == proto.DataDelete { // Send "Not Found" if entry was removed
			return status.Errorf(codes.NotFound, "entry removed")
		}
		res := &g.DataEntryResponse{Address: req.Address, Entry: entry.ToProtobuf()}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
		return nil
	}
	entries, err := s.state.RetrieveEntries(rcp)
	if err != nil {
		if err.Error() == "not found" {
			return nil
		}
		return status.Errorf(codes.Internal, err.Error())
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

func (s *Server) ResolveAlias(_ context.Context, req *wrapperspb.StringValue) (*wrapperspb.BytesValue, error) {
	alias := proto.NewAlias(s.scheme, req.Value)
	addr, err := s.state.AddrByAlias(*alias)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &wrapperspb.BytesValue{Value: addr.Bytes()}, nil
}

func (s *Server) sendWavesBalance(rcp proto.Recipient, srv g.AccountsApi_GetBalancesServer) error {
	var res g.BalanceResponse
	balanceInfo, err := s.state.FullWavesBalance(rcp)
	if err != nil {
		res.Balance = &g.BalanceResponse_Waves{Waves: &g.BalanceResponse_WavesBalances{}}
	} else {
		res.Balance = &g.BalanceResponse_Waves{Waves: balanceInfo.ToProtobuf()}
	}
	return srv.Send(&res)
}

type getActiveLeasesHandler struct {
	srv g.AccountsApi_GetActiveLeasesServer
	s   *Server
}

func (h *getActiveLeasesHandler) handle(tx proto.Transaction, _ bool) error {
	var id []byte
	var sender proto.WavesAddress
	var recipient proto.Recipient
	var amount int64
	var err error
	switch ltx := tx.(type) {
	case *proto.LeaseWithSig:
		id = ltx.ID.Bytes()
		sender, err = proto.NewAddressFromPublicKey(h.s.scheme, ltx.SenderPK)
		if err != nil {
			return err
		}
		recipient = ltx.Recipient
		amount = int64(ltx.Amount)
	case *proto.LeaseWithProofs:
		id = ltx.ID.Bytes()
		sender, err = proto.NewAddressFromPublicKey(h.s.scheme, ltx.SenderPK)
		if err != nil {
			return err
		}
		recipient = ltx.Recipient
		amount = int64(ltx.Amount)
	default:
		return nil
	}

	height, err := h.s.state.TransactionHeightByID(id)
	if err != nil {
		return errors.Wrap(err, "failed to get tx height by ID")
	}
	rcp, err := recipient.ToProtobuf()
	if err != nil {
		return err
	}
	res := &g.LeaseResponse{
		LeaseId:             id,
		OriginTransactionId: id,
		Sender:              sender.Bytes(),
		Recipient:           rcp,
		Amount:              amount,
		Height:              int64(height),
	}

	err = h.srv.Send(res)
	if err != nil {
		return errors.Wrap(err, "failed to send")
	}
	return nil
}
