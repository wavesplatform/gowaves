package state

import (
	"encoding/binary"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	LastHeightKeyPrefix byte = iota
	BlockAtHeightKeyPrefix
	TradesKeyPrefix
	CandlesKeyPrefix
	CandlesSecondKeyPrefix
	BlockTradesKeyPrefix
	EarliestHeightKeyPrefix
	PairTradesKeyPrefix
	PairPublicKeyTradesKeyPrefix
	AssetInfoKeyPrefix
	AssetInfoHistoryKeyPrefix
	PublicKeyToAddressKeyPrefix
	AddressToPublicKeyKeyPrefix
	MarketsKeyPrefix
	AliasToAddressKeyPrefix
	AliasHistoryKeyPrefix
	AssetIssuerKeyPrefix
	AssetBalanceKeyPrefix
	AssetBalanceHistoryKeyPrefix
)

type uint32Key struct {
	prefix byte
	key    uint32
}

func (k uint32Key) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = k.prefix
	binary.BigEndian.PutUint32(buf[1:], k.key)
	return buf
}

type DigestKey struct {
	prefix byte
	key    crypto.Digest
}

func (k DigestKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = k.prefix
	copy(buf[1:], k.key[:])
	return buf
}

func publicKeyKey(prefix byte, pk crypto.PublicKey) []byte {
	k := make([]byte, 1+crypto.PublicKeySize)
	k[0] = prefix
	copy(k[1:], pk[:])
	return k
}

func addressKey(prefix byte, address proto.Address) []byte {
	k := make([]byte, 1+proto.AddressSize)
	k[0] = prefix
	copy(k[1:], address[:])
	return k
}

func signatureKey(prefix byte, sig crypto.Signature) []byte {
	k := make([]byte, 1+crypto.SignatureSize)
	k[0] = prefix
	copy(k[1:], sig[:])
	return k
}

func uint32AndDigestKey(prefix byte, a uint32, b crypto.Digest) []byte {
	k := make([]byte, 1+4+crypto.DigestSize)
	k[0] = prefix
	binary.BigEndian.PutUint32(k[1:], a)
	copy(k[1+4:], b[:])
	return k
}
