package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	stderrs "errors"
	"fmt"
	"io"
	"slices"

	"github.com/ccoveille/go-safecast/v2"
	"github.com/jinzhu/copier"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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

var ErrInvalidBlockIDDataSize = errors.New("invalid data size for BlockID")

type BlockIDType byte

const (
	SignatureID BlockIDType = iota + 1
	DigestID
)

type BlockID struct {
	sig    crypto.Signature
	dig    crypto.Digest
	idType BlockIDType
}

func NewBlockIDFromBase58(s string) (BlockID, error) {
	// The Base58 encoded Signature can take up to Ceil(64 * log2(256)/log2(58)) = Ceil(64 * 1.37) = 88 characters.
	const maxExpectedStringLength = 88
	if len(s) > maxExpectedStringLength {
		return BlockID{}, ErrInvalidBlockIDDataSize
	}
	d, err := base58.Decode(s)
	if err != nil {
		return BlockID{}, err
	}
	switch len(d) {
	case crypto.SignatureSize:
		sig := crypto.Signature{}
		copy(sig[:], d)
		return NewBlockIDFromSignature(sig), nil
	case crypto.DigestSize:
		dig := crypto.Digest{}
		copy(dig[:], d)
		return NewBlockIDFromDigest(dig), nil
	default:
		return BlockID{}, ErrInvalidBlockIDDataSize
	}
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
	switch len(data) {
	case crypto.SignatureSize:
		sig, err := crypto.NewSignatureFromBytes(data)
		if err != nil {
			return BlockID{}, err
		}
		res.sig = sig
		res.idType = SignatureID
	case crypto.DigestSize:
		dig, err := crypto.NewDigestFromBytes(data)
		if err != nil {
			return BlockID{}, err
		}
		res.dig = dig
		res.idType = DigestID
	default:
		return BlockID{}, ErrInvalidBlockIDDataSize
	}
	return res, nil
}

// IsValid checks if BlockID is valid for the given BlockVersion.
// Uninitialized BlockID is always not valid.
func (id BlockID) IsValid(version BlockVersion) bool {
	if version >= ProtobufBlockVersion {
		return id.idType == DigestID
	}
	return id.idType == SignatureID
}

// Bytes returns the slice of bytes of BlockID.
// If BlockID is not initialized, nil is returned.
func (id BlockID) Bytes() []byte {
	switch id.idType {
	case SignatureID:
		return id.sig.Bytes()
	case DigestID:
		return id.dig.Bytes()
	default:
		return nil
	}
}

// AppendBytes appends the bytes of BlockID to dst and returns the extended slice.
// Unlike Bytes, it does not allocate when dst has enough capacity.
// If BlockID is not initialized, dst is returned unchanged.
func (id BlockID) AppendBytes(dst []byte) []byte {
	switch id.idType {
	case SignatureID:
		return append(dst, id.sig[:]...)
	case DigestID:
		return append(dst, id.dig[:]...)
	default:
		return dst
	}
}

// IsSignature returns true if BlockID is a Signature.
func (id BlockID) IsSignature() bool {
	return id.idType == SignatureID
}

// Signature returns BlockID as a Signature.
// If BlockID is not a Signature, empty Signature is returned.
func (id BlockID) Signature() crypto.Signature {
	if id.idType == SignatureID {
		return id.sig
	}
	return crypto.Signature{}
}

// ShortString returns a short string representation of BlockID.
// If BlockID is not initialized, empty string is returned.
func (id BlockID) ShortString() string {
	switch id.idType {
	case SignatureID:
		return id.sig.ShortString()
	case DigestID:
		return id.dig.ShortString()
	default:
		return ""
	}
}

// String returns a string representation of BlockID.
func (id BlockID) String() string {
	return base58.Encode(id.Bytes())
}

func (id BlockID) MarshalJSON() ([]byte, error) {
	data := id.Bytes()
	if data == nil { // intentionally using nil to represent null BlockID
		return []byte(jsonNull), nil
	}
	return common.ToBase58JSON(data), nil
}

func (id *BlockID) UnmarshalJSON(value []byte) error {
	if bytes.Equal(value, jsonNullBytes) {
		*id = BlockID{} // initialize as nil BlockID
		return nil
	}
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

// WriteTo writes binary representation of BlockID into Writer. It writes only Digest or Signature bytes.
func (id BlockID) WriteTo(w io.Writer) (int64, error) {
	var n int
	var err error
	switch id.idType {
	case SignatureID:
		n, err = w.Write(id.sig[:])
	case DigestID:
		n, err = w.Write(id.dig[:])
	default:
		return 0, errors.New("undefined BlockID type")
	}
	return int64(n), err
}

func (id BlockID) Len() int {
	switch id.idType {
	case SignatureID:
		return crypto.SignatureSize
	case DigestID:
		return crypto.DigestSize
	default:
		return 0
	}
}

// ReadFrom reads the binary representation of BlockID from an io.Reader. It reads only the content of the ID
// (either crypto.Digest or crypto.Signature). ReadFrom does not process any additional data that might
// describe the type of the ID.
//
// If the BlockID instance has been pre-initialized with a type, ReadFrom will read data of that specific type.
// If the BlockID instance has not been initialized, ReadFrom will attempt to read 32 bytes twice
// and determine the type of ID upon successful reading of those parts.
//
// If the data size is less than 32 bytes or does not match the exact size of a crypto.Digest or crypto.Signature,
// an BlockIDDataSizeError error will be returned.
func (id *BlockID) ReadFrom(r io.Reader) (int64, error) {
	switch id.idType {
	case SignatureID:
		return id.readSignatureFrom(r)
	case DigestID:
		return id.readDigestFrom(r)
	default:
		return id.readUndefinedFrom(r)
	}
}

func (id *BlockID) readSignatureFrom(r io.Reader) (int64, error) {
	n, err := io.ReadFull(r, id.sig[:])
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return int64(n), stderrs.Join(ErrInvalidBlockIDDataSize, err)
		}
		return int64(n), err
	}
	return int64(n), nil
}

func (id *BlockID) readDigestFrom(r io.Reader) (int64, error) {
	n, err := io.ReadFull(r, id.dig[:])
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return int64(n), stderrs.Join(ErrInvalidBlockIDDataSize, err)
		}
		return int64(n), err
	}
	return int64(n), nil
}

func (id *BlockID) readUndefinedFrom(r io.Reader) (int64, error) {
	lr := io.LimitReader(r, crypto.SignatureSize)
	n1, err := io.ReadFull(lr, id.dig[:])
	if err != nil { // Not enough data to read even a digest.
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return int64(n1), stderrs.Join(ErrInvalidBlockIDDataSize, err)
		}
		return int64(n1), err
	}
	n2, err := io.ReadFull(lr, id.sig[crypto.DigestSize:])
	if err != nil { // Not enough data to read a second half of a signature.
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			// Return ID as a Digest. Also note that we return real number of bytes read.
			id.idType = DigestID
			return int64(n1 + n2), nil
		}
		return int64(n1 + n2), err
	}
	// We have read 64 bytes, so it's a signature.
	id.idType = SignatureID
	copy(id.sig[:crypto.DigestSize], id.dig[:])
	id.dig = crypto.Digest{}
	return int64(n1 + n2), nil
}

func (id *BlockID) IsPayload() {}

func (id BlockID) Equals(other BlockID) bool {
	if id.idType != other.idType {
		return false
	}
	switch id.idType {
	case SignatureID:
		return id.sig == other.sig
	case DigestID:
		return id.dig == other.dig
	}
	return false
}

type ChallengedHeader struct {
	Timestamp uint64 `json:"timestamp"`
	NxtConsensus
	Features           []int16          `json:"features,omitempty"`
	GeneratorPublicKey crypto.PublicKey `json:"generatorPublicKey"`
	RewardVote         int64            `json:"desiredReward"`
	StateHash          crypto.Digest    `json:"stateHash"`
	BlockSignature     crypto.Signature `json:"headerSignature"`
}

func int16SliceToUint32(ins []int16) []uint32 {
	outs := make([]uint32, len(ins))
	for i, in := range ins {
		outs[i] = uint32(in)
	}
	return outs
}

func (ch *ChallengedHeader) ToProtobuf() (*g.Block_Header_ChallengedHeader, error) {
	headerSig, err := ch.BlockSignature.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &g.Block_Header_ChallengedHeader{
		BaseTarget:          int64(ch.BaseTarget),
		GenerationSignature: ch.GenSignature,
		FeatureVotes:        int16SliceToUint32(ch.Features),
		Timestamp:           int64(ch.Timestamp),
		Generator:           ch.GeneratorPublicKey.Bytes(),
		RewardVote:          ch.RewardVote,
		StateHash:           ch.StateHash.Bytes(),
		HeaderSignature:     headerSig,
	}, nil
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
	TransactionBlockLength uint32              `json:"transactionBlockLength,omitempty"`
	TransactionCount       int                 `json:"transactionCount"`
	GeneratorPublicKey     crypto.PublicKey    `json:"generatorPublicKey"`
	BlockSignature         crypto.Signature    `json:"signature"`
	TransactionsRoot       B58Bytes            `json:"transactionsRoot,omitempty"`
	StateHash              *crypto.Digest      `json:"stateHash,omitempty"`        // is nil before protocol version 1.5
	ChallengedHeader       *ChallengedHeader   `json:"challengedHeader,omitempty"` // is nil before protocol version 1.5
	FinalizationVoting     *FinalizationVoting `json:"finalizationVoting,omitempty"`
	// This field must be generated and set after Block unmarshalling.
	ID BlockID `json:"id"`
}

func (b *BlockHeader) GetStateHash() (crypto.Digest, bool) {
	var (
		sh      crypto.Digest
		present = b.StateHash != nil
	)
	if present {
		sh = *b.StateHash
	}
	return sh, present
}

func (b *BlockHeader) GetFinalizationVoting() (FinalizationVoting, bool) {
	var (
		fv      FinalizationVoting
		present = b.FinalizationVoting != nil
	)
	if present {
		fv = *b.FinalizationVoting
	}
	return fv, present
}

func (b *BlockHeader) GetChallengedHeader() (ChallengedHeader, bool) {
	var (
		ch      ChallengedHeader
		present = b.ChallengedHeader != nil
	)
	if present {
		ch = *b.ChallengedHeader
	}
	return ch, present
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

func (b *BlockHeader) origHeader() (*BlockHeader, bool) {
	ch, present := b.GetChallengedHeader()
	if !present { // fast path
		return b, false
	}
	chSH := ch.StateHash // make a copy of the original value

	orig := *b
	orig.ChallengedHeader = nil // remove the challenged header
	orig.Timestamp = ch.Timestamp
	orig.NxtConsensus = ch.NxtConsensus
	orig.Features = ch.Features
	orig.GeneratorPublicKey = ch.GeneratorPublicKey
	orig.RewardVote = ch.RewardVote
	orig.StateHash = &chSH
	orig.BlockSignature = ch.BlockSignature
	return &orig, true
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
	var challengedHeader *g.Block_Header_ChallengedHeader
	if ch, present := b.GetChallengedHeader(); present {
		var err error
		challengedHeader, err = ch.ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	var stateHash []byte
	if sh, present := b.GetStateHash(); present {
		stateHash = sh.Bytes()
	}
	var finalizationVoting *g.FinalizationVoting
	if fv, present := b.GetFinalizationVoting(); present {
		var err error
		finalizationVoting, err = fv.ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	return &g.Block_Header{
		ChainId:             int32(scheme),
		Reference:           b.Parent.Bytes(),
		BaseTarget:          int64(b.BaseTarget),
		GenerationSignature: b.GenSignature.Bytes(),
		FeatureVotes:        int16SliceToUint32(b.Features),
		Timestamp:           int64(b.Timestamp),
		Version:             int32(b.Version),
		Generator:           b.GeneratorPublicKey.Bytes(),
		RewardVote:          b.RewardVote,
		TransactionsRoot:    b.TransactionsRoot,
		StateHash:           stateHash,
		ChallengedHeader:    challengedHeader,
		FinalizationVoting:  finalizationVoting,
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

func (b *BlockHeader) HeaderToProtobufWithHeight(
	scheme Scheme,
	height uint64,
	vrf []byte,
	rewards Rewards,
) (*pb.BlockWithHeight, error) {
	block, err := b.HeaderToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	rewards = rewards.Sorted() // rewards should be sorted according to the scala node implementation
	rewardShares := make([]*g.RewardShare, len(rewards))
	for i, r := range rewards {
		rewardShares[i] = &g.RewardShare{
			Address: r.Address().Bytes(),
			Reward:  int64(r.Amount()),
		}
	}
	return &pb.BlockWithHeight{
		Block:        block,
		Height:       uint32(height),
		Vrf:          vrf,
		RewardShares: rewardShares,
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
			// Negative vote (-1 means no vote) is encoded as two's complement, as in the Scala node.
			res = binary.BigEndian.AppendUint64(res, uint64(b.RewardVote)) // #nosec G115 -- bit-preserving conversion.
		}
	} else {
		txCount, err := safecast.Convert[byte](b.TransactionCount)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert transactions count to byte")
		}
		res = append(res, txCount) //nolint:makezero // header is written and filled before.
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
	}
	return b.MarshalBinary(scheme)
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
	if b.Version < ProtobufBlockVersion {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if _, err := b.WriteToWithoutSignature(buf, scheme); err != nil {
			return false, err
		}
		return crypto.Verify(b.GeneratorPublicKey, b.BlockSignature, buf.Bytes()), nil
	}

	headerBytes, err := b.MarshalHeaderToProtobufWithoutSignature(scheme)
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal header to protobuf")
	}
	isHeaderValid := crypto.Verify(b.GeneratorPublicKey, b.BlockSignature, headerBytes)
	if !isHeaderValid { // fast path, block is invalid
		return false, nil
	}

	origHeader, changed := b.origHeader()
	if !changed { // fast path, valid block without challenge
		return true, nil
	}
	// block with challenge
	origHeaderBytes, err := origHeader.MarshalHeaderToProtobufWithoutSignature(scheme)
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal challenged header to protobuf")
	}
	isOrigHeaderValid := crypto.Verify(origHeader.GeneratorPublicKey, origHeader.BlockSignature, origHeaderBytes)
	return isOrigHeaderValid, nil
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
	return bytes.Equal(b.TransactionsRoot, rh), nil
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
	return pbBlock.MarshalVTStrict()
}

func (b *Block) Marshaller() Marshaller {
	return BlockMarshaller{
		b: b,
	}
}

func (b *Block) UnmarshalFromProtobuf(data []byte) error {
	var pbBlock = &g.Block{}
	err := pbBlock.UnmarshalVT(data)
	if err != nil {
		return err
	}
	var c ProtobufConverter
	res, err := c.Block(pbBlock)
	if err != nil {
		return err
	}
	*b = res
	return nil
}

func (b *Block) ToProtobuf(scheme Scheme) (*g.Block, error) {
	protoBlock, err := b.HeaderToProtobuf(scheme)
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

func (b *Block) ToProtobufWithHeight(
	currentScheme Scheme,
	height Height,
	vrf []byte,
	rewards Rewards,
) (*pb.BlockWithHeight, error) {
	block, err := b.HeaderToProtobufWithHeight(currentScheme, height, vrf, rewards)
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
		cnt, cntErr := safecast.Convert[uint32](b.TransactionCount)
		if cntErr != nil {
			return 0, fmt.Errorf("failed to convert transactions count to uint32: %w", cntErr)
		}
		s.Uint32(cnt)
	} else {
		txCount, err := safecast.Convert[byte](b.TransactionCount)
		if err != nil {
			return 0, errors.Wrap(err, "failed to convert transactions count to byte")
		}
		s.Byte(txCount)
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
	if pErr := pushTransactionsToMerkleTree(tree, b.Transactions, scheme); pErr != nil {
		return nil, pErr
	}
	return tree.Root().Bytes(), nil
}

// pushTransactionsToMerkleTree pushes the Merkle leaves of the given transactions into the tree.
func pushTransactionsToMerkleTree(tree *crypto.MerkleTree, txs Transactions, scheme Scheme) error {
	for _, tx := range txs {
		mb, err := tx.MerkleBytes(scheme)
		if err != nil {
			return err
		}
		tree.Push(mb)
	}
	return nil
}

func CreateBlock(
	transactions Transactions,
	timestamp Timestamp,
	parentID BlockID,
	publicKey crypto.PublicKey,
	nxtConsensus NxtConsensus,
	version BlockVersion,
	features []int16,
	rewardVote int64,
	scheme Scheme,
	stateHash *crypto.Digest,
	blockFinalizationVoting *FinalizationVoting,
) (*Block, error) {
	return createBlock(transactions, timestamp, parentID, publicKey, nxtConsensus, version, features, rewardVote,
		scheme, stateHash, blockFinalizationVoting, nil, 0)
}

// AppendTransactionsToBlock creates a new block with the transactions of the given block followed by the given
// transactions. All other header fields except the state hash and the finalization voting are taken from the
// given block. The new block is not signed, but its ID is generated (for protobuf blocks the ID does not depend
// on the signature).
//
// For protobuf blocks the transactions root is computed incrementally: if the Merkle tree state of the given
// block is known (the block was created by CreateBlock or AppendTransactionsToBlock recently), only the leaves
// of the new transactions are hashed, so the cost does not grow with the size of the liquid block.
func AppendTransactionsToBlock(
	block *Block,
	transactions Transactions,
	scheme Scheme,
	stateHash *crypto.Digest,
	blockFinalizationVoting *FinalizationVoting,
) (*Block, error) {
	if block == nil {
		return nil, errors.New("no block provided")
	}
	var tree *crypto.MerkleTree
	if block.Version >= ProtobufBlockVersion {
		tree = block.cachedTransactionsTree(scheme)
	}
	treeCount := 0
	if tree != nil {
		treeCount = len(block.Transactions)
	}
	txs := block.Transactions.Join(transactions)
	return createBlock(txs, block.Timestamp, block.Parent, block.GeneratorPublicKey, block.NxtConsensus,
		block.Version, block.Features, block.RewardVote, scheme, stateHash, blockFinalizationVoting, tree, treeCount)
}

// cachedTransactionsTree returns a copy of the cached Merkle tree state of the block transactions or nil.
// The cached state is used only if its root matches the block transactions root.
func (b *Block) cachedTransactionsTree(scheme Scheme) *crypto.MerkleTree {
	tree := liquidMerkleStates.get(b.ID, scheme, len(b.Transactions))
	if tree == nil {
		return nil
	}
	root := tree.Root()
	if !bytes.Equal(root[:], b.TransactionsRoot) {
		return nil
	}
	return tree
}

// createBlock creates a new block. If the tree is not nil, it must hold the Merkle state of the first treeCount
// transactions; only the remaining transactions are pushed into it to calculate the transactions root.
func createBlock(
	transactions Transactions,
	timestamp Timestamp,
	parentID BlockID,
	publicKey crypto.PublicKey,
	nxtConsensus NxtConsensus,
	version BlockVersion,
	features []int16,
	rewardVote int64,
	scheme Scheme,
	stateHash *crypto.Digest,
	blockFinalizationVoting *FinalizationVoting,
	tree *crypto.MerkleTree,
	treeCount int,
) (*Block, error) {
	consensusLength := nxtConsensus.BinarySize()
	b := &Block{
		Version:              version,
		Timestamp:            timestamp,
		Parent:               parentID,
		FeaturesCount:        len(features),
		Features:             features,
		RewardVote:           rewardVote,
		ConsensusBlockLength: uint32(consensusLength), // #nosec G115: consensus binary size is small.
		NxtConsensus:         nxtConsensus,
		TransactionCount:     transactions.Count(),
		GeneratorPublicKey:   publicKey,
		StateHash:            stateHash,
		FinalizationVoting:   blockFinalizationVoting,
		Transactions:         transactions,
	}
	switch {
	case version < NgBlockVersion:
		b.TransactionBlockLength = uint32(transactions.BinarySize() + 1) // add extra sizeof(byte) == 1 bytes for version
	case version <= RewardBlockVersion:
		b.TransactionBlockLength = uint32(transactions.BinarySize() + 4) // add extra sizeof(int) == 4 bytes for version
	case version >= ProtobufBlockVersion:
		if tree == nil {
			t, err := crypto.NewMerkleTree()
			if err != nil {
				return nil, err
			}
			tree, treeCount = t, 0
		}
		if err := pushTransactionsToMerkleTree(tree, transactions[treeCount:], scheme); err != nil {
			return nil, err
		}
		b.TransactionsRoot = tree.Root().Bytes()
	}
	if err := b.GenerateBlockID(scheme); err != nil {
		return nil, errors.Wrap(err, "failed to generate block ID")
	}
	if version >= ProtobufBlockVersion {
		liquidMerkleStates.put(b.ID, scheme, len(transactions), tree)
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
	}
	return a.b.MarshalBinary(scheme)
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
	return a.appendBinary(make([]byte, 0, a.BinarySize()), scheme)
}

// txBinaryAppender is implemented by the transactions that can write their binary form (the same bytes as
// MarshalBinary) straight into a caller's buffer.
type txBinaryAppender interface {
	appendTxBinary(dst []byte, scheme Scheme) ([]byte, error)
}

// signedTxNoCopyConverter is implemented by the Waves transactions that can build their signed protobuf form
// without copying the proofs. The result is only for marshalling, it must not be retained.
type signedTxNoCopyConverter interface {
	toProtobufSignedNoCopy(scheme Scheme) (*g.SignedTransaction, error)
}

// appendBinary appends every transaction as a 4-byte big-endian length followed by its MarshalBinary bytes.
// The result is byte-identical to WriteToBinary; when dst has BinarySize() spare capacity it does not grow.
func (a Transactions) appendBinary(dst []byte, scheme Scheme) ([]byte, error) {
	for _, tx := range a {
		if ap, ok := tx.(txBinaryAppender); ok {
			pos := len(dst)
			dst = append(dst, 0, 0, 0, 0)
			var err error
			dst, err = ap.appendTxBinary(dst, scheme)
			if err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint32(dst[pos:], uint32(len(dst)-pos-uint32Size)) // #nosec G115 // As before.
			continue
		}
		bts, err := tx.MarshalBinary(scheme)
		if err != nil {
			return nil, err
		}
		dst = binary.BigEndian.AppendUint32(dst, uint32(len(bts))) // #nosec G115 // As before.
		dst = append(dst, bts...)
	}
	return dst, nil
}

// signedProtobufs returns the signed protobuf messages of the transactions and the total size of their
// length-prefixed encoding. Transactions without a non-copying converter are returned as raw bytes in raw.
// The messages alias the transactions' proofs and must only be used for marshalling.
func (a Transactions) signedProtobufs(scheme Scheme) ([]*g.SignedTransaction, [][]byte, int, error) {
	pbs := make([]*g.SignedTransaction, len(a))
	var raw [][]byte
	size := 0
	for i, tx := range a {
		if c, ok := tx.(signedTxNoCopyConverter); ok {
			pb, err := c.toProtobufSignedNoCopy(scheme)
			if err != nil {
				return nil, nil, 0, err
			}
			pbs[i] = pb
			size += uint32Size + pb.SizeVT()
			continue
		}
		bts, err := tx.MarshalSignedToProtobuf(scheme)
		if err != nil {
			return nil, nil, 0, err
		}
		if raw == nil {
			raw = make([][]byte, len(a))
		}
		raw[i] = bts
		size += uint32Size + len(bts)
	}
	return pbs, raw, size, nil
}

// putSignedProtobufs writes the length-prefixed messages prepared by signedProtobufs into buf, which must be
// exactly the size returned by signedProtobufs. The bytes equal the protobuf branch of Transactions.WriteTo.
func putSignedProtobufs(buf []byte, pbs []*g.SignedTransaction, raw [][]byte) error {
	pos := 0
	for i, pb := range pbs {
		if pb == nil {
			bts := raw[i]
			binary.BigEndian.PutUint32(buf[pos:], uint32(len(bts))) // #nosec G115 // As before.
			pos += uint32Size
			pos += copy(buf[pos:], bts)
			continue
		}
		l := pb.SizeVT()
		binary.BigEndian.PutUint32(buf[pos:], uint32(l)) // #nosec G115 // As before.
		pos += uint32Size
		n, err := pb.MarshalToSizedBufferVTStrict(buf[pos : pos+l])
		if err != nil {
			return err
		}
		if n != l {
			return errors.Errorf("unexpected protobuf transaction size %d, expected %d", n, l)
		}
		pos += l
	}
	if pos != len(buf) {
		return errors.Errorf("unexpected protobuf transactions size %d, expected %d", pos, len(buf))
	}
	return nil
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
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	b, err := a.appendBinary(slices.Grow(buf.B[:0], a.BinarySize()), scheme)
	if err != nil {
		return 0, err
	}
	buf.B = b
	n, err := w.Write(b)
	return int64(n), err
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
	transactions, err := UnmarshalTransactionsListJSON(data)
	if err != nil {
		return err
	}
	*a = transactions
	return nil
}

// UnmarshalTransactionsListJSON decodes a JSON array of transactions of mixed types in a single pass.
// Each array element is handed to txListElementJSON as a sub-slice of data (without copying), probed for its type and
// version and then decoded exactly once into the guessed concrete transaction type.
// A JSON null input yields a nil slice; a null element is reported as an error.
func UnmarshalTransactionsListJSON(data []byte) ([]Transaction, error) {
	var elements []txListElementJSON
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, errors.Wrap(err, "TransactionType unmarshal")
	}
	if elements == nil {
		return nil, nil
	}
	transactions := make([]Transaction, len(elements))
	for i := range elements {
		transactions[i] = elements[i].tx
	}
	return transactions, nil
}

// txListElementJSON decodes one element of a transactions JSON array. The input slice is not retained.
type txListElementJSON struct {
	tx Transaction
}

func (e *txListElementJSON) UnmarshalJSON(raw []byte) error {
	if string(raw) == jsonNull {
		return errors.New("null transaction")
	}
	var tv TransactionTypeVersion
	if err := json.Unmarshal(raw, &tv); err != nil {
		return err
	}
	tx, err := GuessTransactionType(&tv)
	if err != nil {
		return err
	}
	// The element was already validated by the array decode, so a custom unmarshaler is called directly.
	if u, ok := tx.(json.Unmarshaler); ok {
		err = u.UnmarshalJSON(raw)
	} else {
		err = json.Unmarshal(raw, tx)
	}
	if err != nil {
		return err
	}
	e.tx = tx
	return nil
}
