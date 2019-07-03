package proto

import (
	"io"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/deserializer"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	"go.uber.org/zap"
)

type Signer struct {
	Generator crypto.PublicKey
	Signature crypto.Signature
}

type MicroBlock struct {
	VersionField          byte
	PrevResBlockSigField  crypto.Signature
	TotalResBlockSigField crypto.Signature
	Signer                Signer
	TransactionCount      uint32
	Transactions          *TransactionsRepresentation
}

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

	a.Signer.Generator, err = d.PublicKey()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock public key")
	}

	a.Signer.Signature, err = d.Signature()
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal microblock signature")
	}

	return nil
}

func (a *MicroBlock) VerifySignature() (bool, error) {
	buf := bytebufferpool.Get()
	_, err := a.WriteWithoutSignature(buf)
	if err != nil {
		return false, err
	}

	return crypto.Verify(a.Signer.Generator, a.Signer.Signature, buf.Bytes()), nil
}

func (a *MicroBlock) WriteTo(w io.Writer) (int64, error) {
	n, _ := a.WriteWithoutSignature(w)
	n2, _ := w.Write(a.Signer.Signature[:])
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
	s.Bytes(a.Signer.Generator[:])
	return s.N(), nil
}

// CheckPointMessage represents a CheckPoint message
type MicroBlockMessage struct {
	Body []byte
}

func (*MicroBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	panic("implement me")
}

func (*MicroBlockMessage) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
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

	zap.S().Infof("header %+v", h)

	b := make([]byte, len(data[:h.PayloadLength]))
	copy(b, data)
	a.Body = b
	return nil
}

func (*MicroBlockMessage) MarshalBinary() (data []byte, err error) {
	panic("implement me")
}

type MicroBlockInvMessage struct {
	Body []byte
}

func (a *MicroBlockInvMessage) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockInvMessage) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockInvMessage) MarshalBinary() (data []byte, err error) {
	panic("implement me")
}

type MicroBlockRequestMessage struct {
	Body []byte
}

func (a *MicroBlockRequestMessage) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockRequestMessage) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockRequestMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = MaxHeaderLength + uint32(len(a.Body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDMicroblockRequest
	h.PayloadLength = uint32(len(a.Body))
	dig, err := crypto.FastHash(a.Body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])
	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return append(hdr, a.Body...), nil
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
	a.Body = make([]byte, h.PayloadLength)
	copy(a.Body, data)
	return nil
}

type MicroBlockRequest struct {
	TotalBlockSig crypto.Signature
}

func (a *MicroBlockRequest) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockRequest) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (a *MicroBlockRequest) UnmarshalBinary(data []byte) error {
	panic("implement me")
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
