package server

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetInfo(ctx context.Context, req *g.AssetRequest) (*g.AssetInfoResponse, error) {
	id, err := crypto.NewDigestFromBytes(req.AssetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	ai, err := s.state.FullAssetInfo(id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	res, err := ai.ToProtobuf(s.scheme)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return res, nil
}
