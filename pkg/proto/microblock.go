package proto

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/deserializer"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	MicroBlockInvSizeSig  = crypto.PublicKeySize + crypto.SignatureSize*3
	MicroBlockInvSizeHash = crypto.PublicKeySize + crypto.DigestSize*2 + crypto.SignatureSize
)

type MicroBlock struct {
	VersionField BlockVersion
	// Reference for previous block.
	Reference BlockID
	// Block signature.
	TotalResBlockSigField crypto.Signature
	TotalBlockID          BlockID
	TransactionCount      uint32
	Transactions          Transactions
	SenderPK              crypto.PublicKey
	Signature             crypto.Signature
	StateHash             *crypto.Digest // is nil before protocol version 1.5
	PartialFinalization   *FinalizationVoting
}

type MicroblockTotalSig = crypto.Signature

func (a *MicroBlock) GetStateHash() (crypto.Digest, bool) {
	var (
		sh      crypto.Digest
		present = a.StateHash != nil
	)
	if present {
		sh = *a.StateHash
	}
	return sh, present
}

func (a *MicroBlock) GetPartialFinalization() (FinalizationVoting, bool) {
	var (
		fin     FinalizationVoting
		present = a.PartialFinalization != nil
	)
	if present {
		fin = *a.PartialFinalization
	}
	return fin, present
}

func (a *MicroBlock) UnmarshalFromProtobuf(b []byte) error {
	var pbMicroBlock = &g.SignedMicroBlock{}
	if err := pbMicroBlock.UnmarshalVT(b); err != nil {
		return errors.Wrap(err, "SignedMicroBlock: failed to unmarshal")
	}
	var c ProtobufConverter
	res, err := c.MicroBlock(pbMicroBlock)
	if err != nil {
		return errors.Wrap(err, "ProtobufConverter")
	}
	*a = res
	return nil
}

func (a *MicroBlock) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	pbMicro, err := a.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	return MarshalToProtobufDeterministic(pbMicro)
}

func (a *MicroBlock) ToProtobuf(scheme Scheme) (*g.SignedMicroBlock, error) {
	sig, err := a.Signature.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ref := a.Reference.Bytes()
	total, err := a.TotalResBlockSigField.MarshalBinary()
	if err != nil {
		return nil, err
	}
	txs, err := a.Transactions.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	var stateHash []byte
	if sh, present := a.GetStateHash(); present {
		stateHash = sh.Bytes()
	}
	var finalizationProtobuf *g.FinalizationVoting
	if fin, present := a.GetPartialFinalization(); present {
		var cnvrtErr error
		finalizationProtobuf, cnvrtErr = fin.ToProtobuf()
		if cnvrtErr != nil {
			return nil, cnvrtErr
		}
	}
	return &g.SignedMicroBlock{
		MicroBlock: &g.MicroBlock{
			Version:               int32(a.VersionField),
			Reference:             ref,
			UpdatedBlockSignature: total,
			SenderPublicKey:       a.SenderPK.Bytes(),
			Transactions:          txs,
			StateHash:             stateHash,
			FinalizationVoting:    finalizationProtobuf,
		},
		Signature:    sig,
		TotalBlockId: a.TotalBlockID.Bytes(),
	}, nil
}

func (a *MicroBlock) UnmarshalBinary(b []byte, scheme Scheme) error {
	var err error
	d := deserializer.NewDeserializer(b)

	v, err := d.Byte()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock version")
	}
	a.VersionField = BlockVersion(v)

	proto := a.VersionField >= ProtobufBlockVersion
	if proto {
		ref, err := d.Digest()
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal reference")
		}
		a.Reference = NewBlockIDFromDigest(ref)
	} else {
		sig, err := d.Signature()
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal reference")
		}
		a.Reference = NewBlockIDFromSignature(sig)
	}

	a.TotalResBlockSigField, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock totalResBlockSigField")
	}

	tBytesLength, err := d.Uint32()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock transaction bytes len")
	}

	a.TransactionCount, err = d.Uint32()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock transaction count")
	}

	bts, err := d.Bytes(uint(tBytesLength) - 4)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock transaction bytes")
	}
	if proto {
		txs := new(Transactions)
		if err := txs.UnmarshalFromProtobuf(bts); err != nil {
			return errors.Wrap(err, "failed to unmarshal transactions from protobuf")
		}
		a.Transactions = *txs
	} else {
		a.Transactions, err = NewTransactionsFromBytes(bts, int(a.TransactionCount), scheme)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal transactions")
		}
	}

	a.SenderPK, err = d.PublicKey()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock public key")
	}

	a.Signature, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock signature")
	}

	return nil
}

func (a *MicroBlock) VerifySignature(scheme Scheme) (bool, error) {
	b, err := a.marshalWithoutSignature(scheme, 0)
	if err != nil {
		return false, err
	}
	return crypto.Verify(a.SenderPK, a.Signature, b), nil
}

func (a *MicroBlock) Sign(scheme Scheme, secret crypto.SecretKey) error {
	b, err := a.marshalWithoutSignature(scheme, 0)
	if err != nil {
		return err
	}
	sig, err := crypto.Sign(secret, b)
	if err != nil {
		return err
	}
	a.Signature = sig
	return nil
}

func (a *MicroBlock) WriteTo(scheme Scheme, w io.Writer) (int64, error) {
	b, err := a.MarshalBinary(scheme)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

func (a *MicroBlock) WriteWithoutSignature(scheme Scheme, w io.Writer) (int64, error) {
	b, err := a.marshalWithoutSignature(scheme, 0)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// microBlockTxsHeaderSize is the size of the transactions length and count fields.
const microBlockTxsHeaderSize = 2 * uint32Size

// marshalWithoutSignature encodes the microblock without its signature into one buffer with extra bytes of spare
// capacity. Transactions are written in place: binary ones with their append helpers, protobuf ones with
// MarshalToSizedBufferVTStrict. The bytes are the same as the former serializer-based WriteWithoutSignature.
func (a *MicroBlock) marshalWithoutSignature(scheme Scheme, extra int) ([]byte, error) {
	ref := a.Reference.Bytes()
	var stateHash []byte
	if sh, present := a.GetStateHash(); present {
		stateHash = sh.Bytes()
	}
	isProto := a.VersionField >= ProtobufBlockVersion
	var (
		finalization     *g.FinalizationVoting
		finalizationSize int
	)
	if isProto && a.PartialFinalization != nil { // Write finalization only for protobuf micro blocks.
		fp, err := a.PartialFinalization.ToProtobuf()
		if err != nil {
			return nil, err
		}
		finalization = fp
		finalizationSize = fp.SizeVT()
	}
	head := 1 + len(ref) + crypto.SignatureSize + microBlockTxsHeaderSize
	tail := crypto.PublicKeySize + len(stateHash) + finalizationSize + extra
	var buf []byte
	if isProto {
		pbs, raw, txsSize, err := a.Transactions.signedProtobufs(scheme)
		if err != nil {
			return nil, err
		}
		buf = make([]byte, head+txsSize, head+txsSize+tail)
		if pErr := putSignedProtobufs(buf[head:], pbs, raw); pErr != nil {
			return nil, pErr
		}
	} else {
		var err error
		buf = make([]byte, head, head+a.Transactions.BinarySize()+tail)
		buf, err = a.Transactions.appendBinary(buf, scheme)
		if err != nil {
			return nil, err
		}
	}
	buf[0] = byte(a.VersionField)
	pos := 1
	pos += copy(buf[pos:], ref)
	pos += copy(buf[pos:], a.TotalResBlockSigField[:])
	txsLen := len(buf) - head + uint32Size                // The length includes the transactions count field.
	binary.BigEndian.PutUint32(buf[pos:], uint32(txsLen)) // #nosec G115 // As before.
	binary.BigEndian.PutUint32(buf[pos+uint32Size:], a.TransactionCount)
	buf = append(buf, a.SenderPK[:]...)
	buf = append(buf, stateHash...)
	if finalization != nil {
		l := len(buf)
		buf = buf[:l+finalizationSize]
		n, err := finalization.MarshalToSizedBufferVTStrict(buf[l:])
		if err != nil {
			return nil, err
		}
		if n != finalizationSize {
			return nil, errors.Errorf("unexpected finalization size %d, expected %d", n, finalizationSize)
		}
	}
	return buf, nil
}

func (a *MicroBlock) MarshalBinary(scheme Scheme) ([]byte, error) {
	b, err := a.marshalWithoutSignature(scheme, crypto.SignatureSize)
	if err != nil {
		return nil, err
	}
	return append(b, a.Signature[:]...), nil
}

func (a *MicroBlock) String() string {
	pf := "nil"
	if a.PartialFinalization != nil {
		pf = a.PartialFinalization.String()
	}
	return fmt.Sprintf("MicroBlock{Version: %d, Reference: %s, TotalResBlockSigField: %s, TotalBlockID: %s, "+
		"TransactionCount: %d, SenderPK: %s, Signature: %s, StateHash: %s, PartialFinalization: %s}",
		a.VersionField, a.Reference.String(), a.TotalResBlockSigField.String(), a.TotalBlockID.String(),
		a.TransactionCount, a.SenderPK.String(), a.Signature.String(), a.StateHash.String(), pf)
}

// MicroBlockMessage represents a MicroBlock message.
type MicroBlockMessage struct {
	Body BytesPayload
}

func (m *MicroBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroblock, "MicroBlockMessage", &m.Body)
}

func (m *MicroBlockMessage) WriteTo(w io.Writer) (int64, error) {
	return writeBytesMessageNamed(w, ContentIDMicroblock, "", m.Body)
}

func (m *MicroBlockMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDMicroblock {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]

	if len(data) < crypto.SignatureSize*2+1 {
		return errors.New("invalid micro block size")
	}
	b := make([]byte, len(data[:h.payloadLength]))
	copy(b, data)

	m.Body = b
	return nil
}

func (m *MicroBlockMessage) MarshalBinary() ([]byte, error) {
	return marshalMessage(ContentIDMicroblock, m.Body)
}

func (m *MicroBlockMessage) IsMessage() {}

func (m *MicroBlockMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Body = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type MicroBlockInvMessage struct {
	Body BytesPayload
}

func (m *MicroBlockInvMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDInvMicroblock, "MicroBlockInvMessage", &m.Body)
}

func (m *MicroBlockInvMessage) WriteTo(w io.Writer) (int64, error) {
	h, err := NewHeader(ContentIDInvMicroblock, m.Body)
	if err != nil {
		return 0, err
	}
	n1, err := h.WriteTo(w)
	if err != nil {
		return 0, err
	}
	n2, err := w.Write(m.Body)
	if err != nil {
		return 0, err
	}
	return n1 + int64(n2), nil
}

func (m *MicroBlockInvMessage) MarshalBinary() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := m.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.B)
	return out, nil
}

func (m *MicroBlockInvMessage) IsMessage() {}

func (m *MicroBlockInvMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Body = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// MicroBlockRequestMessage total block signature or ID.
type MicroBlockRequestMessage struct {
	TotalBlockSig BlockID
}

func (m *MicroBlockRequestMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroblockRequest, "MicroBlockRequestMessage", &m.TotalBlockSig)
}

func (m *MicroBlockRequestMessage) WriteTo(w io.Writer) (int64, error) {
	h, err := NewHeader(ContentIDMicroblockRequest, m.TotalBlockSig.Bytes())
	if err != nil {
		return 0, err
	}
	n2, err := h.WriteTo(w)
	if err != nil {
		return 0, err
	}
	n3, err := w.Write(m.TotalBlockSig.Bytes())
	if err != nil {
		return 0, err
	}
	return n2 + int64(n3), nil
}

func (m *MicroBlockRequestMessage) MarshalBinary() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := m.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.B)
	return out, nil
}

func (m *MicroBlockRequestMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDMicroblockRequest {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	id, err := NewBlockIDFromBytes(data[:h.payloadLength])
	if err != nil {
		return fmt.Errorf("failed to unmarshal MicroBlockRequestMessage: %w", err)
	}
	m.TotalBlockSig = id
	return nil
}

func (m *MicroBlockRequestMessage) IsMessage() {}

func (m *MicroBlockRequestMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockID); ok {
		m.TotalBlockSig = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type MicroBlockInv struct {
	PublicKey    crypto.PublicKey
	TotalBlockID BlockID
	Reference    BlockID
	Signature    crypto.Signature
}

func (a *MicroBlockInv) MarshalBinary() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := a.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.B)
	return out, nil
}

func (a *MicroBlockInv) UnmarshalBinary(data []byte) error {
	sigId := len(data) == MicroBlockInvSizeSig
	hashId := len(data) == MicroBlockInvSizeHash
	if !sigId && !hashId {
		return errors.Errorf("MicroBlockInv UnmarshalBinary: invalid data size, expected to be %d or %d, found %d", MicroBlockInvSizeSig, MicroBlockInvSizeHash, len(data))
	}
	var err error
	d := deserializer.NewDeserializer(data)
	a.PublicKey, err = d.PublicKey()
	if err != nil {
		return err
	}
	if hashId {
		totalId, err := d.Digest()
		if err != nil {
			return err
		}
		a.TotalBlockID = NewBlockIDFromDigest(totalId)
		ref, err := d.Digest()
		if err != nil {
			return err
		}
		a.Reference = NewBlockIDFromDigest(ref)
	} else {
		sig, err := d.Signature()
		if err != nil {
			return err
		}
		a.TotalBlockID = NewBlockIDFromSignature(sig)
		ref, err := d.Signature()
		if err != nil {
			return err
		}
		a.Reference = NewBlockIDFromSignature(ref)
	}
	a.Signature, err = d.Signature()
	return err
}

func (a *MicroBlockInv) WriteTo(w io.Writer) (int64, error) {
	s := serializer.NewNonFallable(w)
	s.Bytes(a.PublicKey.Bytes())
	s.Bytes(a.TotalBlockID.Bytes())
	s.Bytes(a.Reference.Bytes())
	s.Bytes(a.Signature.Bytes())
	return s.N(), nil
}

func (a *MicroBlockInv) Sign(key crypto.SecretKey, schema Scheme) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	err := a.bodyBytes(buf, schema)
	if err != nil {
		return err
	}
	a.Signature, err = crypto.Sign(key, buf.Bytes())
	return err
}

func (a *MicroBlockInv) bodyBytes(w io.Writer, schema Scheme) error {
	addr, err := NewAddressFromPublicKey(schema, a.PublicKey)
	if err != nil {
		return err
	}
	s := serializer.NewNonFallable(w)
	s.Bytes(addr.Bytes())
	s.Bytes(a.TotalBlockID.Bytes())
	s.Bytes(a.Reference.Bytes())
	return nil
}

func (a *MicroBlockInv) Verify(schema Scheme) (bool, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	err := a.bodyBytes(buf, schema)
	if err != nil {
		return false, err
	}
	return crypto.Verify(a.PublicKey, a.Signature, buf.Bytes()), nil
}

func NewUnsignedMicroblockInv(PublicKey crypto.PublicKey, TotalBlockID BlockID, Reference BlockID) *MicroBlockInv {
	return &MicroBlockInv{
		PublicKey:    PublicKey,
		TotalBlockID: TotalBlockID,
		Reference:    Reference,
	}
}

func (m *MicroBlockInvMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDInvMicroblock {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	body := make([]byte, h.payloadLength)
	copy(body, data[:h.payloadLength])
	m.Body = body
	return nil
}

// PBMicroBlockMessage represents a Protobuf MicroBlock message.
type PBMicroBlockMessage struct {
	MicroBlockBytes BytesPayload
}

func (m *PBMicroBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroblock, "PBMicroBlockMessage", &m.MicroBlockBytes)
}

func (m *PBMicroBlockMessage) WriteTo(w io.Writer) (int64, error) {
	return writeBytesMessageNamed(w, ContentIDPBMicroBlock, "", m.MicroBlockBytes)
}

func (m *PBMicroBlockMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDPBMicroBlock {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	if h.payloadLength < crypto.DigestSize {
		return errors.New("PBMicroBlockMessage UnmarshalBinary: invalid data size")
	}
	data = data[17:]

	if common.SafeIntToUint32(len(data)) < h.payloadLength {
		return errors.New("invalid data size")
	}
	mbBytes := data[:h.payloadLength]
	m.MicroBlockBytes = make([]byte, len(mbBytes))
	copy(m.MicroBlockBytes, mbBytes)
	return nil
}

func (m *PBMicroBlockMessage) MarshalBinary() ([]byte, error) {
	return marshalMessage(ContentIDPBMicroBlock, m.MicroBlockBytes)
}

func (m *PBMicroBlockMessage) IsMessage() {}

func (m *PBMicroBlockMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.MicroBlockBytes = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// AppendMicroBlock combines a block and a microblock into a new block.
// It checks that the microblock reference matches the block ID, combines transactions and finalization voting.
// NOTE that this function does nothing with new block signature. The ID of the new block is generated, for
// protobuf blocks it is final because it does not depend on the signature, for legacy blocks the ID is the
// signature and must be generated again after setting it.
func AppendMicroBlock(block *Block, microBlock *MicroBlock, scheme Scheme) (*Block, error) {
	if block == nil {
		return nil, errors.New("no block provided")
	}
	if microBlock == nil {
		return nil, errors.New("no microblock provided")
	}
	if block.ID != microBlock.Reference {
		return nil, fmt.Errorf("microblock reference '%s' does not match block ID '%s'",
			microBlock.Reference.String(), block.ID.String())
	}

	fv := CombineFinalizationVoting(block.FinalizationVoting, microBlock.PartialFinalization)
	newBlock, err := AppendTransactionsToBlock(block, microBlock.Transactions, scheme, microBlock.StateHash, fv)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block: %w", err)
	}
	return newBlock, nil
}
