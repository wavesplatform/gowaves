package server

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetInfo(_ context.Context, req *g.AssetRequest) (*g.AssetInfoResponse, error) {
	// we expect full asset id (crypto.Digest)
	fullAssetID, err := crypto.NewDigestFromBytes(req.AssetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	ai, err := s.state.FullAssetInfo(proto.AssetIDFromDigest(fullAssetID))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	res, err := ai.ToProtobuf(s.scheme)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return res, nil
}

func (s *Server) GetNFTList(req *g.NFTRequest, srv g.AssetsApi_GetNFTListServer) error {
	c := proto.ProtobufConverter{FallbackChainID: s.scheme}
	addr, err := c.Address(s.scheme, req.Address)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	var afterAssetID *proto.AssetID
	if len(req.AfterAssetId) != 0 {
		// we expect full asset id (crypto.Digest)
		fullAssetID, err := crypto.NewDigestFromBytes(req.AfterAssetId)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, err.Error())
		}
		assetID := proto.AssetIDFromDigest(fullAssetID)
		afterAssetID = &assetID
	}
	nfts, err := s.state.NFTList(proto.NewRecipientFromAddress(addr), uint64(req.Limit), afterAssetID)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	for _, nft := range nfts {
		ai, err := nft.ToProtobuf(s.scheme)
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
		res := &g.NFTResponse{AssetId: nft.ID.Bytes(), AssetInfo: ai}
		if err := srv.Send(res); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}
