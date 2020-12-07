package ride

import (
	"bytes"
	"encoding/binary"
)

const (
	s0 byte = iota
	s1
	s2
	s3
	s4
	s5
	s6
	s7
	s8
	s9
	strue
	sfalse
	sint
	suint16
	sbytes
	sstring
	spoint
)

type Serializer struct {
	b bytes.Buffer
}

func (a *Serializer) Int(v rideInt) {
	if v >= 0 && v <= 9 {
		a.b.WriteByte(byte(v))
		return
	}
	a.b.WriteByte(sint)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	a.b.Write(b)
}

func (a *Serializer) Uint16(v uint16) {
	a.Int(rideInt(v))
}

func (a *Serializer) Point(p point) {
	a.b.WriteByte(spoint)
	a.Uint16(p.position)
}
