package serializer

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonFallableSerializer_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	o := bytes.NewBuffer([]byte{1, 2, 3, 4, 5})
	s := NewNonFallable(buf)
	_, _ = o.WriteTo(s)

	require.EqualValues(t, 5, s.N())
	require.Equal(t, []byte{1, 2, 3, 4, 5}, buf.Bytes())
}

func writeAllNonFallable(s *NonFallableSerializer) {
	s.Byte(0xab)
	s.Bool(true)
	s.Bool(false)
	s.Uint16(0x0102)
	s.Uint32(0x03040506)
	s.Uint64(0x0708090a0b0c0d0e)
	s.Int64(-1)
	s.String("str")
	_ = s.StringWithUInt16Len("ab")
	s.Bytes([]byte{0xff})
}

func expectedAllNonFallable() []byte {
	return []byte{
		0xab, 1, 0,
		0x01, 0x02,
		0x03, 0x04, 0x05, 0x06,
		0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		's', 't', 'r',
		0, 2, 'a', 'b',
		0xff,
	}
}

func TestNonFallableSerializer_AllPrimitivesBytes(t *testing.T) {
	buf := &bytes.Buffer{}
	s := NewNonFallable(buf)
	writeAllNonFallable(s)
	require.Equal(t, expectedAllNonFallable(), buf.Bytes())
	require.EqualValues(t, len(expectedAllNonFallable()), s.N())
	require.Error(t, s.StringWithUInt16Len(strings.Repeat("a", math.MaxUint16+1)))
}

type plainNFWriter struct{ b []byte }

func (w *plainNFWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func TestNonFallableSerializer_AllPrimitivesPlainWriter(t *testing.T) {
	w := &plainNFWriter{}
	s := NewNonFallable(w)
	writeAllNonFallable(s)
	require.Equal(t, expectedAllNonFallable(), w.b)
}

func TestNonFallableSerializer_PrimitivesDoNotAllocate(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 0, 1<<20))
	s := NewNonFallable(buf)
	data := []byte{1, 2, 3}
	allocs := testing.AllocsPerRun(1000, func() {
		s.Byte(1)
		s.Bool(true)
		s.Uint16(1)
		s.Uint32(1)
		s.Uint64(1)
		s.Int64(1)
		s.String("abc")
		_ = s.StringWithUInt16Len("abc")
		s.Bytes(data)
	})
	require.Zero(t, allocs)
}
