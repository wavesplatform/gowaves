package reader

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
const E_BLOCK_V2 byte = 10
const E_LIST byte = 11

const DEC_LET byte = 0
const DEC_FUNC byte = 1

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

func ScriptBytesFromBase64Str(base64String string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}

	l := len(decoded)

	if l < 4 {
		return nil, errors.Errorf("expected script len at least 4 bytes, got %d", l)
	}

	d, err := crypto.SecureHash(decoded[:l-4])
	if err != nil {
		return nil, err
	}
	if bytes.Equal(d.Bytes()[:4], decoded[:l-4]) {
		return nil, errors.Errorf("invalid checksum, expected %+v, found %+v", d.Bytes()[:4], decoded[:l-4])
	}

	return decoded[:l-4], nil
}

func NewReaderFromBase64(base64String string) (*BytesReader, error) {
	decoded, err := ScriptBytesFromBase64Str(base64String)
	if err != nil {
		return nil, err
	}
	return NewBytesReader(decoded), nil
}

func ScriptBytesFromBase64(base64Bytes []byte) ([]byte, error) {
	res := make([]byte, len(base64Bytes))
	n, err := base64.StdEncoding.Decode(res, base64Bytes)
	if err != nil {
		return nil, err
	}
	if n < 4 {
		return nil, errors.Errorf("expected script len at least 4 bytes, got %d", n)
	}
	return res[:n-4], nil
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

func (a *BytesReader) ReadByte() (byte, error) {
	return a.Next(), nil
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

func (a *BytesReader) ReadByteString() []byte {
	length := int(binary.BigEndian.Uint32(a.bytes[a.pos : a.pos+4]))
	a.pos += 4
	stringBytes := a.bytes[a.pos : a.pos+length]
	a.pos += length
	return stringBytes
}

func (a *BytesReader) ReadBytes() []byte {
	n := int(a.ReadInt())
	out := a.bytes[a.pos : a.pos+n]
	a.pos += n
	return out
}

func (a *BytesReader) Content() []byte {
	return a.bytes
}

func (a *BytesReader) Pos() int {
	return a.pos
}

func (a *BytesReader) Rest() []byte {
	return a.bytes[a.pos:]
}
