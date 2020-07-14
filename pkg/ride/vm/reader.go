package vm

import "encoding/binary"

type Reader struct {
	pos  int
	code []byte
}

func NewReader(code []byte) *Reader {
	return &Reader{code: code}
}

func (a *Reader) HasNext() bool {
	return a.pos+1 < len(a.code)
}

func (a *Reader) Next() byte {
	b := a.code[a.pos]
	a.pos++
	return b
}

func (a *Reader) Skip(n int) {
	a.pos += n
}

func (a *Reader) String() string {
	size := a.Short()
	bts := a.Read(int(size))
	return string(bts)
}

func (a *Reader) Bytes() []byte {
	size := a.Short()
	bts := a.Read(int(size))
	return bts
}

func (a *Reader) I32() int32 {
	rs := binary.BigEndian.Uint32(a.code[a.pos:])
	a.pos += 4
	return int32(rs)
}

func (a *Reader) Short() uint16 {
	rs := binary.BigEndian.Uint16(a.code[a.pos : a.pos+2])
	a.pos += 2
	return rs
}

func (a *Reader) Long() int64 {
	rs := binary.BigEndian.Uint64(a.code[a.pos : a.pos+8])
	a.pos += 8
	return int64(rs)
}

func (a *Reader) Read(n int) []byte {
	out := a.code[a.pos : a.pos+n]
	a.pos += n
	return out
}

func (a *Reader) Pos() int {
	return a.pos
}

func (a *Reader) Jmp(to int) {
	a.pos = to
}

func (a *Reader) Move(i32 int32) {
	a.pos = int(i32)
}
