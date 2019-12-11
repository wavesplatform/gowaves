package proto

import (
	"io"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/deserializer"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

type MicroBlock struct {
	VersionField byte
	// reference for previous keyblock or microblock
	PrevResBlockSigField  crypto.Signature
	TotalResBlockSigField crypto.Signature
	TransactionCount      uint32
	Transactions          *TransactionsRepresentation
	SenderPK              crypto.PublicKey
	Signature             crypto.Signature
}

// func is not using right now
/*
func CreateMicroBlock(PrevResBlockSigField crypto.Signature, TotalResBlockSigField crypto.Signature, tr *TransactionsRepresentation, SenderPK crypto.PublicKey) *MicroBlock {
	return &MicroBlock{
		VersionField:          3,
		PrevResBlockSigField:  PrevResBlockSigField,
		TotalResBlockSigField: TotalResBlockSigField,
		Transactions:          tr,
		TransactionCount:      uint32(tr.Count()),
		SenderPK:              SenderPK,
	}
}
*/

type MicroblockTotalSig = crypto.Signature

func (a *MicroBlock) UnmarshalBinary(b []byte) error {
	var err error
	d := deserializer.NewDeserializer(b)

	a.VersionField, err = d.Byte()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock version")
	}

	a.PrevResBlockSigField, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock prevResBlockSigField")
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
	a.Transactions = NewReprFromBytes(bts, int(a.TransactionCount))

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

func (a *MicroBlock) VerifySignature() (bool, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.WriteWithoutSignature(buf)
	if err != nil {
		return false, err
	}

	return crypto.Verify(a.SenderPK, a.Signature, buf.Bytes()), nil
}

func (a *MicroBlock) Sign(secret crypto.SecretKey) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := a.WriteWithoutSignature(buf)
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

func (a *MicroBlock) WriteTo(w io.Writer) (int64, error) {
	n, _ := a.WriteWithoutSignature(w)
	n2, _ := w.Write(a.Signature[:])
	return n + int64(n2), nil
}

func (a *MicroBlock) WriteWithoutSignature(w io.Writer) (int64, error) {
	s := serializer.NewNonFallable(w)
	s.Byte(a.VersionField)
	s.Bytes(a.PrevResBlockSigField[:])
	s.Bytes(a.TotalResBlockSigField[:])

	bts, err := a.Transactions.Bytes()
	if err != nil {
		return 0, err
	}
	s.Uint32(uint32(len(bts)) + 4)

	s.Uint32(a.TransactionCount)
	s.Bytes(bts)
	s.Bytes(a.SenderPK[:])
	return s.N(), nil
}

// MicroBlockMessage represents a MicroBlock message
type MicroBlockMessage struct {
	Body io.WriterTo
}

func (*MicroBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	panic("implement me")
}

func (a *MicroBlockMessage) WriteTo(w io.Writer) (int64, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	n, err := a.Body.WriteTo(buf)
	if err != nil {
		return n, err
	}

	h, err := MakeHeader(ContentIDMicroblock, buf.Bytes())
	if err != nil {
		return 0, err
	}

	n1, err := h.WriteTo(w)
	if err != nil {
		return n1, err
	}

	n2, err := buf.WriteTo(w)
	if err != nil {
		return n1 + n2, err
	}
	return n1 + n2, nil
}

func (a *MicroBlockMessage) UnmarshalBinary(data []byte) error {
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
	b := make([]byte, len(data[:h.PayloadLength]))
	copy(b, data)
	a.Body = Bytes(b)
	return nil
}

func (a *MicroBlockMessage) MarshalBinary() ([]byte, error) {
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

type MicroBlockInvMessage struct {
	Body []byte
}

func (a *MicroBlockInvMessage) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockInvMessage) WriteTo(w io.Writer) (n int64, err error) {
	var h Header
	h.Length = MaxHeaderLength + uint32(len(a.Body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDInvMicroblock
	h.PayloadLength = uint32(len(a.Body))
	dig, err := crypto.FastHash(a.Body)
	if err != nil {
		return 0, err
	}
	copy(h.PayloadCsum[:], dig[:4])
	n1, err := h.WriteTo(w)
	if err != nil {
		return 0, err
	}
	n2, err := w.Write(a.Body)
	if err != nil {
		return 0, err
	}
	return n1 + int64(n2), nil
}

func (a *MicroBlockInvMessage) MarshalBinary() ([]byte, error) {
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

type MicroBlockRequestMessage struct {
	Body io.WriterTo
}

func (a *MicroBlockRequestMessage) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockRequestMessage) WriteTo(w io.Writer) (int64, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	n, err := a.Body.WriteTo(buf)
	if err != nil {
		return n, err
	}

	var h Header
	h.Length = MaxHeaderLength + uint32(buf.Len()) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDMicroblockRequest
	h.PayloadLength = uint32(buf.Len())
	dig, err := crypto.FastHash(buf.B)
	if err != nil {
		return 0, err
	}
	copy(h.PayloadCsum[:], dig[:4])
	n2, err := h.WriteTo(w)
	if err != nil {
		return 0, err
	}

	n3, err := buf.WriteTo(w)
	if err != nil {
		return 0, err
	}
	return n2 + n3, nil
}

func (a *MicroBlockRequestMessage) MarshalBinary() ([]byte, error) {
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

func (a *MicroBlockRequestMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDMicroblockRequest {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	body := make([]byte, h.PayloadLength)
	copy(body, data)
	a.Body = Bytes(body)
	return nil
}

type MicroBlockRequest struct {
	TotalBlockSig crypto.Signature
}

func (a *MicroBlockRequest) ReadFrom(r io.Reader) (int64, error) {
	body := make([]byte, crypto.SignatureSize)
	n, err := r.Read(body)
	if err != nil {
		return int64(n), err
	}
	sig, err := crypto.NewSignatureFromBytes(body)
	if err != nil {
		return int64(n), err
	}
	a.TotalBlockSig = sig
	return int64(n), nil
}

func (a *MicroBlockRequest) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(a.TotalBlockSig[:])
	return int64(n), err
}

func (a *MicroBlockRequest) UnmarshalBinary(data []byte) error {
	sig, err := crypto.NewSignatureFromBytes(data)
	if err != nil {
		return err
	}
	a.TotalBlockSig = sig
	return nil
}

func (a *MicroBlockRequest) MarshalBinary() ([]byte, error) {
	return a.TotalBlockSig[:], nil
}

type MicroBlockInv struct {
	PublicKey     crypto.PublicKey
	TotalBlockSig crypto.Signature
	PrevBlockSig  crypto.Signature
	Signature     crypto.Signature
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
	var err error
	d := deserializer.NewDeserializer(data)
	a.PublicKey, err = d.PublicKey()
	if err != nil {
		return err
	}

	a.TotalBlockSig, err = d.Signature()
	if err != nil {
		return err
	}

	a.PrevBlockSig, err = d.Signature()
	if err != nil {
		return err
	}

	a.Signature, err = d.Signature()
	if err != nil {
		return err
	}
	return nil
}

func (a *MicroBlockInv) WriteTo(w io.Writer) (int64, error) {
	s := serializer.NewNonFallable(w)
	s.Bytes(a.PublicKey.Bytes())
	s.Bytes(a.TotalBlockSig.Bytes())
	s.Bytes(a.PrevBlockSig.Bytes())
	s.Bytes(a.Signature.Bytes())
	return s.N(), nil
}

func (a *MicroBlockInv) Sign(key crypto.SecretKey, schema Scheme) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	err := a.signableBytes(buf, schema)
	if err != nil {
		return err
	}
	a.Signature, err = crypto.Sign(key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (a *MicroBlockInv) signableBytes(w io.Writer, schema Scheme) error {
	addr, err := NewAddressFromPublicKey(schema, a.PublicKey)
	if err != nil {
		return err
	}
	s := serializer.NewNonFallable(w)
	s.Bytes(addr.Bytes())
	s.Bytes(a.TotalBlockSig.Bytes())
	s.Bytes(a.PrevBlockSig.Bytes())
	return nil
}

func (a *MicroBlockInv) Verify(schema Scheme) (bool, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	err := a.signableBytes(buf, schema)
	if err != nil {
		return false, err
	}
	return crypto.Verify(a.PublicKey, a.Signature, buf.Bytes()), nil
}

func NewUnsignedMicroblockInv(
	PublicKey crypto.PublicKey,
	TotalBlockSig crypto.Signature,
	PrevBlockSig crypto.Signature) *MicroBlockInv {

	return &MicroBlockInv{
		PublicKey:     PublicKey,
		TotalBlockSig: TotalBlockSig,
		PrevBlockSig:  PrevBlockSig,
	}
}

func (a *MicroBlockInvMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDInvMicroblock {
		return errors.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	body := make([]byte, h.PayloadLength)
	copy(body, data[:h.PayloadLength])
	a.Body = body
	return nil
}
