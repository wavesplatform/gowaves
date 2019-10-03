package proto

import "github.com/wavesplatform/gowaves/pkg/crypto"

type BlockInfo struct {
	Timestamp           uint64
	Height              uint64
	BaseTarget          uint64
	GenerationSignature crypto.Signature
	Generator           Address
	GeneratorPublicKey  crypto.PublicKey
}

type AssetInfo struct {
	ID              crypto.Digest
	Quantity        uint64
	Decimals        byte
	Issuer          Recipient
	IssuerPublicKey crypto.PublicKey
	Reissuable      bool
	Scripted        bool
	Sponsored       bool
}
