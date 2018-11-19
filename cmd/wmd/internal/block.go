package internal

import (
	"encoding/binary"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type Block struct {
	Version                uint8
	Timestamp              uint64
	Parent                 crypto.Signature
	ConsensusBlockLength   uint32
	BaseTarget             uint64
	GenSignature           crypto.Digest
	TransactionBlockLength uint32
	Transactions           []byte
	GenPublicKey           crypto.PublicKey
	BlockSignature         crypto.Signature
}

// UnmarshalBinary decodes Block from binary form
func (b *Block) UnmarshalBinary(data []byte) error {
	b.Version = data[0]
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	copy(b.Parent[:], data[9:73])
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	copy(b.GenSignature[:], data[85:117])
	b.TransactionBlockLength = binary.BigEndian.Uint32(data[117:121])
	transBytes := data[121 : 121+len(data[121:])-32-64]
	b.Transactions = make([]byte, len(transBytes))
	copy(b.Transactions, transBytes)
	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])
	return nil
}
