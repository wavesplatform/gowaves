package proto

import (
	"encoding/binary"
	"fmt"
)

const (
	HeaderLength         = 17
	HandshakeFixedLength = 272
)

type Header struct {
	Length        uint32
	Magic         uint32
	ContentID     uint8
	PayloadLength uint32
	PayloadCsum   uint32
}

type Handshake struct {
	NameLength         uint8
	Name               string
	VersionMajor       uint32
	VersionMinor       uint32
	VersionPatch       uint32
	NodeNameLength     uint8
	NodeName           string
	NodeNonce          uint64
	DeclaredAddrLength uint32
	DeclaredAddrBytes  []byte
	Timestamp          uint64
}

func (h *Handshake) binaryLen() int {
	return int(h.NameLength) + int(h.NodeNameLength) + HandshakeFixedLength
}

func (h *Handshake) marshalBinaryName() ([]byte, error) {
	data := make([]byte, h.NameLength+1)
	data[0] = h.NameLength
	copy(data[1:h.NameLength], h.Name)

	return data, nil
}

func (h *Handshake) marshalBinaryVersion() ([]byte, error) {
	data := make([]byte, 96)

	binary.BigEndian.PutUint32(data[0:4], h.VersionMajor)
	binary.BigEndian.PutUint32(data[4:8], h.VersionMinor)
	binary.BigEndian.PutUint32(data[8:12], h.VersionPatch)

	return data, nil
}

func (h *Handshake) marshalBinaryNodeName() ([]byte, error) {
	data := make([]byte, h.NodeNameLength+1)

	data[0] = h.NodeNameLength
	copy(data[1:h.NodeNameLength], h.NodeName)

	return data, nil
}

func (h *Handshake) marshalBinaryAddr() ([]byte, error) {
	data := make([]byte, 20+h.DeclaredAddrLength)

	binary.BigEndian.PutUint64(data[0:8], h.NodeNonce)
	binary.BigEndian.PutUint32(data[8:12], h.DeclaredAddrLength)

	copy(data[12:12+h.DeclaredAddrLength], h.DeclaredAddrBytes)
	binary.BigEndian.PutUint64(data[12+h.DeclaredAddrLength:20+h.DeclaredAddrLength], h.Timestamp)

	return data, nil
}

func (h *Handshake) MarshalBinary() ([]byte, error) {
	data1, err := h.marshalBinaryName()
	if err != nil {
		return nil, err
	}
	data2, err := h.marshalBinaryVersion()
	if err != nil {
		return nil, err
	}
	data3, err := h.marshalBinaryNodeName()
	if err != nil {
		return nil, err
	}
	data4, err := h.marshalBinaryAddr()
	if err != nil {
		return nil, err
	}

	data1 = append(data1, data2...)
	data1 = append(data1, data3...)
	data1 = append(data1, data4...)
	return data1, nil
}

func (h *Header) MarshalBinary() ([]byte, error) {
	data := make([]byte, HeaderLength)

	binary.BigEndian.PutUint32(data[0:4], h.Length)
	binary.BigEndian.PutUint32(data[4:8], h.Magic)
	data[8] = h.ContentID
	binary.BigEndian.PutUint32(data[9:13], h.PayloadLength)
	binary.BigEndian.PutUint32(data[13:17], h.PayloadCsum)

	return data, nil
}

func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) < HeaderLength {
		return fmt.Errorf("cannot unmarshal into header: got %v bytes, need %v", len(data), HeaderLength)
	}

	h.Length = binary.BigEndian.Uint32(data[0:4])
	h.Magic = binary.BigEndian.Uint32(data[4:8])
	h.ContentID = data[8]
	h.PayloadLength = binary.BigEndian.Uint32(data[9:13])
	h.PayloadCsum = binary.BigEndian.Uint32(data[13:17])

	return nil
}
