package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) getBlockByHeight(height proto.Height, includeTransactions bool) (*g.BlockWithHeight, error) {
	block, err := s.state.BlockByHeight(height)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	res, err := block.ToProtobuf(s.scheme, height)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if !includeTransactions {
		res.Block.Transactions = nil
	}
	return res, nil
}

func (s *Server) GetBlock(ctx context.Context, req *g.BlockRequest) (*g.BlockWithHeight, error) {
	switch r := req.Request.(type) {
	case *g.BlockRequest_BlockId:
		id, err := crypto.NewSignatureFromBytes(r.BlockId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		blockHeight, err := s.state.BlockIDToHeight(id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return s.getBlockByHeight(blockHeight, req.IncludeTransactions)
	case *g.BlockRequest_Height:
		return s.getBlockByHeight(proto.Height(r.Height), req.IncludeTransactions)
	case *g.BlockRequest_Reference:
		id, err := crypto.NewSignatureFromBytes(r.Reference)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		parentHeight, err := s.state.BlockIDToHeight(id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		blockHeight := parentHeight + 1
		return s.getBlockByHeight(blockHeight, req.IncludeTransactions)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown argument type")
	}
}

func (s *Server) GetBlockRange(req *g.BlockRangeRequest, srv g.BlocksApi_GetBlockRangeServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetCurrentHeight(ctx context.Context, req *empty.Empty) (*wrappers.UInt32Value, error) {
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &wrappers.UInt32Value{Value: uint32(height)}, nil
}
