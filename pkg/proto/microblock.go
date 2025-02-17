package proto

import (
	"bytes"
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
	VersionField byte
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
	return &g.SignedMicroBlock{
		MicroBlock: &g.MicroBlock{
			Version:               int32(a.VersionField),
			Reference:             ref,
			UpdatedBlockSignature: total,
			SenderPublicKey:       a.SenderPK.Bytes(),
			Transactions:          txs,
			StateHash:             stateHash,
		},
		Signature:    sig,
		TotalBlockId: a.TotalBlockID.Bytes(),
	}, nil
}

func (a *MicroBlock) UnmarshalBinary(b []byte, scheme Scheme) error {
	var err error
	d := deserializer.NewDeserializer(b)

	a.VersionField, err = d.Byte()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock version")
	}

	proto := a.VersionField >= byte(ProtobufBlockVersion)
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
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.WriteWithoutSignature(scheme, buf)
	if err != nil {
		return false, err
	}

	return crypto.Verify(a.SenderPK, a.Signature, buf.Bytes()), nil
}

func (a *MicroBlock) Sign(scheme Scheme, secret crypto.SecretKey) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.WriteWithoutSignature(scheme, buf)
	if err != nil {
		return err
	}
	sig, err := crypto.Sign(secret, buf.Bytes())
	if err != nil {
		return err
	}
	a.Signature = sig
	return nil
}

func (a *MicroBlock) WriteTo(scheme Scheme, w io.Writer) (int64, error) {
	n, _ := a.WriteWithoutSignature(scheme, w)
	n2, _ := w.Write(a.Signature.Bytes())
	return n + int64(n2), nil
}

func (a *MicroBlock) WriteWithoutSignature(scheme Scheme, w io.Writer) (int64, error) {
	s := serializer.NewNonFallable(w)
	s.Byte(a.VersionField)
	s.Bytes(a.Reference.Bytes())
	s.Bytes(a.TotalResBlockSigField.Bytes())
	// Serialize transactions in separate buffer to get the size
	txsBuf := new(bytes.Buffer)
	txsSerializer := serializer.NewNonFallable(txsBuf)
	proto := a.VersionField >= byte(ProtobufBlockVersion)
	if _, err := a.Transactions.WriteTo(proto, scheme, txsSerializer); err != nil {
		return 0, err
	}
	// Write transactions bytes size and its count
	s.Uint32(uint32(txsBuf.Len() + 4))
	s.Uint32(a.TransactionCount)
	// Write transactions bytes
	s.Bytes(txsBuf.Bytes())
	s.Bytes(a.SenderPK.Bytes())
	// Write state hash if it's present
	var stateHash []byte
	if sh, present := a.GetStateHash(); present {
		stateHash = sh.Bytes()
	}
	s.Bytes(stateHash)
	return s.N(), nil
}

func (a *MicroBlock) MarshalBinary(scheme Scheme) ([]byte, error) {
	buf := &bytes.Buffer{}
	_, err := a.WriteTo(scheme, buf)
	return buf.Bytes(), err
}

// MicroBlockMessage represents a MicroBlock message.
type MicroBlockMessage struct {
	Body BytesPayload
}

func (m *MicroBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroblock, "MicroBlockMessage", &m.Body)
}

func (m *MicroBlockMessage) WriteTo(w io.Writer) (int64, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := buf.Write(m.Body)
	if err != nil {
		return 0, err
	}

	h, err := NewHeader(ContentIDMicroblock, buf.Bytes())
	if err != nil {
		return 0, err
	}

	n1, err := h.WriteTo(w)
	if err != nil {
		return n1, err
	}

	n2, err := buf.WriteTo(w)
	return n1 + n2, err
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
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := m.MicroBlockBytes.WriteTo(buf)
	if err != nil {
		return 0, err
	}

	h, err := NewHeader(ContentIDPBMicroBlock, buf.Bytes())
	if err != nil {
		return 0, err
	}

	n1, err := h.WriteTo(w)
	if err != nil {
		return n1, err
	}

	n2, err := buf.WriteTo(w)
	return n1 + n2, err
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

func (m *PBMicroBlockMessage) IsMessage() {}

func (m *PBMicroBlockMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.MicroBlockBytes = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}
