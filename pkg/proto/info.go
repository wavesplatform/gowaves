package proto

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type BlockInfo struct {
	Version             BlockVersion
	Timestamp           uint64
	Height              uint64
	BaseTarget          uint64
	Generator           WavesAddress
	GeneratorPublicKey  crypto.PublicKey
	GenerationSignature B58Bytes
	VRF                 B58Bytes
	Rewards             Rewards
}

func NewBlockInfo(version BlockVersion, timestamp, height, baseTarget uint64,
	generatorAddress WavesAddress, generatorPK crypto.PublicKey,
	generationSignature, vrf []byte, rewards Rewards) *BlockInfo {
	return &BlockInfo{
		Version:             version,
		Timestamp:           timestamp,
		Height:              height,
		BaseTarget:          baseTarget,
		Generator:           generatorAddress,
		GeneratorPublicKey:  generatorPK,
		GenerationSignature: generationSignature,
		VRF:                 vrf,
		Rewards:             rewards,
	}
}

func BlockInfoFromHeader(header *BlockHeader, generator WavesAddress, height uint64, vrf []byte, rewards Rewards) (*BlockInfo, error) {
	return &BlockInfo{
		Version:             header.Version,
		Timestamp:           header.Timestamp,
		Height:              height,
		BaseTarget:          header.BaseTarget,
		GenerationSignature: header.GenSignature,
		Generator:           generator,
		GeneratorPublicKey:  header.GeneratorPublicKey,
		VRF:                 vrf,
		Rewards:             rewards.Sorted(),
	}, nil
}

func (bi *BlockInfo) CopyVRF() []byte {
	if bi.Version >= ProtobufBlockVersion {
		return common.Dup(bi.VRF)
	}
	return nil
}

func (bi *BlockInfo) CopyGenerationSignature() []byte {
	return common.Dup(bi.GenerationSignature)
}

func (bi *BlockInfo) CopyGeneratorPublicKey() []byte {
	return common.Dup(bi.GeneratorPublicKey.Bytes())
}

func (bi *BlockInfo) IsEmptyGenerator() bool {
	return bi.GeneratorPublicKey == crypto.PublicKey{}
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

type AssetConstInfo struct {
	ID          crypto.Digest
	IssueHeight Height
	Issuer      WavesAddress
	Decimals    uint8
}

type AssetInfo struct {
	AssetConstInfo
	Quantity        uint64
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

type EnrichedFullAssetInfo struct {
	FullAssetInfo
	SequenceInBlock uint32
}
