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

func DigestTail(digest crypto.Digest) [12]byte {
	var r [12]byte
	copy(r[:], digest[AssetIDSize:])
	return r
}

func ReconstructDigest(id AssetID, tail [12]byte) crypto.Digest {
	var r crypto.Digest
	copy(r[:AssetIDSize], id[:])
	copy(r[AssetIDSize:], tail[:])
	return r
}
