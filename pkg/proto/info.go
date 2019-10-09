package proto

import "github.com/wavesplatform/gowaves/pkg/crypto"

type BlockInfo struct {
	Timestamp           uint64
	Height              uint64
	BaseTarget          uint64
	GenerationSignature crypto.Digest
	Generator           Address
	GeneratorPublicKey  crypto.PublicKey
}

func BlockInfoFromHeader(scheme byte, header *BlockHeader, height uint64) (*BlockInfo, error) {
	generator, err := NewAddressFromPublicKey(scheme, header.GenPublicKey)
	if err != nil {
		return nil, err
	}
	return &BlockInfo{
		Timestamp:           header.Timestamp,
		Height:              height,
		BaseTarget:          header.BaseTarget,
		GenerationSignature: header.GenSignature,
		Generator:           generator,
		GeneratorPublicKey:  header.GenPublicKey,
	}, nil
}

type AssetInfo struct {
	ID              crypto.Digest
	Quantity        uint64
	Decimals        byte
	Issuer          Address
	IssuerPublicKey crypto.PublicKey
	Reissuable      bool
	Scripted        bool
	Sponsored       bool
}
