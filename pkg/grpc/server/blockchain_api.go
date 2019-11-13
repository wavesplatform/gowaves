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
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	block, err := s.state.BlockByHeight(height)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &g.BaseTargetResponse{BaseTarget: int64(block.BaseTarget)}, nil
}

func (s *Server) GetCumulativeScore(ctx context.Context, req *empty.Empty) (*g.ScoreResponse, error) {
	score, err := s.state.CurrentScore()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	scoreBytes, err := score.GobEncode()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &g.ScoreResponse{Score: scoreBytes}, nil
}
