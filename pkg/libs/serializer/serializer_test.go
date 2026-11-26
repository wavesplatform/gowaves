package serializer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerializer_Byte(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.Byte('b'))
	require.Equal(t, []byte{'b'}, buf.Bytes())
	require.EqualValues(t, 1, s.N())
}

func TestSerializer_Uint16(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.Uint16(257))
	require.Equal(t, []byte{1, 1}, buf.Bytes())
	require.EqualValues(t, 2, s.N())
}

func TestSerializer_StringWithUInt16Len(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.StringWithUInt16Len("abc"))
	require.Equal(t, []byte{0, 3, 'a', 'b', 'c'}, buf.Bytes())
	require.EqualValues(t, 5, s.N())
}

func TestSerializer_Uint32(t *testing.T) {
	var billion uint32 = 1000000000
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.Uint32(billion))
	require.Equal(t, binary.BigEndian.Uint32(buf.Bytes()), billion)
}

func TestSerializer_Uint64(t *testing.T) {
	var billion uint64 = 1000000000
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.Uint64(billion))
	require.Equal(t, binary.BigEndian.Uint64(buf.Bytes()), billion)
}

func TestSerializer_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	o := bytes.NewBuffer([]byte{1, 2, 3, 4, 5})
	s := New(buf)
	_, _ = o.WriteTo(s)

	require.EqualValues(t, 5, s.N())
	require.Equal(t, []byte{1, 2, 3, 4, 5}, buf.Bytes())
}

func BenchmarkSerializer_Bytes(b *testing.B) {
	b.ReportAllocs()

	buf := bytes.NewBuffer(make([]byte, 1024))
	s := New(buf)
	data := []byte{1, 2}
	for b.Loop() {
		_, _ = s.Write(data)
	}
}

func TestSerializer_StringWithUInt32Len(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.StringWithUInt32Len("abc"))
	require.Equal(t, []byte{0, 0, 0, 3, 'a', 'b', 'c'}, buf.Bytes())
	require.EqualValues(t, 7, s.N())
}

func TestSerializer_BytesWithUInt32Len(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	require.NoError(t, s.BytesWithUInt32Len([]byte("abc")))
	require.Equal(t, []byte{0, 0, 0, 3, 'a', 'b', 'c'}, buf.Bytes())
	require.EqualValues(t, 7, s.N())
}

func writeAllPrimitives(t testing.TB, s *Serializer) {
	require.NoError(t, s.Byte(0xab))
	require.NoError(t, s.Bool(true))
	require.NoError(t, s.Bool(false))
	require.NoError(t, s.Uint16(0x0102))
	require.NoError(t, s.Uint32(0x03040506))
	require.NoError(t, s.Uint64(0x0708090a0b0c0d0e))
	require.NoError(t, s.String("str"))
	require.NoError(t, s.StringWithUInt16Len("ab"))
	require.NoError(t, s.StringWithUInt32Len("cd"))
	require.NoError(t, s.Bytes([]byte{0xff}))
	require.NoError(t, s.BytesWithUInt16Len([]byte{0xee}))
	require.NoError(t, s.BytesWithUInt32Len([]byte{0xdd}))
}

func expectedAllPrimitives() []byte {
	return []byte{
		0xab, 1, 0,
		0x01, 0x02,
		0x03, 0x04, 0x05, 0x06,
		0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
		's', 't', 'r',
		0, 2, 'a', 'b',
		0, 0, 0, 2, 'c', 'd',
		0xff,
		0, 1, 0xee,
		0, 0, 0, 1, 0xdd,
	}
}

func TestSerializer_AllPrimitivesBytes(t *testing.T) {
	buf := &bytes.Buffer{}
	s := New(buf)
	writeAllPrimitives(t, s)
	require.Equal(t, expectedAllPrimitives(), buf.Bytes())
	require.EqualValues(t, len(expectedAllPrimitives()), s.N())
}

// plainWriter does not implement io.StringWriter, to check the io.WriteString fallback.
type plainWriter struct{ b []byte }

func (w *plainWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func TestSerializer_AllPrimitivesPlainWriter(t *testing.T) {
	w := &plainWriter{}
	s := New(w)
	writeAllPrimitives(t, s)
	require.Equal(t, expectedAllPrimitives(), w.b)
	require.EqualValues(t, len(expectedAllPrimitives()), s.N())
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("fail") }

func TestSerializer_Errors(t *testing.T) {
	s := New(failingWriter{})
	require.Error(t, s.Byte(1))
	require.Error(t, s.Bool(true))
	require.Error(t, s.Uint16(1))
	require.Error(t, s.Uint32(1))
	require.Error(t, s.Uint64(1))
	require.Error(t, s.String("a"))
	require.Error(t, s.StringWithUInt16Len("a"))
	require.Error(t, s.StringWithUInt32Len("a"))
	require.Error(t, s.Bytes([]byte{1}))
	require.Error(t, s.BytesWithUInt16Len([]byte{1}))
	require.Error(t, s.BytesWithUInt32Len([]byte{1}))
	_, err := s.Write([]byte{1})
	require.Error(t, err)
	require.EqualValues(t, 0, s.N())
	require.Error(t, New(&bytes.Buffer{}).StringWithUInt16Len(strings.Repeat("a", math.MaxUint16+1)))
}

func TestSerializer_PrimitivesDoNotAllocate(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 0, 1<<20))
	s := New(buf)
	data := []byte{1, 2, 3}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = s.Byte(1)
		_ = s.Bool(true)
		_ = s.Uint16(1)
		_ = s.Uint32(1)
		_ = s.Uint64(1)
		_ = s.String("abc")
		_ = s.StringWithUInt16Len("abc")
		_ = s.StringWithUInt32Len("abc")
		_ = s.Bytes(data)
		_ = s.BytesWithUInt16Len(data)
		_ = s.BytesWithUInt32Len(data)
	})
	require.Zero(t, allocs)
}

func BenchmarkSerializer_Primitives(b *testing.B) {
	b.ReportAllocs()
	buf := bytes.NewBuffer(make([]byte, 0, 1<<10))
	s := New(buf)
	for b.Loop() {
		buf.Reset()
		_ = s.Byte(1)
		_ = s.Bool(true)
		_ = s.Uint16(1)
		_ = s.Uint32(1)
		_ = s.Uint64(1)
		_ = s.StringWithUInt16Len("abc")
	}
}
