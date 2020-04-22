package server

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	case *g.BlockRequest_Reference:
		id, err := proto.NewBlockIDFromBytes(r.Reference)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		parentHeight, err := s.state.BlockIDToHeight(id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		blockHeight := parentHeight + 1
		return s.headerOrBlockByHeight(blockHeight, req.IncludeTransactions)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown argument type")
	}
}

func (s *Server) GetBlockRange(req *g.BlockRangeRequest, srv g.BlocksApi_GetBlockRangeServer) error {
	generator := req.GetGenerator()
	hasFilter := generator != nil
	for height := proto.Height(req.FromHeight); height <= proto.Height(req.ToHeight); height++ {
		block, err := s.headerOrBlockByHeight(height, req.IncludeTransactions)
		if err != nil {
			return status.Errorf(codes.NotFound, err.Error())
		}
		if hasFilter && !bytes.Equal(block.Block.Header.Generator, generator) {
			continue
		}
		if err := srv.Send(block); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *Server) GetCurrentHeight(ctx context.Context, req *empty.Empty) (*wrappers.UInt32Value, error) {
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &wrappers.UInt32Value{Value: uint32(height)}, nil
}
