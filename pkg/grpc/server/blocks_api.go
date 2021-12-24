package server

import (
	"bytes"
	"context"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *Server) headerByHeight(height proto.Height) (*g.BlockWithHeight, error) {
	header, err := s.state.HeaderByHeight(height)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	res, err := header.HeaderToProtobufWithHeight(s.scheme, height)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return res, nil
}

func (s *Server) blockByHeight(height proto.Height) (*g.BlockWithHeight, error) {
	block, err := s.state.BlockByHeight(height)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	res, err := block.ToProtobufWithHeight(s.scheme, height)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return res, nil
}

func (s *Server) headerOrBlockByHeight(height proto.Height, includeTransactions bool) (*g.BlockWithHeight, error) {
	if includeTransactions {
		return s.blockByHeight(height)
	}
	return s.headerByHeight(height)
}

func (s *Server) GetBlock(ctx context.Context, req *g.BlockRequest) (*g.BlockWithHeight, error) {
	switch r := req.Request.(type) {
	case *g.BlockRequest_BlockId:
		id, err := proto.NewBlockIDFromBytes(r.BlockId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		blockHeight, err := s.state.BlockIDToHeight(id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return s.headerOrBlockByHeight(blockHeight, req.IncludeTransactions)
	case *g.BlockRequest_Height:
		return s.headerOrBlockByHeight(proto.Height(r.Height), req.IncludeTransactions)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown argument type")
	}
}

func (s *Server) GetBlockRange(req *g.BlockRangeRequest, srv g.BlocksApi_GetBlockRangeServer) error {
	var filter func(b *g.BlockWithHeight) bool
	switch t := req.Filter.(type) {
	case *g.BlockRangeRequest_GeneratorPublicKey:
		filter = func(b *g.BlockWithHeight) bool {
			return bytes.Equal(t.GeneratorPublicKey, b.Block.Header.Generator)
		}
	case *g.BlockRangeRequest_GeneratorAddress:
		addr, err := proto.RebuildAddress(s.scheme, t.GeneratorAddress)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid address: %s", err.Error())
		}
		filter = func(b *g.BlockWithHeight) bool {
			genaddr, _ := proto.RebuildAddress(s.scheme, t.GeneratorAddress)
			return addr == genaddr
		}
	default:
		filter = func(b *g.BlockWithHeight) bool {
			return true
		}
	}
	stateHeight, err := s.state.Height()
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	if req.ToHeight > uint32(stateHeight) {
		req.ToHeight = uint32(stateHeight)
	}
	for height := proto.Height(req.FromHeight); height <= proto.Height(req.ToHeight); height++ {
		block, err := s.headerOrBlockByHeight(height, req.IncludeTransactions)
		if err != nil {
			return status.Errorf(codes.NotFound, err.Error())
		}
		if !filter(block) {
			continue
		}
		if err := srv.Send(block); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) GetCurrentHeight(ctx context.Context, req *emptypb.Empty) (*wrapperspb.UInt32Value, error) {
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &wrapperspb.UInt32Value{Value: uint32(height)}, nil
}
