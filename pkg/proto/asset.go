package proto

import (
	"github.com/mr-tron/base58"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	AssetIDSize     = 20
	AssetIDTailSize = crypto.DigestSize - AssetIDSize
)

type AssetID [AssetIDSize]byte

func (a AssetID) Bytes() []byte {
	return a[:]
}

func (a AssetID) String() string {
	return base58.Encode(a[:])
}

func (a AssetID) Digest(tail [AssetIDTailSize]byte) crypto.Digest {
	var fullAssetID crypto.Digest
	copy(fullAssetID[:AssetIDSize], a[:])
	copy(fullAssetID[AssetIDSize:], tail[:])
	return fullAssetID
}

func AssetIDFromDigest(digest crypto.Digest) AssetID {
	var id AssetID
	copy(id[:], digest[:AssetIDSize])
	return id
}

func DigestTail(digest crypto.Digest) [AssetIDTailSize]byte {
	var tail [AssetIDTailSize]byte
	copy(tail[:], digest[AssetIDSize:])
	return tail
}

func ReconstructDigest(id AssetID, tail [AssetIDTailSize]byte) crypto.Digest {
	return id.Digest(tail)
}
