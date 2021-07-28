package proto

import "github.com/wavesplatform/gowaves/pkg/crypto"

const AssetIDSize = 20

type AssetID [AssetIDSize]byte

func (a AssetID) Bytes() []byte {
	return a[:]
}

func AssetIDFromDigest(digest crypto.Digest) AssetID {
	r := AssetID{}
	copy(r[:], digest[:AssetIDSize])
	return r
}
