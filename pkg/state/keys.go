package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	// Balances (main state).
	BalanceKeyPrefix byte = iota

	// Valid block IDs.
	BlockIdKeyPrefix

	// For block storage.
	// IDs of blocks and transactions --> offsets in files.
	BlockOffsetKeyPrefix
	TxOffsetKeyPrefix

	// Min height of BlockReadWriter's files.
	RwHeightKeyPrefix
	// Height of main db.
	DbHeightKeyPrefix
)

type BalanceKey struct {
	Address proto.Address
	Asset   []byte
}

func (k *BalanceKey) Bytes() []byte {
	if k.Asset != nil {
		buf := make([]byte, 1+proto.AddressSize+crypto.DigestSize)
		buf[0] = BalanceKeyPrefix
		copy(buf[1:], k.Address[:])
		copy(buf[1+proto.AddressSize:], k.Asset)
		return buf
	} else {
		buf := make([]byte, 1+proto.AddressSize)
		buf[0] = BalanceKeyPrefix
		copy(buf[1:], k.Address[:])
		return buf
	}
}

type BlockIdKey struct {
	BlockID crypto.Signature
}

func (k *BlockIdKey) Bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = BlockIdKeyPrefix
	copy(buf[1:], k.BlockID[:])
	return buf
}

type BlockOffsetKey struct {
	BlockID crypto.Signature
}

func (k *BlockOffsetKey) Bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = BlockOffsetKeyPrefix
	copy(buf[1:], k.BlockID[:])
	return buf
}

type TxOffsetKey struct {
	TxID []byte
}

func (k *TxOffsetKey) Bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = TxOffsetKeyPrefix
	copy(buf[1:], k.TxID)
	return buf
}
