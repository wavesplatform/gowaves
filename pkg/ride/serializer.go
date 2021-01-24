package ride

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

const (
	strue    byte = 101
	sfalse        = 102
	sint          = 103
	suint16       = 104
	sbytes        = 105
	sString       = 106
	spoint        = 107
	sMap          = 108
	sNoValue      = 109
)

type Serializer struct {
	b *bytes.Buffer
}

func NewSerializer() Serializer {
	return Serializer{b: &bytes.Buffer{}}
}

func (a *Serializer) RideInt(v rideInt) error {
	if v >= 0 && v <= 100 {
		a.b.WriteByte(byte(v))
		return nil
	}
	a.b.WriteByte(sint)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	a.b.Write(b)
	return nil
}

func (a *Serializer) RideNoValue() error {
	return a.b.WriteByte(sNoValue)
}

func (a *Serializer) Tuple(values ...rideType) error {
	panic("not implemented")
}

func (a *Serializer) Uint16(v uint16) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	a.b.Write(b)
}

func (a *Serializer) Bool(v bool) {
	if v {
		a.b.WriteByte(strue)
	} else {
		a.b.WriteByte(sfalse)
	}
}

func (a *Serializer) Point(p point) {
	a.b.WriteByte(spoint)
	a.Uint16(p.position)
}

func (a *Serializer) Byte(b byte) {
	a.b.WriteByte(b)
}

func (a *Serializer) RideBytes(v rideBytes) error {
	a.b.WriteByte(sbytes)
	return a.Bytes(v)
}

func (a *Serializer) Bytes(v []byte) error {
	if len(v) > math.MaxUint16 {
		return errors.New("bytes length overflow")
	}
	a.Uint16(uint16(len(v)))
	a.b.Write(v)
	return nil
}

func (a *Serializer) Source() []byte {
	return a.b.Bytes()
}

func (a *Serializer) RideMap(size int) error {
	a.b.WriteByte(sMap)
	if size > math.MaxUint16 {
		return errors.New("size overflow")
	}
	a.Uint16(uint16(size))
	return nil
}

func (a *Serializer) RideString(v rideString) error {
	a.b.WriteByte(sString)
	return a.String(v)
}

func (a *Serializer) String(v rideString) error {
	if len(v) > math.MaxUint16 {
		return errors.New("bytes length overflow")
	}
	a.Uint16(uint16(len(v)))
	a.b.Write([]byte(v))
	return nil
}

func (a Serializer) RideBool(v rideBoolean) error {
	a.Bool(bool(v))
	return nil
}

func (a *Serializer) Map(size int, f func(Map) error) error {
	err := a.RideMap(size)
	if err != nil {
		return err
	}
	return f(a)
}

type Map interface {
	Uint16(v uint16)
	RideInt(v rideInt) error
	RideBytes(v rideBytes) error
	RideString(v rideString) error
}
