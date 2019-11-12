package server

import (
	"context"

	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetInfo(ctx context.Context, req *g.AssetRequest) (*g.AssetInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}
