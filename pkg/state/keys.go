package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// Balances (main state).
	balanceKeyPrefix byte = iota

	// Valid block IDs.
	blockIdKeyPrefix

	// For block storage.
	// IDs of blocks and transactions --> offsets in files.
	blockOffsetKeyPrefix
	txOffsetKeyPrefix

	// Minimum height to which rollback is possible.
	rollbackMinHeightKeyPrefix
	// Min height of blockReadWriter's files.
	rwHeightKeyPrefix
	// Height of main db.
	dbHeightKeyPrefix

	// Score at height.
	scoreKeyPrefix
	// Assets.
	assetConstKeyPrefix
	assetHistKeyPrefix

	// Known peers
	knownPeersPrefix
)

type balanceKey struct {
	address proto.Address
	asset   []byte
}

func (k *balanceKey) bytes() []byte {
	if k.asset != nil {
		buf := make([]byte, 1+proto.AddressSize+crypto.DigestSize)
		buf[0] = balanceKeyPrefix
		copy(buf[1:], k.address[:])
		copy(buf[1+proto.AddressSize:], k.asset)
		return buf
	} else {
		buf := make([]byte, 1+proto.AddressSize)
		buf[0] = balanceKeyPrefix
		copy(buf[1:], k.address[:])
		return buf
	}
}

type blockIdKey struct {
	blockID crypto.Signature
}

func (k *blockIdKey) bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = blockIdKeyPrefix
	copy(buf[1:], k.blockID[:])
	return buf
}

type blockOffsetKey struct {
	blockID crypto.Signature
}

func (k *blockOffsetKey) bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = blockOffsetKeyPrefix
	copy(buf[1:], k.blockID[:])
	return buf
}

type txOffsetKey struct {
	txID []byte
}

func (k *txOffsetKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = txOffsetKeyPrefix
	copy(buf[1:], k.txID)
	return buf
}

type scoreKey struct {
	height uint64
}

func (k *scoreKey) bytes() []byte {
	buf := make([]byte, 9)
	buf[0] = scoreKeyPrefix
	binary.LittleEndian.PutUint64(buf[1:], k.height)
	return buf
}

type assetConstKey struct {
	assetID crypto.Digest
}

func (k *assetConstKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = assetConstKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}

type assetHistKey struct {
	assetID crypto.Digest
}

func (k *assetHistKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = assetHistKeyPrefix
	copy(buf[1:], k.assetID[:])
	return buf
}
