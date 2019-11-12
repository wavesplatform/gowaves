package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetBlock(ctx context.Context, req *g.BlockRequest) (*g.BlockWithHeight, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetBlockRange(req *g.BlockRangeRequest, srv g.BlocksApi_GetBlockRangeServer) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetCurrentHeight(ctx context.Context, req *empty.Empty) (*wrappers.UInt32Value, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}
