package serializer

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/pkg/errors"
)

// NonFallableSerializer writes big-endian primitives to the underlying writer ignoring write errors.
// Primitive writes go through the internal scratch array, so they do not allocate as long as the
// underlying writer does not retain the passed slice (as required by the io.Writer contract).
type NonFallableSerializer struct {
	w       io.Writer
	n       int
	scratch [8]byte
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
	binary.BigEndian.PutUint16(a.scratch[:2], v)
	a.Bytes(a.scratch[:2])
}

func (a *NonFallableSerializer) Uint32(v uint32) {
	binary.BigEndian.PutUint32(a.scratch[:4], v)
	a.Bytes(a.scratch[:4])
}

func (a *NonFallableSerializer) Uint64(v uint64) {
	binary.BigEndian.PutUint64(a.scratch[:8], v)
	a.Bytes(a.scratch[:8])
}

func (a *NonFallableSerializer) Int64(v int64) {
	a.Uint64(uint64(v)) // #nosec G115: two's complement reinterpretation is intended.
}

// String writes the string bytes using io.WriteString, avoiding the conversion to a byte slice
// when the underlying writer implements io.StringWriter.
func (a *NonFallableSerializer) String(s string) {
	n, _ := io.WriteString(a.w, s)
	a.n += n
}

func (a *NonFallableSerializer) Byte(b byte) {
	a.scratch[0] = b
	a.Bytes(a.scratch[:1])
}

func (a *NonFallableSerializer) N() int64 {
	return int64(a.n)
}

func (a *NonFallableSerializer) Bool(b bool) {
	var v byte = 0
	if b {
		v = 1
	}
	a.Byte(v)
}

func (a *NonFallableSerializer) Bytes(b []byte) {
	n, _ := a.w.Write(b)
	a.n += n
}
