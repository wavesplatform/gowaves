package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

type BlockVersion byte

const (
	GenesisBlockVersion BlockVersion = iota + 1
	PlainBlockVersion
	NgBlockVersion
	RewardBlockVersion
)

type NxtConsensus struct {
	BaseTarget   uint64        `json:"base-target"`
	GenSignature crypto.Digest `json:"generation-signature"`
}

// Block info (except transactions)
type BlockHeader struct {
	Version                BlockVersion     `json:"version"`
	Timestamp              uint64           `json:"timestamp"`
	Parent                 crypto.Signature `json:"reference"`
	FeaturesCount          int              `json:"-"`
	Features               []int16          `json:"features,omitempty"`
	RewardVote             int64            `json:"desiredReward"`
	ConsensusBlockLength   uint32           `json:"-"`
	NxtConsensus           `json:"nxt-consensus"`
	TransactionBlockLength uint32           `json:"transactionBlockLength,omitempty"`
	TransactionCount       int              `json:"transactionCount"`
	GenPublicKey           crypto.PublicKey `json:"-"`
	BlockSignature         crypto.Signature `json:"signature"`

	Height uint64 `json:"height"`
}

func featuresToBinary(features []int16) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, features); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func featuresFromBinary(data []byte) ([]int16, error) {
	res := make([]int16, len(data)/2)
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.BigEndian, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (b *BlockHeader) MarshalHeaderToBinary() ([]byte, error) {
	res := make([]byte, 1+8+64+4+8+32+4)
	res[0] = byte(b.Version)
	binary.BigEndian.PutUint64(res[1:9], b.Timestamp)
	copy(res[9:], b.Parent[:])
	binary.BigEndian.PutUint32(res[73:77], b.ConsensusBlockLength)
	binary.BigEndian.PutUint64(res[77:85], b.BaseTarget)
	copy(res[85:117], b.GenSignature[:])
	binary.BigEndian.PutUint32(res[117:121], b.TransactionBlockLength)
	if b.Version >= NgBlockVersion {
		// Add tx count and features count.
		buf := make([]byte, 8)
		binary.BigEndian.PutUint32(buf[:4], uint32(b.TransactionCount))
		binary.BigEndian.PutUint32(buf[4:], uint32(b.FeaturesCount))
		res = append(res, buf...)
		// Add features.
		fb, err := featuresToBinary(b.Features)
		if err != nil {
			return nil, err
		}
		res = append(res, fb...)
		if b.Version >= RewardBlockVersion {
			rvb := make([]byte, 8)
			binary.BigEndian.PutUint64(rvb, uint64(b.RewardVote))
			res = append(res, rvb...)
		}
	} else {
		res = append(res, byte(b.TransactionCount))
	}
	res = append(res, b.GenPublicKey[:]...)
	res = append(res, b.BlockSignature[:]...)

	return res, nil
}

func (b *BlockHeader) UnmarshalHeaderFromBinary(data []byte) (err error) {
	// TODO make benchmarks to figure out why multiple length checks slow down that much
	// and (probably) get rid of recover().
	defer func() {
		if recover() != nil {
			err = errors.New("invalid data size")
		}
	}()

	b.Version = BlockVersion(data[0])
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	copy(b.Parent[:], data[9:73])
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	copy(b.GenSignature[:], data[85:117])
	b.TransactionBlockLength = binary.BigEndian.Uint32(data[117:121])
	b.RewardVote = -1
	if b.Version >= NgBlockVersion {
		if b.TransactionBlockLength < 4 {
			return errors.New("TransactionBlockLength is too small")
		}
		b.TransactionCount = int(binary.BigEndian.Uint32(data[121:125]))
		b.FeaturesCount = int(binary.BigEndian.Uint32(data[125:129]))
		b.Features = make([]int16, b.FeaturesCount)
		fb, err := featuresFromBinary(data[129 : 129+2*b.FeaturesCount])
		if err != nil {
			return errors.Wrap(err, "failed to convert features from binary representation")
		}
		copy(b.Features, fb)
		if b.Version >= RewardBlockVersion {
			pos := 129 + 2*b.FeaturesCount
			b.RewardVote = int64(binary.BigEndian.Uint64(data[pos : pos+8]))
		}
	} else {
		if b.TransactionBlockLength < 1 {
			return errors.New("TransactionBlockLength is too small")
		}
		b.TransactionCount = int(data[121])
	}
	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}

func AppendHeaderBytesToTransactions(headerBytes []byte, transactions []byte) ([]byte, error) {
	headerLen := len(headerBytes)
	if headerLen < 1 {
		return nil, errors.New("insufficient header data size")
	}
	featuresSize := 0
	version := BlockVersion(headerBytes[0])
	if version >= NgBlockVersion {
		if len(headerBytes) < 129 {
			return nil, errors.New("insufficient header data size")
		}
		featuresCount := int(binary.BigEndian.Uint32(headerBytes[125:129]))
		// featuresCount * int16 + int for featuresCount itself.
		featuresSize = featuresCount*2 + 4
	}
	if headerLen < crypto.PublicKeySize+crypto.SignatureSize+featuresSize {
		return nil, errors.New("insufficient header data size")
	}
	headerBeforeTx := headerBytes[:headerLen-crypto.PublicKeySize-crypto.SignatureSize-featuresSize]
	headerAfterTx := headerBytes[headerLen-crypto.PublicKeySize-crypto.SignatureSize-featuresSize:]
	res := make([]byte, headerLen+len(transactions))
	copy(res, headerBeforeTx)
	filled := len(headerBeforeTx)
	copy(res[filled:], transactions)
	filled += len(transactions)
	copy(res[filled:], headerAfterTx)
	return res, nil
}

// Block is a block of the blockchain
type Block struct {
	BlockHeader
	Transactions *TransactionsRepresentation `json:"transactions,omitempty"`
}

func (b *Block) Clone() *Block {
	out := &Block{}
	_ = copier.Copy(out, b)
	return out
}

const bytesV = 1
const transactionsV = 2

type TransactionsRepresentation struct {
	v            byte
	bytes        []byte
	transactions Transactions
	count        int
}

func NewReprFromBytes(b []byte, count int) *TransactionsRepresentation {
	bytes := make([]byte, len(b))
	copy(bytes, b)
	return &TransactionsRepresentation{
		bytes: bytes,
		v:     bytesV,
		count: count,
	}
}

func NewReprFromTransactions(t Transactions) *TransactionsRepresentation {
	return &TransactionsRepresentation{
		transactions: t,
		v:            transactionsV,
		count:        t.Count(),
	}
}

func (a *TransactionsRepresentation) Bytes() ([]byte, error) {
	if a == nil {
		return nil, nil
	}
	if a.v&bytesV > 0 {
		return a.bytes, nil
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.transactions.WriteTo(buf)
	if err != nil {
		return nil, err
	}

	a.bytes = append([]byte(nil), buf.Bytes()...)
	a.v = a.v | bytesV
	return a.bytes, nil
}

func (a *TransactionsRepresentation) BytesUnchecked() []byte {
	return a.bytes
}

func (a *TransactionsRepresentation) Transactions() (Transactions, error) {
	if a == nil {
		return nil, nil
	}
	if a.v&transactionsV > 0 {
		return a.transactions, nil
	}

	bts := a.bytes
	for i := 0; i < a.count; i++ {
		if len(bts) < 4 {
			return nil, errors.New("invalid tx size: exceeds bytes slice bounds")
		}
		n := int(binary.BigEndian.Uint32(bts[0:4]))
		if n+4 > len(bts) {
			return nil, errors.New("invalid tx size: exceeds bytes slice bounds")
		}
		txBytes := bts[4 : n+4]
		tx, err := BytesToTransaction(txBytes)
		if err != nil {
			return nil, err
		}
		a.transactions = append(a.transactions, tx)
		bts = bts[4+n:]
	}

	a.v = a.v | transactionsV
	return a.transactions, nil
}

func (a *TransactionsRepresentation) Count() int {
	if a == nil {
		return 0
	}
	return a.count
}

func (a TransactionsRepresentation) Join(other *TransactionsRepresentation) (*TransactionsRepresentation, error) {
	b1, err := a.Bytes()
	if err != nil {
		return nil, err
	}
	b2, err := other.Bytes()
	if err != nil {
		return nil, err
	}
	return NewReprFromBytes(append(b1, b2...), a.Count()+other.Count()), nil
}

func (a TransactionsRepresentation) MarshalJSON() ([]byte, error) {
	transactions, err := a.Transactions()
	if len(transactions) == 0 {
		return []byte("[]"), nil
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(transactions)
}

func (a *TransactionsRepresentation) UnmarshalJSON(data []byte) error {
	var tt []*TransactionTypeVersion
	err := json.Unmarshal(data, &tt)
	if err != nil {
		return errors.Wrap(err, "TransactionTypeVersion unmarshal")
	}
	transactions := make([]Transaction, len(tt))
	for i, row := range tt {
		realType, err := GuessTransactionType(row)
		if err != nil {
			return err
		}
		transactions[i] = realType
	}
	err = json.Unmarshal(data, &transactions)
	if err != nil {
		return err
	}

	*a = *NewReprFromTransactions(transactions)
	return nil
}

func (b *Block) Sign(secret crypto.SecretKey) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := b.WriteToWithoutSignature(buf)
	if err != nil {
		return err
	}
	sign, err := crypto.Sign(secret, buf.Bytes())
	if err != nil {
		return err
	}
	b.BlockSignature = sign
	if _, err := buf.Write(sign[:]); err != nil {
		return err
	}
	return nil
}

func (b *Block) VerifySignature() (bool, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := b.WriteToWithoutSignature(buf)
	if err != nil {
		return false, err
	}
	return crypto.Verify(b.GenPublicKey, b.BlockSignature, buf.Bytes()), nil
}

// MarshalBinary encodes Block to binary form
func (b *Block) MarshalBinary() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := b.WriteTo(buf)
	if err != nil {
		return nil, err
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func (b *Block) WriteTo(w io.Writer) (int64, error) {
	n, err := b.WriteToWithoutSignature(w)
	if err != nil {
		return 0, err
	}

	n2, err := w.Write(b.BlockSignature[:])
	if err != nil {
		return 0, err
	}

	return n + int64(n2), nil
}

//WriteToWithoutSignature writes binary representation of block into Writer.
//It does not sign and write signature.
func (b *Block) WriteToWithoutSignature(w io.Writer) (int64, error) {
	s := serializer.NewNonFallable(w)
	s.Byte(byte(b.Version))
	s.Uint64(b.Timestamp)
	s.Bytes(b.Parent[:])
	s.Uint32(b.ConsensusBlockLength)
	s.Uint64(b.BaseTarget)
	s.Bytes(b.GenSignature[:])

	transactionsBytes, err := b.Transactions.Bytes()
	if err != nil {
		return 0, err
	}
	// transactions
	s.Uint32(b.TransactionBlockLength)
	if b.Version >= NgBlockVersion {
		s.Uint32(uint32(b.TransactionCount))
	} else {
		s.Byte(byte(b.TransactionCount))
	}
	s.Bytes(transactionsBytes)

	// features
	if b.Version >= NgBlockVersion {
		s.Uint32(uint32(b.FeaturesCount))
		fb, err := featuresToBinary(b.Features)
		if err != nil {
			return 0, err
		}
		s.Bytes(fb)
		if b.Version >= RewardBlockVersion {
			s.Int64(b.RewardVote)
		}
	}

	s.Bytes(b.GenPublicKey[:])
	return s.N(), nil
}

// UnmarshalBinary decodes Block from binary form
func (b *Block) UnmarshalBinary(data []byte) (err error) {
	// TODO make benchmarks to figure out why multiple length checks slow down that much
	//  and (probably) get rid of recover().
	defer func() {
		if recover() != nil {
			err = errors.New("invalid data size")
		}
	}()

	b.Version = BlockVersion(data[0])
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	copy(b.Parent[:], data[9:73])
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	copy(b.GenSignature[:], data[85:117])

	b.TransactionBlockLength = binary.BigEndian.Uint32(data[117:121])
	b.RewardVote = -1
	if b.Version >= NgBlockVersion {
		if b.TransactionBlockLength < 4 {
			return errors.New("TransactionBlockLength is too small")
		}
		b.TransactionCount = int(binary.BigEndian.Uint32(data[121:125]))
		txEnd := 121 + b.TransactionBlockLength
		transBytes := data[125:txEnd]
		b.Transactions = NewReprFromBytes(transBytes, b.TransactionCount)
		featuresStart := txEnd + 4
		b.FeaturesCount = int(binary.BigEndian.Uint32(data[txEnd:featuresStart]))
		b.Features = make([]int16, b.FeaturesCount)
		fb, err := featuresFromBinary(data[featuresStart : featuresStart+uint32(2*b.FeaturesCount)])
		if err != nil {
			return errors.Wrap(err, "failed to convert features from binary representation")
		}
		copy(b.Features, fb)
		if b.Version >= RewardBlockVersion {
			pos := featuresStart + uint32(2*b.FeaturesCount)
			b.RewardVote = int64(binary.BigEndian.Uint64(data[pos : pos+8]))
		}
	} else {
		if b.TransactionBlockLength < 1 {
			return errors.New("TransactionBlockLength is too small")
		}
		b.TransactionCount = int(data[121])
		transBytes := data[122 : 122+b.TransactionBlockLength-1]
		b.Transactions = NewReprFromBytes(transBytes, b.TransactionCount)
	}

	copy(b.GenPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])

	return nil
}

func CreateBlock(transactions *TransactionsRepresentation, timestamp Timestamp, parentSig crypto.Signature, publicKey crypto.PublicKey, NxtConsensus NxtConsensus, version BlockVersion, Features []int16, RewardVote int64) (*Block, error) {
	txb, err := transactions.Bytes()
	if err != nil {
		return nil, err
	}

	b := Block{
		BlockHeader: BlockHeader{
			Version:                version,
			Timestamp:              timestamp,
			Parent:                 parentSig,
			FeaturesCount:          len(Features),
			Features:               Features,
			RewardVote:             RewardVote,
			ConsensusBlockLength:   40, // digest size (32) + uint64 (8)
			TransactionBlockLength: uint32(len(txb) + 4),
			TransactionCount:       transactions.Count(),
			GenPublicKey:           publicKey,
			NxtConsensus:           NxtConsensus,
		},
		Transactions: transactions,
	}

	return &b, nil
}

func BlockEncodeJson(b *Block) ([]byte, error) {
	other := *b
	if b.Transactions.Count() == 0 {
		other.Transactions = nil
	}
	bts, err := json.Marshal(other)
	if err != nil {
		return nil, err
	}
	return bts, nil
}

//BlockGetSignature get signature from block without deserialization
func BlockGetSignature(data []byte) (crypto.Signature, error) {
	sig := crypto.Signature{}
	if len(data) < 64 {
		return sig, errors.Errorf("not enough bytes to decode block signature, want at least 64, found %d", len(data))
	}
	copy(sig[:], data[len(data)-64:])
	return sig, nil
}

//BlockGetParent get parent signature from block without deserialization
func BlockGetParent(data []byte) (crypto.Signature, error) {
	parent := crypto.Signature{}
	if len(data) < 73 {
		return parent, errors.Errorf("not enough bytes to decode block parent signature, want at least 73, found %d", len(data))
	}
	copy(parent[:], data[9:73])
	return parent, nil
}

type Transactions []Transaction

func (a Transactions) WriteTo(w io.Writer) (int64, error) {
	s := serializer.New(w)
	for _, t := range a {
		bts, err := t.MarshalBinary()
		if err != nil {
			return 0, err
		}

		err = s.Uint32(uint32(len(bts)))
		if err != nil {
			return 0, err
		}

		err = s.Bytes(bts)
		if err != nil {
			return 0, err
		}
	}
	return s.N(), nil
}

func (a Transactions) Count() int {
	return len(a)
}
