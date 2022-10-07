package proto

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
)

type BlockInfo struct {
	Timestamp           uint64
	Height              uint64
	BaseTarget          uint64
	GenerationSignature B58Bytes
	Generator           WavesAddress
	GeneratorPublicKey  crypto.PublicKey
	VRF                 B58Bytes
}

func BlockInfoFromHeader(scheme byte, header *BlockHeader, height uint64, vrf []byte) (*BlockInfo, error) {
	generator, err := NewAddressFromPublicKey(scheme, header.GeneratorPublicKey)
	if err != nil {
		return nil, err
	}
	return &BlockInfo{
		Timestamp:           header.Timestamp,
		Height:              height,
		BaseTarget:          header.BaseTarget,
		GenerationSignature: header.GenSignature,
		Generator:           generator,
		GeneratorPublicKey:  header.GeneratorPublicKey,
		VRF:                 vrf,
	}, nil
}

type FullAssetInfo struct {
	AssetInfo
	Name             string
	Description      string
	ScriptInfo       ScriptInfo
	SponsorshipCost  uint64
	IssueTransaction Transaction
	SponsorBalance   uint64
}

func (i *FullAssetInfo) ToProtobuf(scheme Scheme) (*g.AssetInfoResponse, error) {
	res := i.AssetInfo.ToProtobuf()
	res.Name = i.Name
	res.Description = i.Description
	res.Script = i.ScriptInfo.ToProtobuf()
	res.Sponsorship = int64(i.SponsorshipCost)
	// Issue transaction is optional here
	var protoTransaction *waves.SignedTransaction
	if i.IssueTransaction != nil {
		var err error
		protoTransaction, err = i.IssueTransaction.ToProtobufSigned(scheme)
		if err != nil {
			return nil, err
		}
	}
	res.IssueTransaction = protoTransaction
	res.SponsorBalance = int64(i.SponsorBalance)
	return res, nil
}

type AssetInfo struct {
	ID              crypto.Digest
	Quantity        uint64
	Decimals        byte
	Issuer          WavesAddress
	IssuerPublicKey crypto.PublicKey
	Reissuable      bool
	Scripted        bool
	Sponsored       bool
}

func (ai *AssetInfo) ToProtobuf() *g.AssetInfoResponse {
	return &g.AssetInfoResponse{
		Issuer:      ai.IssuerPublicKey.Bytes(),
		Decimals:    int32(ai.Decimals),
		Reissuable:  ai.Reissuable,
		TotalVolume: int64(ai.Quantity),
	}
}
