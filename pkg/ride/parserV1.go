package ride

import (
	"bytes"
	"encoding/binary"
)

func newParserV1(r *bytes.Reader, id [32]byte, header scriptHeader) *parser {
	p := &parser{
		r:      r,
		id:     id,
		header: header,
	}
	p.readShort = readShortV1
	p.readInt = readIntV1
	p.readLong = readLongV1
	return p
}

func readShortV1(r *bytes.Reader) (int16, error) {
	buf := [2]byte{}
	_, err := r.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf[:])), nil
}

func readIntV1(r *bytes.Reader) (int32, error) {
	buf := [4]byte{}
	_, err := r.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

func readLongV1(r *bytes.Reader) (int64, error) {
	buf := [8]byte{}
	_, err := r.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf[:])), nil
}
