package serializer

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
	"math"
)

type Serializer struct {
	w io.Writer
	n int
}

func New(w io.Writer) *Serializer {
	return &Serializer{
		w: w,
		n: 0,
	}
}

func (a *Serializer) StringWithUInt16Len(s string) error {
	if len(s) > math.MaxUint16 {
		return errors.Errorf("too long string, expected max %d, found %d", math.MaxUint16, len(s))
	}
	err := a.Uint16(uint16(len(s)))
	if err != nil {
		return err
	}
	err = a.String(s)
	if err != nil {
		return err
	}
	return nil
}

func (a *Serializer) Uint16(v uint16) error {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	n, err := a.w.Write(buf)
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) Uint32(v uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	n, err := a.w.Write(buf)
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) Uint64(v uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	n, err := a.w.Write(buf)
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) String(s string) error {
	n, err := a.w.Write([]byte(s))
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) Byte(b byte) error {
	n, err := a.w.Write([]byte{b})
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) N() int64 {
	return int64(a.n)
}

func (a *Serializer) Bool(b bool) error {
	var v byte = 0
	if b {
		v = 1
	}
	n, err := a.w.Write([]byte{v})
	if err != nil {
		return err
	}
	a.n += n
	return nil
}

func (a *Serializer) Bytes(b []byte) error {
	n, err := a.w.Write(b)
	if err != nil {
		return err
	}
	a.n += n
	return nil
}
