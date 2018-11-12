package proto

import (
	"encoding/binary"
)

// Block is a block of the blockchain
type Block struct {
	Version                uint8
	Timestamp              uint64
	Parent                 BlockID
	ConsensusBlockLength   uint32
	BaseTarget             uint64
	GenSignature           [32]byte
	TransactionBlockLength uint32
	Transactions           []byte
	GenPublicKey           [32]byte
	BlockSignature         BlockID
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
	copy(b.Transactions, data[121:])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}
