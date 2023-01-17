package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/jinzhu/copier"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	protobuf "google.golang.org/protobuf/proto"
)

type BlockVersion byte

const (
	GenesisBlockVersion BlockVersion = iota + 1
	PlainBlockVersion
	NgBlockVersion
	RewardBlockVersion
	ProtobufBlockVersion
)

type Marshaller interface {
	Marshal(scheme Scheme) ([]byte, error)
}

type NxtConsensus struct {
	BaseTarget   uint64   `json:"base-target"`
	GenSignature B58Bytes `json:"generation-signature"`
}

func (nc *NxtConsensus) BinarySize() int {
	return 8 + len(nc.GenSignature)
}

type BlockIDType byte

const (
	SignatureID BlockIDType = iota
	DigestID
)

type BlockID struct {
	sig    crypto.Signature
	dig    crypto.Digest
	idType BlockIDType
}

func NewBlockIDFromBase58(s string) (BlockID, error) {
	dig, err := crypto.NewDigestFromBase58(s)
	if err != nil {
		sig, err := crypto.NewSignatureFromBase58(s)
		if err != nil {
			return BlockID{}, err
		}
		return NewBlockIDFromSignature(sig), nil
	}
	return NewBlockIDFromDigest(dig), nil
}

func MustBlockIDFromBase58(s string) BlockID {
	block, err := NewBlockIDFromBase58(s)
	if err != nil {
		panic(err)
	}
	return block
}

func NewBlockIDFromSignature(sig crypto.Signature) BlockID {
	return BlockID{sig: sig, idType: SignatureID}
}

func NewBlockIDFromDigest(dig crypto.Digest) BlockID {
	return BlockID{dig: dig, idType: DigestID}
}

func NewBlockIDFromBytes(data []byte) (BlockID, error) {
	res := BlockID{}
	if len(data) == crypto.SignatureSize {
		sig, err := crypto.NewSignatureFromBytes(data)
		if err != nil {
			return BlockID{}, err
		}
		res.sig = sig
		res.idType = SignatureID
	} else if len(data) == crypto.DigestSize {
		dig, err := crypto.NewDigestFromBytes(data)
		if err != nil {
			return BlockID{}, err
		}
		res.dig = dig
		res.idType = DigestID
	} else {
		return BlockID{}, errors.Errorf("NewBlockIDFromBytes: invalid data size %d", len(data))
	}
	return res, nil
}

func (id BlockID) IsValid(version BlockVersion) bool {
	if version >= ProtobufBlockVersion {
		return id.idType == DigestID
	}
	return id.idType == SignatureID
}

func (id BlockID) Bytes() []byte {
	if id.idType == SignatureID {
		return id.sig.Bytes()
	} else if id.idType == DigestID {
		return id.dig.Bytes()
	}
	return nil
}

func (id BlockID) IsSignature() bool {
	return id.idType == SignatureID
}

func (id BlockID) Signature() crypto.Signature {
	return id.sig
}

func (id BlockID) ShortString() string {
	if id.idType == SignatureID {
		return id.sig.ShortString()
	} else if id.idType == DigestID {
		return id.dig.ShortString()
	}
	return ""
}

func (id BlockID) String() string {
	return base58.Encode(id.Bytes())
}

func (id BlockID) MarshalJSON() ([]byte, error) {
	return common.ToBase58JSON(id.Bytes()), nil
}

func (id *BlockID) UnmarshalJSON(value []byte) error {
	b, err := common.FromBase58JSONUnchecked(value, "BlockID")
	if err != nil {
		return err
	}
	res, err := NewBlockIDFromBytes(b)
	if err != nil {
		return err
	}
	*id = res
	return nil
}

// BlockHeader contains Block meta-information without transactions
type BlockHeader struct {
	Version                BlockVersion `json:"version"`
	Timestamp              uint64       `json:"timestamp"`
	Parent                 BlockID      `json:"reference"`
	FeaturesCount          int          `json:"-"`
	Features               []int16      `json:"features,omitempty"`
	RewardVote             int64        `json:"desiredReward"`
	ConsensusBlockLength   uint32       `json:"-"`
	NxtConsensus           `json:"nxt-consensus"`
	TransactionBlockLength uint32           `json:"transactionBlockLength,omitempty"`
	TransactionCount       int              `json:"transactionCount"`
	GeneratorPublicKey     crypto.PublicKey `json:"generatorPublicKey"`
	BlockSignature         crypto.Signature `json:"signature"`
	TransactionsRoot       B58Bytes         `json:"transactionsRoot,omitempty"`

	ID BlockID `json:"id"` // this field must be generated and set after Block unmarshalling
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

func (b *BlockHeader) GenerateBlockID(scheme Scheme) error {
	if b.Version < ProtobufBlockVersion {
		b.ID = NewBlockIDFromSignature(b.BlockSignature)
		return nil
	}
	headerBytes, err := b.MarshalHeaderToProtobufWithoutSignature(scheme)
	if err != nil {
		return err
	}
	hash, err := crypto.FastHash(headerBytes)
	if err != nil {
		return err
	}
	b.ID = NewBlockIDFromDigest(hash)
	return nil
}

func (b *BlockHeader) BlockID() BlockID {
	if b.Version < ProtobufBlockVersion {
		return NewBlockIDFromSignature(b.BlockSignature)
	}
	return b.ID
}

func (b *BlockHeader) MarshalHeader(scheme Scheme) ([]byte, error) {
	if b.Version >= ProtobufBlockVersion {
		return b.MarshalHeaderToProtobuf(scheme)
	}
	return b.MarshalHeaderToBinary()
}

func (b *BlockHeader) MarshalHeaderToProtobufWithoutSignature(scheme Scheme) ([]byte, error) {
	header, err := b.HeaderToProtobufHeader(scheme)
	if err != nil {
		return nil, err
	}
	return MarshalToProtobufDeterministic(header)
}

func (b *BlockHeader) MarshalHeaderToProtobuf(scheme Scheme) ([]byte, error) {
	header, err := b.HeaderToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	return MarshalToProtobufDeterministic(header)
}

func (b *BlockHeader) HeaderToProtobufHeader(scheme Scheme) (*g.Block_Header, error) {
	ref := b.Parent.Bytes()
	pkBytes := b.GeneratorPublicKey.Bytes()
	features := make([]uint32, len(b.Features))
	for i := range b.Features {
		features[i] = uint32(b.Features[i])
	}
	return &g.Block_Header{
		ChainId:             int32(scheme),
		Reference:           ref,
		BaseTarget:          int64(b.BaseTarget),
		GenerationSignature: b.GenSignature.Bytes(),
		FeatureVotes:        features,
		Timestamp:           int64(b.Timestamp),
		Version:             int32(b.Version),
		Generator:           pkBytes,
		RewardVote:          b.RewardVote,
		TransactionsRoot:    b.TransactionsRoot,
	}, nil
}

func (b *BlockHeader) HeaderToProtobuf(scheme Scheme) (*g.Block, error) {
	sig, err := b.BlockSignature.MarshalBinary()
	if err != nil {
		return nil, err
	}
	header, err := b.HeaderToProtobufHeader(scheme)
	if err != nil {
		return nil, err
	}
	return &g.Block{
		Header:    header,
		Signature: sig,
	}, nil
}

func (b *BlockHeader) HeaderToProtobufWithHeight(currentScheme Scheme, height uint64) (*pb.BlockWithHeight, error) {
	block, err := b.HeaderToProtobuf(currentScheme)
	if err != nil {
		return nil, err
	}
	return &pb.BlockWithHeight{
		Block:  block,
		Height: uint32(height),
	}, nil
}

func (b *BlockHeader) MarshalHeaderToBinary() ([]byte, error) {
	if b.Version >= ProtobufBlockVersion {
		return nil, errors.New("BlockHeader.MarshalHeaderToBinary: binary format is not defined for Block versions > 4")
	}
	res := make([]byte, 1+8+64+4+8+32+4)
	res[0] = byte(b.Version)
	binary.BigEndian.PutUint64(res[1:9], b.Timestamp)
	parentBytes := b.Parent.Bytes()
	if len(parentBytes) != crypto.SignatureSize {
		return nil, errors.New("bad parent length for non-protobuf block")
	}
	copy(res[9:], parentBytes)
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
	res = append(res, b.GeneratorPublicKey[:]...)
	res = append(res, b.BlockSignature[:]...)

	return res, nil
}

func (b *BlockHeader) UnmarshalHeaderFromBinary(data []byte, scheme Scheme) (err error) {
	// TODO make benchmarks to figure out why multiple length checks slow down that much
	// and (probably) get rid of recover().
	defer func() {
		if recover() != nil {
			err = errors.New("invalid data size")
		}
	}()

	b.Version = BlockVersion(data[0])
	if b.Version >= ProtobufBlockVersion {
		return errors.New("binary format is not defined for Block versions > 4")
	}
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	parentBytes := data[9:73]
	parent, err := NewBlockIDFromBytes(parentBytes)
	if err != nil {
		return err
	}
	b.Parent = parent
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	b.GenSignature = make([]byte, crypto.DigestSize)
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
		b.Features = []int16{}
	}
	copy(b.GeneratorPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])
	if err := b.GenerateBlockID(scheme); err != nil {
		return errors.Wrap(err, "failed to generate block ID")
	}
	return nil
}

func AppendHeaderBytesToTransactions(headerBytes, transactions []byte) ([]byte, error) {
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
	Transactions Transactions `json:"transactions,omitempty"`
}

func (b *Block) Marshal(scheme Scheme) ([]byte, error) {
	if b.Version >= ProtobufBlockVersion {
		return b.MarshalToProtobuf(scheme)
	} else {
		return b.MarshalBinary(scheme)
	}
}

func (b *Block) Clone() *Block {
	out := &Block{}
	if err := copier.Copy(out, b); err != nil {
		panic(err.Error())
	}
	return out
}

func (b *Block) Sign(scheme Scheme, secret crypto.SecretKey) error {
	var bb []byte
	if b.Version >= ProtobufBlockVersion {
		b, err := b.MarshalHeaderToProtobufWithoutSignature(scheme)
		if err != nil {
			return err
		}
		bb = b
	} else {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if _, err := b.WriteToWithoutSignature(buf, scheme); err != nil {
			return err
		}
		bb = buf.Bytes()
	}
	sign, err := crypto.Sign(secret, bb)
	if err != nil {
		return err
	}
	b.BlockSignature = sign
	return nil
}

func (b *Block) SetTransactionsRoot(scheme Scheme) error {
	rh, err := b.transactionsRoot(scheme)
	if err != nil {
		return err
	}
	b.TransactionsRoot = rh
	return nil
}

func (b *Block) SetTransactionsRootIfPossible(scheme Scheme) error {
	if b.Version < ProtobufBlockVersion {
		return nil
	}
	return b.SetTransactionsRoot(scheme)
}

func (b *Block) VerifySignature(scheme Scheme) (bool, error) {
	var bb []byte
	if b.Version >= ProtobufBlockVersion {
		b, err := b.MarshalHeaderToProtobufWithoutSignature(scheme)
		if err != nil {
			return false, err
		}
		bb = b
	} else {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if _, err := b.WriteToWithoutSignature(buf, scheme); err != nil {
			return false, err
		}
		bb = buf.Bytes()
	}
	return crypto.Verify(b.GeneratorPublicKey, b.BlockSignature, bb), nil
}

func (b *Block) VerifyTransactionsRoot(scheme Scheme) (bool, error) {
	// For old versions of Block always return true
	if b.Version < ProtobufBlockVersion {
		return true, nil
	}
	rh, err := b.transactionsRoot(scheme)
	if err != nil {
		return false, err
	}
	return bytes.Equal(b.BlockHeader.TransactionsRoot, rh), nil
}

// MarshalBinary encodes Block to binary form
func (b *Block) MarshalBinary(scheme Scheme) ([]byte, error) {
	if b.Version >= ProtobufBlockVersion {
		return nil, errors.New("binary format is not defined for Block versions > 4")
	}
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := b.WriteTo(buf, scheme)
	if err != nil {
		return nil, err
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func (b *Block) WriteTo(w io.Writer, scheme Scheme) (int64, error) {
	if b.Version >= ProtobufBlockVersion {
		return 0, errors.New("binary format is not defined for Block versions > 4")
	}
	n, err := b.WriteToWithoutSignature(w, scheme)
	if err != nil {
		return 0, err
	}

	n2, err := w.Write(b.BlockSignature[:])
	if err != nil {
		return 0, err
	}

	return n + int64(n2), nil
}

func (b *Block) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	pbBlock, err := b.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	return MarshalToProtobufDeterministic(pbBlock)
}

func (b *Block) Marshaller() Marshaller {
	return BlockMarshaller{
		b: b,
	}
}

func (b *Block) UnmarshalFromProtobuf(data []byte) error {
	var pbBlock g.Block
	if err := protobuf.Unmarshal(data, &pbBlock); err != nil {
		return err
	}
	var c ProtobufConverter
	res, err := c.Block(&pbBlock)
	if err != nil {
		return err
	}
	*b = res
	return nil
}

func (b *Block) ToProtobuf(scheme Scheme) (*g.Block, error) {
	protoBlock, err := b.BlockHeader.HeaderToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	protoTransactions, err := b.Transactions.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	protoBlock.Transactions = protoTransactions
	return protoBlock, nil
}

func (b *Block) ToProtobufWithHeight(currentScheme Scheme, height uint64) (*pb.BlockWithHeight, error) {
	block, err := b.BlockHeader.HeaderToProtobufWithHeight(currentScheme, height)
	if err != nil {
		return nil, err
	}
	txs, err := b.Transactions.ToProtobuf(currentScheme)
	if err != nil {
		return nil, err
	}
	block.Block.Transactions = txs
	return block, nil
}

// WriteToWithoutSignature writes binary representation of block into Writer.
// It does not sign and write signature.
func (b *Block) WriteToWithoutSignature(w io.Writer, scheme Scheme) (int64, error) {
	if b.Version >= ProtobufBlockVersion {
		return 0, errors.New("binary format is not defined for Block versions > 4")
	}
	s := serializer.NewNonFallable(w)
	s.Byte(byte(b.Version))
	s.Uint64(b.Timestamp)
	parentBytes := b.Parent.Bytes()
	if len(parentBytes) != crypto.SignatureSize {
		return 0, errors.New("bad parent length for non-protobuf block")
	}
	s.Bytes(parentBytes)
	s.Uint32(b.ConsensusBlockLength)
	s.Uint64(b.BaseTarget)
	s.Bytes(b.GenSignature[:])

	// transactions
	s.Uint32(b.TransactionBlockLength)
	if b.Version >= NgBlockVersion {
		s.Uint32(uint32(b.TransactionCount))
	} else {
		s.Byte(byte(b.TransactionCount))
	}
	if _, err := b.Transactions.WriteToBinary(s, scheme); err != nil {
		return 0, err
	}

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

	s.Bytes(b.GeneratorPublicKey[:])
	return s.N(), nil
}

// UnmarshalBinary decodes Block from binary form
func (b *Block) UnmarshalBinary(data []byte, scheme Scheme) (err error) {
	// TODO make benchmarks to figure out why multiple length checks slow down that much
	//  and (probably) get rid of recover().
	defer func() {
		if recover() != nil {
			err = errors.New("invalid data size")
		}
	}()

	b.Version = BlockVersion(data[0])
	if b.Version >= ProtobufBlockVersion {
		return errors.New("binary format is not defined for Block versions > 4")
	}
	b.Timestamp = binary.BigEndian.Uint64(data[1:9])
	parentBytes := data[9:73]
	parent, err := NewBlockIDFromBytes(parentBytes)
	if err != nil {
		return err
	}
	b.Parent = parent
	b.ConsensusBlockLength = binary.BigEndian.Uint32(data[73:77])
	b.BaseTarget = binary.BigEndian.Uint64(data[77:85])
	b.GenSignature = make([]byte, crypto.DigestSize)
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
		b.Transactions, err = NewTransactionsFromBytes(transBytes, b.TransactionCount, scheme)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal transactions from bytes")
		}
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
		b.Transactions, err = NewTransactionsFromBytes(transBytes, b.TransactionCount, scheme)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal transactions from bytes")
		}
		b.Features = []int16{}
	}

	copy(b.GeneratorPublicKey[:], data[len(data)-64-32:len(data)-64])
	copy(b.BlockSignature[:], data[len(data)-64:])
	if err := b.GenerateBlockID(scheme); err != nil {
		return errors.Wrap(err, "failed to generate block ID")
	}
	return nil
}

func (b *Block) transactionsRoot(scheme Scheme) ([]byte, error) {
	if b.Version < ProtobufBlockVersion {
		return nil, errors.Errorf("no transactions root prior block version %d, current version %d", ProtobufBlockVersion, b.Version)
	}
	tree, err := crypto.NewMerkleTree()
	if err != nil {
		return nil, err
	}
	for _, tx := range b.Transactions {
		b, err := tx.MerkleBytes(scheme)
		if err != nil {
			return nil, err
		}
		tree.Push(b)
	}
	return tree.Root().Bytes(), nil
}

func CreateBlock(transactions Transactions, timestamp Timestamp, parentID BlockID, publicKey crypto.PublicKey, nxtConsensus NxtConsensus, version BlockVersion, features []int16, rewardVote int64, scheme Scheme) (*Block, error) {
	consensusLength := nxtConsensus.BinarySize()
	b := &Block{
		BlockHeader: BlockHeader{
			Version:              version,
			Timestamp:            timestamp,
			Parent:               parentID,
			FeaturesCount:        len(features),
			Features:             features,
			RewardVote:           rewardVote,
			ConsensusBlockLength: uint32(consensusLength),
			NxtConsensus:         nxtConsensus,
			TransactionCount:     transactions.Count(),
			GeneratorPublicKey:   publicKey,
		},
		Transactions: transactions,
	}
	switch {
	case version < NgBlockVersion:
		b.TransactionBlockLength = uint32(transactions.BinarySize() + 1) // add extra sizeof(byte) == 1 bytes for version
	case version <= RewardBlockVersion:
		b.TransactionBlockLength = uint32(transactions.BinarySize() + 4) // add extra sizeof(int) == 4 bytes for version
	case version >= ProtobufBlockVersion:
		err := b.SetTransactionsRoot(scheme)
		if err != nil {
			return nil, err
		}
	}
	if err := b.GenerateBlockID(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to generate block ID")
	}
	return b, nil
}

// BlockGetSignature get signature from block without deserialization
func BlockGetSignature(data []byte) (crypto.Signature, error) {
	sig := crypto.Signature{}
	if len(data) < crypto.SignatureSize {
		return sig, errors.Errorf("not enough bytes to decode block signature, want at least %d, found %d",
			crypto.SignatureSize, len(data),
		)
	}
	copy(sig[:], data[len(data)-crypto.SignatureSize:])
	return sig, nil
}

type BlockMarshaller struct {
	b *Block
}

func (a BlockMarshaller) Marshal(scheme Scheme) ([]byte, error) {
	if a.b.Version >= ProtobufBlockVersion {
		return a.b.MarshalToProtobuf(scheme)
	} else {
		return a.b.MarshalBinary(scheme)
	}
}

type Transactions []Transaction

func NewTransactionsFromBytes(data []byte, count int, scheme Scheme) (Transactions, error) {
	return BytesToTransactions(count, data, scheme)
}

func (a Transactions) BinarySize() int {
	size := 0
	for _, tx := range a {
		size += 4 + tx.BinarySize()
	}
	return size
}

func (a Transactions) MarshalBinary(scheme Scheme) ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.WriteToBinary(buf, scheme)
	if err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func (a Transactions) WriteTo(proto bool, scheme Scheme, w io.Writer) (int64, error) {
	if !proto {
		return a.WriteToBinary(w, scheme)
	}
	s := serializer.New(w)
	for _, t := range a {
		bts, err := t.MarshalSignedToProtobuf(scheme)
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

func (a Transactions) WriteToBinary(w io.Writer, scheme Scheme) (int64, error) {
	s := serializer.New(w)
	for _, t := range a {
		bts, err := t.MarshalBinary(scheme)
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

func (a Transactions) ToProtobuf(scheme Scheme) ([]*g.SignedTransaction, error) {
	protoTransactions := make([]*g.SignedTransaction, len(a))
	for i, tx := range a {
		protoTx, err := tx.ToProtobufSigned(scheme)
		if err != nil {
			return nil, err
		}
		protoTransactions[i] = protoTx
	}
	return protoTransactions, nil
}

func (a *Transactions) UnmarshalFromProtobuf(data []byte) error {
	transactions := Transactions{}
	for len(data) > 0 {
		txSize := int(binary.BigEndian.Uint32(data[0:4]))
		if txSize+4 > len(data) {
			return errors.New("invalid data size")
		}
		txBytes := data[4 : txSize+4]
		tx, err := SignedTxFromProtobuf(txBytes)
		if err != nil {
			return err
		}
		transactions = append(transactions, tx)
		data = data[txSize+4:]
	}
	*a = transactions
	return nil
}

func (a Transactions) Join(other Transactions) Transactions {
	newTransactions := make([]Transaction, other.Count()+a.Count())
	copy(newTransactions, a)
	copy(newTransactions[a.Count():], other)
	return newTransactions
}

func (a Transactions) Count() int {
	return len(a)
}

func (a Transactions) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]Transaction(a))
}

func (a *Transactions) UnmarshalJSON(data []byte) error {
	var tt []*TransactionTypeVersion
	err := json.Unmarshal(data, &tt)
	if err != nil {
		return errors.Wrap(err, "TransactionType unmarshal")
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

	*a = transactions
	return nil
}
