package server

import (
	"context"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// allFeatures combines blockchain features from state with features
// which are defined in `settings`.
func (s *Server) allFeatures() ([]int16, error) {
	total := make(map[int16]bool)
	blockchainFeatures, err := s.state.AllFeatures()
	if err != nil {
		return nil, err
	}
	for _, id := range blockchainFeatures {
		total[id] = true
	}
	for id := range settings.FeaturesInfo {
		total[int16(id)] = true
	}
	res := make([]int16, len(total))
	i := 0
	for id := range total {
		res[i] = id
		i++
	}
	return res, nil
}

func (s *Server) nodeStatusFromBool(implemented bool) g.FeatureActivationStatus_NodeFeatureStatus {
	if implemented {
		return g.FeatureActivationStatus_IMPLEMENTED
	}
	return g.FeatureActivationStatus_NOT_IMPLEMENTED
}

// featureActivationStatus retrieves all the info for given feature ID.
func (s *Server) featureActivationStatus(id int16, height uint64) (*g.FeatureActivationStatus, error) {
	res := &g.FeatureActivationStatus{Id: int32(id)}
	res.NodeStatus = g.FeatureActivationStatus_NOT_IMPLEMENTED
	info, ok := settings.FeaturesInfo[settings.Feature(id)]
	if ok {
		res.NodeStatus = s.nodeStatusFromBool(info.Implemented)
		res.Description = info.Description
	}
	activated, err := s.state.IsActiveAtHeight(id, height)
	if err != nil {
		return nil, err
	}
	approved, err := s.state.IsApprovedAtHeight(id, height)
	if err != nil {
		return nil, err
	}
	if activated {
		height, err := s.state.ActivationHeight(id)
		if err != nil {
			return nil, err
		}
		res.ActivationHeight = int32(height)
		res.BlockchainStatus = g.FeatureActivationStatus_ACTIVATED
	} else if approved {
		res.BlockchainStatus = g.FeatureActivationStatus_APPROVED
	} else {
		res.BlockchainStatus = g.FeatureActivationStatus_UNDEFINED
	}
	supportingBlocks, err := s.state.VotesNumAtHeight(id, height)
	if err != nil {
		return nil, err
	}
	res.SupportingBlocks = int32(supportingBlocks)
	return res, nil
}

func (s *Server) GetActivationStatus(ctx context.Context, req *g.ActivationStatusRequest) (*g.ActivationStatusResponse, error) {
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if req.Height > int32(height) {
		return nil, status.Errorf(codes.FailedPrecondition, "requested height exceeds current height")
	}
	reqHeight := uint64(req.Height)
	res := &g.ActivationStatusResponse{Height: req.Height}
	sets, err := s.state.BlockchainSettings()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	res.VotingInterval = int32(sets.ActivationWindowSize(reqHeight))
	res.VotingThreshold = int32(sets.VotesForFeatureElection(reqHeight))
	prevCheck := reqHeight - (reqHeight % uint64(res.VotingInterval))
	res.NextCheck = int32(prevCheck) + res.VotingInterval
	features, err := s.allFeatures()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	res.Features = make([]*g.FeatureActivationStatus, len(features))
	for i, id := range features {
		res.Features[i], err = s.featureActivationStatus(id, reqHeight)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}
	return res, nil
}

func (s *Server) GetBaseTarget(ctx context.Context, req *emptypb.Empty) (*g.BaseTargetResponse, error) {
	height, err := s.state.Height()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	block, err := s.state.BlockByHeight(height)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &g.BaseTargetResponse{BaseTarget: int64(block.BaseTarget)}, nil
}

func (s *Server) GetCumulativeScore(ctx context.Context, req *emptypb.Empty) (*g.ScoreResponse, error) {
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
