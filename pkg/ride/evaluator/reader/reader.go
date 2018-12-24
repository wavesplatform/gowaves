package reader

import (
	"encoding/base64"
	"encoding/binary"

	"github.com/pkg/errors"
)

const E_LONG byte = 0
const E_BYTES byte = 1
const E_STRING byte = 2
const E_IF byte = 3
const E_BLOCK byte = 4
const E_REF byte = 5
const E_TRUE byte = 6
const E_FALSE byte = 7
const E_GETTER byte = 8
const E_FUNCALL byte = 9

const FH_NATIVE byte = 0
const FH_USER byte = 1

var ErrUnexpectedEOF = errors.New("unexpected eof")

type BytesReader struct {
	bytes []byte
	pos   int
	len   int
}

func NewBytesReader(bytes []byte) *BytesReader {
	return &BytesReader{
		bytes: bytes,
		len:   len(bytes),
	}
}

func NewReaderFromBase64(base64String string) (*BytesReader, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}

	l := len(decoded)

	if l < 4 {
		return nil, errors.Errorf("expected script len at least 4 bytes, got %d", l)
	}

	return NewBytesReader(decoded[:l-4]), nil
}

func (a *BytesReader) Len() int {
	return a.len
}

func (a *BytesReader) Next() byte {
	a.pos += 1
	return a.bytes[a.pos-1]
}

func (a *BytesReader) Peek() byte {
	return a.bytes[a.pos]
}

func (a *BytesReader) ReadByte() byte {
	return a.Next()
}

func (a *BytesReader) ReadShort() int16 {
	rs := binary.BigEndian.Uint16(a.bytes[a.pos : a.pos+2])
	a.pos += 2
	return int16(rs)
}

func (a *BytesReader) ReadInt() int32 {
	rs := binary.BigEndian.Uint32(a.bytes[a.pos : a.pos+4])
	a.pos += 4
	return int32(rs)
}

func (a *BytesReader) Eof() bool {
	return a.pos >= a.len
}

func (a *BytesReader) ReadLong() int64 {
	out := int64(binary.BigEndian.Uint64(a.bytes[a.pos : a.pos+8]))
	a.pos += 8
	return out
}

func (a *BytesReader) ReadString() string {
	length := int(binary.BigEndian.Uint32(a.bytes[a.pos : a.pos+4]))
	a.pos += 4
	stringBytes := a.bytes[a.pos : a.pos+length]
	a.pos += length
	return string(stringBytes)
}

func (a *BytesReader) ReadBytes() []byte {
	n := int(a.ReadInt())
	out := a.bytes[a.pos : a.pos+n]
	a.pos += n
	return out
}

func (a *BytesReader) Pos() int {
	return a.pos
}
