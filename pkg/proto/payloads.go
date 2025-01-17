package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/ccoveille/go-safecast"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

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

type SignaturePayload crypto.Signature

func (a SignaturePayload) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(crypto.Signature(a).Bytes())
	return int64(n), err
}

func (a *SignaturePayload) ReadFrom(r io.Reader) (int64, error) {
	buf := [crypto.SignatureSize]byte{}
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return int64(n), err
	}
	s := crypto.Signature{}
	copy(s[:], buf[:])
	*a = SignaturePayload(s)
	return int64(n), nil
}

type Signatures []crypto.Signature

func (a Signatures) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(a))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, s := range a {
		n1, wErr := SignaturePayload(s).WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
	}
	return n, nil
}

func (a *Signatures) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	*a = make([]crypto.Signature, l)
	for i := range *a {
		sp := SignaturePayload{}
		n1, rErr := sp.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		(*a)[i] = crypto.Signature(sp)
		n += n1
	}
	return n, nil
}

type BlockIDsPayload []BlockID

func (a BlockIDsPayload) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(a))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, id := range a {
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
			return n, errors.New("unsupported block ID type")
		}
		n2, wErr := id.WriteTo(w)
		if wErr != nil {
			return n + n2, wErr
		}
		n += n2
	}
	return n, nil
}

func (a *BlockIDsPayload) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	*a = make([]BlockID, l)
	for i := range *a {
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
			return n, errors.New("unsupported block ID size")
		}
		n2, rErr := id.ReadFrom(r)
		if rErr != nil {
			return n + n2, rErr
		}
		(*a)[i] = id
		n += n2
	}
	return n, nil
}

type BytesPayload []byte

func (p BytesPayload) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
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

type CheckpointPayload []CheckpointItem

func (a CheckpointPayload) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(a))
	if err != nil {
		return 0, err
	}
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, item := range a {
		n1, wErr := item.WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
	}
	return n, nil
}

func (a *CheckpointPayload) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	*a = make([]CheckpointItem, l)
	for i := range *a {
		item := CheckpointItem{}
		n1, rErr := item.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		(*a)[i] = item
		n += n1
	}
	return n, nil
}

type PeersPayload []PeerInfo

func (a PeersPayload) WriteTo(w io.Writer) (int64, error) {
	l, err := safecast.ToUint32(len(a))
	if err != nil {
		return 0, err
	}
	const maxPeersCount = 1000
	l = min(maxPeersCount, l)
	n, err := U32(l).WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, info := range a[:l] {
		n1, wErr := info.WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
	}
	return n, nil
}

func (a *PeersPayload) ReadFrom(r io.Reader) (int64, error) {
	var l U32
	n, err := l.ReadFrom(r)
	if err != nil {
		return n, err
	}
	*a = make([]PeerInfo, l)
	for i := range *a {
		info := PeerInfo{}
		n1, rErr := info.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		(*a)[i] = info
		n += n1
	}
	return n, nil
}
