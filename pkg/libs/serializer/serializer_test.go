package serializer

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"testing"
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
