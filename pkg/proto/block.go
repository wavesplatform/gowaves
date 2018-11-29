package proto

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// Block is a block of the blockchain
type Block struct {
	Version                uint8
	Timestamp              uint64
	Parent                 crypto.Signature
	ConsensusBlockLength   uint32
	BaseTarget             uint64
	GenSignature           crypto.Digest
	TransactionBlockLength uint32
	TransactionCount       int
	Transactions           []byte `json:"-"`
	GenPublicKey           crypto.PublicKey
	BlockSignature         crypto.Signature

	Height uint64
}

// MarshalBinary encodes Block to binary form
func (b *Block) MarshalBinary() ([]byte, error) {
	res := make([]byte, 1+8+64+4+8+32+4)
	res[0] = b.Version
	binary.BigEndian.PutUint64(res[1:9], b.Timestamp)
	copy(res[9:], b.Parent[:])
	binary.BigEndian.PutUint32(res[73:77], b.ConsensusBlockLength)
	binary.BigEndian.PutUint64(res[77:85], b.BaseTarget)
	copy(res[85:117], b.GenSignature[:])
	binary.BigEndian.PutUint32(res[117:121], b.TransactionBlockLength)
	res = append(res, b.Transactions...)
	res = append(res, b.GenPublicKey[:]...)
	res = append(res, b.BlockSignature[:]...)

	return res, nil
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
	if b.Version == 3 {
		b.TransactionCount = int(binary.BigEndian.Uint32(data[121:125]))
		transBytes := data[125 : 125+b.TransactionBlockLength]
		b.Transactions = make([]byte, len(transBytes))
		copy(b.Transactions, transBytes)
	} else {
		b.TransactionCount = int(data[121])
		transBytes := data[122 : 122+b.TransactionBlockLength]
		b.Transactions = make([]byte, len(transBytes))
		copy(b.Transactions, transBytes)
	}

	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}
