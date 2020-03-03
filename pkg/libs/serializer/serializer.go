package serializer

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
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

func (a *Serializer) Write(b []byte) (int, error) {
	n, err := a.w.Write(b)
	if err != nil {
		return 0, err
	}
	a.n += n
	return n, nil
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

// StringWithUInt32Len writes to the buffer `buf` four bytes of the string's `s` length followed with the bytes of string itself.
func (a *Serializer) StringWithUInt32Len(s string) error {
	if len(s) > math.MaxUint32 {
		return errors.Errorf("too long string, expected max %d, found %d", math.MaxUint32, len(s))
	}
	err := a.Uint32(uint32(len(s)))
	if err != nil {
		return err
	}
	err = a.String(s)
	if err != nil {
		return err
	}
	return nil
}

func (a *Serializer) BytesWithUInt16Len(data []byte) error {
	sl := uint16(len(data))
	err := a.Uint16(sl)
	if err != nil {
		return err
	}
	return a.Bytes(data)
}

func (a *Serializer) BytesWithUInt32Len(data []byte) error {
	sl := uint32(len(data))
	err := a.Uint32(sl)
	if err != nil {
		return err
	}
	return a.Bytes(data)
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
