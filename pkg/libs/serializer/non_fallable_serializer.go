package serializer

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

type NonFallableSerializer struct {
	w io.Writer
	n int
}

func NewNonFallable(w io.Writer) *NonFallableSerializer {
	return &NonFallableSerializer{
		w: w,
		n: 0,
	}
}

func (a *NonFallableSerializer) Write(b []byte) (int, error) {
	n, err := a.w.Write(b)
	if err != nil {
		return 0, err
	}
	a.n += n
	return n, nil
}

func (a *NonFallableSerializer) StringWithUInt16Len(s string) error {
	if len(s) > math.MaxUint16 {
		return errors.Errorf("too long string, expected max %d, found %d", math.MaxUint16, len(s))
	}
	a.Uint16(uint16(len(s)))
	a.String(s)
	return nil
}

func (a *NonFallableSerializer) Uint16(v uint16) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	n, _ := a.w.Write(buf)
	a.n += n
}

func (a *NonFallableSerializer) Uint32(v uint32) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	n, _ := a.w.Write(buf)
	a.n += n
}

func (a *NonFallableSerializer) Uint64(v uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	n, _ := a.w.Write(buf)
	a.n += n
}

func (a *NonFallableSerializer) Int64(v int64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	n, _ := a.w.Write(buf)
	a.n += n
}

func (a *NonFallableSerializer) String(s string) {
	n, _ := a.w.Write([]byte(s))
	a.n += n
}

func (a *NonFallableSerializer) Byte(b byte) {
	n, _ := a.w.Write([]byte{b})
	a.n += n
}

func (a *NonFallableSerializer) N() int64 {
	return int64(a.n)
}

func (a *NonFallableSerializer) Bool(b bool) {
	var v byte = 0
	if b {
		v = 1
	}
	n, _ := a.w.Write([]byte{v})
	a.n += n
}

func (a *NonFallableSerializer) Bytes(b []byte) {
	n, _ := a.w.Write(b)
	a.n += n
}
