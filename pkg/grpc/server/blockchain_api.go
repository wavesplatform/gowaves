package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	g "github.com/wavesplatform/gowaves/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetActivationStatus(ctx context.Context, req *g.ActivationStatusRequest) (*g.ActivationStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetBaseTarget(ctx context.Context, req *empty.Empty) (*g.BaseTargetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

func (s *Server) GetCumulativeScore(ctx context.Context, req *empty.Empty) (*g.ScoreResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}
