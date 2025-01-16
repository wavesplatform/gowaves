package proto

import (
	"bytes"
	"encoding/binary"
	"io"

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
		n1, wErr := id.WriteTo(w)
		if wErr != nil {
			return n + n1, wErr
		}
		n += n1
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
		id := BlockID{}
		n1, rErr := id.ReadFrom(r)
		if rErr != nil {
			return n + n1, rErr
		}
		(*a)[i] = id
		n += n1
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
