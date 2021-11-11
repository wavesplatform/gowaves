package proto

import (
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

func AssetIDFromDigest(digest crypto.Digest) AssetID {
	r := AssetID{}
	copy(r[:], digest[:AssetIDSize])
	return r
}

func DigestTail(digest crypto.Digest) [AssetIDTailSize]byte {
	var r [AssetIDTailSize]byte
	copy(r[:], digest[AssetIDSize:])
	return r
}

func ReconstructDigest(id AssetID, tail [AssetIDTailSize]byte) crypto.Digest {
	var r crypto.Digest
	copy(r[:AssetIDSize], id[:])
	copy(r[AssetIDSize:], tail[:])
	return r
}
