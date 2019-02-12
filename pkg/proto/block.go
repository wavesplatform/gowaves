package proto

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// Block info (except transactions)
type BlockHeader struct {
	Version                uint8
	Timestamp              uint64
	Parent                 crypto.Signature
	ConsensusBlockLength   uint32
	BaseTarget             uint64
	GenSignature           crypto.Digest
	TransactionBlockLength uint32
	TransactionCount       int
	GenPublicKey           crypto.PublicKey
	BlockSignature         crypto.Signature

	Height uint64
}

func (b *BlockHeader) MarshalHeaderToBinary() ([]byte, error) {
	res := make([]byte, 1+8+64+4+8+32+4)
	res[0] = b.Version
	binary.BigEndian.PutUint64(res[1:9], b.Timestamp)
	copy(res[9:], b.Parent[:])
	binary.BigEndian.PutUint32(res[73:77], b.ConsensusBlockLength)
	binary.BigEndian.PutUint64(res[77:85], b.BaseTarget)
	copy(res[85:117], b.GenSignature[:])
	binary.BigEndian.PutUint32(res[117:121], b.TransactionBlockLength)
	if b.Version == 3 {
		countBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(countBuf, uint32(b.TransactionCount))
		res = append(res, countBuf...)
	} else {
		res = append(res, byte(b.TransactionCount))
	}
	res = append(res, b.GenPublicKey[:]...)
	res = append(res, b.BlockSignature[:]...)

	return res, nil
}

func (b *BlockHeader) UnmarshalHeaderFromBinary(data []byte) error {
	b.Version = data[0]
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	copy(b.Parent[:], data[9:73])
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	copy(b.GenSignature[:], data[85:117])
	b.TransactionBlockLength = binary.BigEndian.Uint32(data[117:121])
	if b.Version == 3 {
		b.TransactionCount = int(binary.BigEndian.Uint32(data[121:125]))
	} else {
		b.TransactionCount = int(data[121])
	}

	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}

// Block is a block of the blockchain
type Block struct {
	BlockHeader
	Transactions []byte `json:"-"`
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
	if b.Version == 3 {
		countBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(countBuf, uint32(b.TransactionCount))
		res = append(res, countBuf...)
	} else {
		res = append(res, byte(b.TransactionCount))
	}
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
		if b.TransactionBlockLength < 4 {
			return errors.New("TransactionBlockLength is too small")
		}
		transBytes := data[125 : 125+b.TransactionBlockLength-4]
		b.Transactions = make([]byte, len(transBytes))
		copy(b.Transactions, transBytes)
	} else {
		b.TransactionCount = int(data[121])
		if b.TransactionBlockLength < 1 {
			return errors.New("TransactionBlockLength is too small")
		}
		transBytes := data[122 : 122+b.TransactionBlockLength-1]
		b.Transactions = make([]byte, len(transBytes))
		copy(b.Transactions, transBytes)
	}

	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}
