package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/ccoveille/go-safecast"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type payloadTag interface {
	IsPayload()
}

type Payload interface {
	payloadTag
	WriteTo(w io.Writer) (int64, error)
	ReadFrom(r io.Reader) (int64, error)
}

type U32 uint32

func (a U32) WriteTo(w io.Writer) (int64, error) {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], uint32(a))
	n, err := w.Write(b[:])
	return int64(n), err
}

func (a *U32) ReadFrom(r io.Reader) (int64, error) {
	b := [4]byte{}
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}
	*a = U32(binary.BigEndian.Uint32(b[:]))
	return int64(n), nil
}

type U64 uint64

func (a U64) WriteTo(w io.Writer) (int64, error) {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(a))
	n, err := w.Write(b[:])
	return int64(n), err
}

func (a *U64) ReadFrom(r io.Reader) (int64, error) {
	b := [8]byte{}
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}
	*a = U64(binary.BigEndian.Uint64(b[:]))
	return int64(n), nil
}

type U8String struct {
	S string
}

func NewU8String(s string) U8String {
	return U8String{S: s}
}

// MarshalBinary encodes U8String to binary form with a one byte length prefix.
func (a U8String) MarshalBinary() ([]byte, error) {
	l := len(a.S)
	if l > math.MaxUint8 {
		return nil, errors.New("string is too long")
	}

	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], a.S)
	return data, nil
}

// WriteTo writes U8String into io.Writer w in binary form.
func (a U8String) WriteTo(w io.Writer) (int64, error) {
	l := len(a.S)
	if l > math.MaxUint8 {
		return 0, errors.New("string is too long")
	}

	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], a.S)
	n, err := w.Write(data)
	return int64(n), err
}

func (a *U8String) ReadFrom(r io.Reader) (int64, error) {
	size := [1]byte{}
	n1, err := io.ReadFull(r, size[:])
	if err != nil {
		return int64(n1), err
	}
	str := make([]byte, size[0])
	n2, err := io.ReadFull(r, str)
	if err != nil {
		return int64(n1 + n2), err
	}
	a.S = string(str)
	return int64(n1 + n2), nil
}

type signaturePayload crypto.Signature

func (p *signaturePayload) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p[:])
	return int64(n), err
}

func (p *signaturePayload) ReadFrom(r io.Reader) (int64, error) {
	buf := [crypto.SignatureSize]byte{}
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return int64(n), err
	}
	s := crypto.Signature(buf)
	*p = signaturePayload(s)
	return int64(n), nil
}

type Signatures []crypto.Signature

func (p *Signatures) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(*p))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, s := range *p {
		sp := signaturePayload(s)
		n1, wErr := sp.WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
	}
	return n, nil
}

func (p *Signatures) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	s := make([]crypto.Signature, l)
	for i := range s {
		sp := signaturePayload{}
		n1, rErr := sp.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		s[i] = crypto.Signature(sp)
		n += n1
	}
	*p = s
	return n, nil
}

func (p *Signatures) IsPayload() {}

type BlockIDsPayload []BlockID

func (p *BlockIDsPayload) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(*p))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, id := range *p {
		switch id.idType {
		case SignatureID:
			n1, wErr := w.Write([]byte{crypto.SignatureSize})
			if wErr != nil {
				return n + int64(n1), wErr
			}
			n += int64(n1)
		case DigestID:
			n1, wErr := w.Write([]byte{crypto.DigestSize})
			if wErr != nil {
				return n + int64(n1), wErr
			}
			n += int64(n1)
		default:
			return n, fmt.Errorf("unsupported block ID type '%d'", id.idType)
		}
		n2, wErr := id.WriteTo(w)
		if wErr != nil {
			return n + n2, wErr
		}
		n += n2
	}
	return n, nil
}

func (p *BlockIDsPayload) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	ids := make([]BlockID, l)
	for i := range ids {
		var idSize [1]byte
		n1, rErr := io.ReadFull(r, idSize[:])
		if rErr != nil {
			return n + int64(n1), rErr
		}
		n += int64(n1)
		var id BlockID
		switch idSize[0] {
		case crypto.SignatureSize:
			id = BlockID{idType: SignatureID}
		case crypto.DigestSize:
			id = BlockID{idType: DigestID}
		default:
			return n, fmt.Errorf("unsupported block ID size '%d'", idSize[0])
		}
		n2, rErr := id.ReadFrom(r)
		if rErr != nil {
			return n + n2, rErr
		}
		ids[i] = id
		n += n2
	}
	*p = ids
	return n, nil
}

func (p *BlockIDsPayload) IsPayload() {}

type BytesPayload []byte

func (p *BytesPayload) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(*p)
	return int64(n), err
}

func (p *BytesPayload) ReadFrom(r io.Reader) (int64, error) {
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(r)
	if err != nil {
		return n, err
	}
	*p = buf.Bytes()
	return n, nil
}

func (p *BytesPayload) IsPayload() {}

type PeerInfos []PeerInfo

func (p *PeerInfos) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(*p))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, info := range (*p)[:l] {
		n1, wErr := info.WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
	}
	return n, nil
}

func (p *PeerInfos) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	ps := make([]PeerInfo, l)
	for i := range ps {
		info := PeerInfo{}
		n1, rErr := info.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		ps[i] = info
		n += n1
	}
	*p = ps
	return n, nil
}

func (p *PeerInfos) IsPayload() {}

// PayloadProducer is a function that creates an instance of Payload by provided PeerMessageID.
type PayloadProducer func(PeerMessageID) (Payload, error)

func CreatePayloadByContentID(contentID PeerMessageID) (Payload, error) {
	switch contentID {
	case ContentIDGetPeers:
		return &EmptyPayload{}, nil
	case ContentIDPeers:
		return &PeerInfos{}, nil
	case ContentIDGetSignatures, ContentIDSignatures:
		return &Signatures{}, nil
	case ContentIDGetBlock, ContentIDMicroblockRequest, ContentIDGetBlockSnapshot, ContentIDMicroBlockSnapshotRequest:
		return &BlockID{}, nil
	case ContentIDBlock, ContentIDScore, ContentIDTransaction, ContentIDMicroblock, ContentIDInvMicroblock,
		ContentIDPBBlock, ContentIDPBMicroBlock, ContentIDPBTransaction, ContentIDBlockSnapshot,
		ContentIDMicroBlockSnapshot:
		return &BytesPayload{}, nil
	case ContentIDGetBlockIDs, ContentIDBlockIDs:
		return &BlockIDsPayload{}, nil
	default:
		return nil, fmt.Errorf("unexpected content ID %d", contentID)
	}
}

type EmptyPayload struct{}

func (p *EmptyPayload) WriteTo(io.Writer) (int64, error) {
	return 0, nil
}

func (p *EmptyPayload) ReadFrom(io.Reader) (int64, error) {
	return 0, nil
}

func (p *EmptyPayload) IsPayload() {}
