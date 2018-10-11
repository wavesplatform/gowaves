package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"net"
)

const (
	headerLength  = 17
	headerMagic   = 0x12345678
	headerCsumLen = 4
)

const (
	contentIDGetPeers      = 0x1
	contentIDPeers         = 0x2
	contentIDGetSignatures = 0x14
	contentIDSignatures    = 0x15
	contentIDGetBlock      = 0x16
	contentIDBlock         = 0x17
	contentIDScore         = 0x18
	contentIDTransaction   = 0x19
	contentIDCheckpoint    = 0x64
)

type BlockSignature crypto.Signature

type header struct {
	Length        uint32
	Magic         uint32
	ContentID     uint8
	PayloadLength uint32
	PayloadCsum   [headerCsumLen]byte
}

func (h *header) MarshalBinary() ([]byte, error) {
	data := make([]byte, 17)

	binary.BigEndian.PutUint32(data[0:4], h.Length)
	binary.BigEndian.PutUint32(data[4:8], headerMagic)
	data[8] = h.ContentID
	binary.BigEndian.PutUint32(data[9:13], h.PayloadLength)
	copy(data[13:17], h.PayloadCsum[:])

	return data, nil
}

func (h *header) UnmarshalBinary(data []byte) error {
	if len(data) < headerLength-4 {
		return fmt.Errorf("data is to short to unmarshal header: %d", len(data))
	}
	h.Length = binary.BigEndian.Uint32(data[0:4])
	h.Magic = binary.BigEndian.Uint32(data[4:8])
	if h.Magic != headerMagic {
		return fmt.Errorf("received wrong magic: want %x, have %x", headerMagic, h.Magic)
	}
	h.ContentID = data[8]
	h.PayloadLength = binary.BigEndian.Uint32(data[9:13])
	if len(data) == headerLength {
		copy(h.PayloadCsum[:], data[13:17])
	}

	return nil
}

type Handshake struct {
	Name              string
	VersionMajor      uint32
	VersionMinor      uint32
	VersionPatch      uint32
	NodeName          string
	NodeNonce         uint64
	DeclaredAddrBytes []byte
	Timestamp         uint64
}

type GetPeersMessage struct{}

func (m *GetPeersMessage) MarshalBinary() ([]byte, error) {
	var header header

	header.Length = headerLength - 8
	header.Magic = headerMagic
	header.ContentID = contentIDGetPeers
	header.PayloadLength = 0
	var empty [0]byte
	dig, err := crypto.FastHash(empty[:])
	if err != nil {
		return nil, err
	}
	copy(header.PayloadCsum[:], dig[:4])

	res, err := header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return res[:headerLength-4], nil
}

func (m *GetPeersMessage) UnmarshalBinary(b []byte) error {
	var header header

	err := header.UnmarshalBinary(b)
	if err != nil {
		return err
	}

	if header.Length != headerLength-8 {
		return fmt.Errorf("getpeers message length is unexpected: want %v have %v", headerLength, header.Length)
	}
	if header.Magic != headerMagic {
		return fmt.Errorf("getpeers message magic is unexpected: want %x have %x", headerMagic, header.Magic)
	}
	if header.ContentID != contentIDGetPeers {
		return fmt.Errorf("getpeers message contentid is unexpected: want %x have %x", contentIDGetPeers, header.ContentID)
	}
	if header.PayloadLength != 0 {
		return fmt.Errorf("getpeers message length is not zero")
	}

	return nil
}

type PeerInfo struct {
	addr net.IP
	port uint16
}

func (m *PeerInfo) MarshalBinary() ([]byte, error) {
	buffer := make([]byte, 8)

	copy(buffer[0:4], m.addr.To4())
	binary.BigEndian.PutUint32(buffer[4:8], uint32(m.port))

	return buffer, nil
}

func (m *PeerInfo) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("too short")
	}

	m.addr = net.IPv4(data[0], data[1], data[2], data[3])
	m.port = uint16(binary.BigEndian.Uint32(data[4:8]))

	return nil
}

type PeersMessage struct {
	Peers []PeerInfo
}

func (m *PeersMessage) MarshalBinary() ([]byte, error) {
	var h header
	body := make([]byte, 4)

	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Peers)))

	for _, k := range m.Peers {
		peer, err := k.MarshalBinary()
		if err != nil {
			return nil, err
		}
		body = append(body, peer...)
	}

	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDPeers
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	hdr = append(hdr, body...)

	return hdr, nil
}

func (m *PeersMessage) UnmarshalBinary(data []byte) error {
	var header header
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	data = data[headerLength:]
	if len(data) < 4 {
		return errors.New("peers message has insufficient length")
	}
	peersCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	for i := uint32(0); i < peersCount; i += 8 {
		var peer PeerInfo
		if err := peer.UnmarshalBinary(data[i : i+8]); err != nil {
			return err
		}
		m.Peers = append(m.Peers, peer)
	}

	return nil
}

type BlockID [64]byte

type GetSignaturesMessage struct {
	Blocks []BlockID
}

func (m *GetSignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Blocks)*64)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, b := range m.Blocks {
		body = append(body, b[:]...)
	}

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDGetSignatures
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *GetSignaturesMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDGetSignatures {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	blockCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]

	for i := uint32(0); i < blockCount; i++ {
		var b BlockID
		if len(data[i:]) < 64 {
			return fmt.Errorf("message too short %v", len(data))
		}
		copy(b[:], data[i:i+64])
		m.Blocks = append(m.Blocks, b)
	}

	return nil
}

type SignaturesMessage struct {
	Signatures []BlockSignature
}

func (m *SignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures))
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDSignatures
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *SignaturesMessage) UnmarshalBinary(data []byte) error {
	var h header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDSignatures {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	sigCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]

	for i := uint32(0); i < sigCount; i++ {
		var sig BlockSignature
		if len(data[i:]) < 64 {
			return fmt.Errorf("message too short: %v", len(data))
		}
		copy(sig[:], data[i:i+64])
		m.Signatures = append(m.Signatures, sig)
	}

	return nil
}

type GetBlockMessage struct {
	BlockID BlockID
}

func (m *GetBlockMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 0, 64)
	body = append(body, m.BlockID[:]...)

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDGetBlock
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

func (m *GetBlockMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}

	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDGetBlock {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 64 {
		return fmt.Errorf("message too short %v", len(data))
	}

	copy(m.BlockID[:], data[:64])

	return nil
}

type BlockMessage struct {
	BlockBytes []byte
}

func (m *BlockMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.BlockBytes)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDBlock
	h.PayloadLength = uint32(len(m.BlockBytes))
	dig, err := crypto.FastHash([]byte{})
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.BlockBytes...)
	return hdr, nil
}

func (m *BlockMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDBlock {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}

	m.BlockBytes = data[17:]

	return nil
}

type ScoreMessage struct {
	Score []byte
}

func (m *ScoreMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.Score)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDScore
	h.PayloadLength = uint32(len(m.Score))
	dig, err := crypto.FastHash(m.Score)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.Score...)
	return hdr, nil
}

func (m *ScoreMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDScore {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}

	m.Score = data[17:]

	return nil
}

type TransactionMessage struct {
	Transaction []byte
}

func (m *TransactionMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.Transaction)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDTransaction
	h.PayloadLength = uint32(len(m.Transaction))
	dig, err := crypto.FastHash([]byte{})
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.Transaction...)
	return hdr, nil
}

func (m *TransactionMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != contentIDTransaction {
		return fmt.Errorf("wrong content idsig in header: %x", h.ContentID)
	}

	m.Transaction = data[17:]

	return nil
}

type CheckpointItem struct {
	Height    uint64
	Signature BlockSignature
}

type CheckPointMessage struct {
	Checkpoints []CheckpointItem
}

func (m *CheckPointMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Checkpoints)*72+100)

	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Checkpoints)))
	for _, c := range m.Checkpoints {
		var height [8]byte
		binary.BigEndian.PutUint64(height[0:8], c.Height)
		body = append(body, height[:]...)
		body = append(body, c.Signature[:]...)
	}

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = contentIDCheckpoint
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	hdr = append(hdr, body...)
	return hdr, nil

}

func (m *CheckPointMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("ckeckpoint message magic is unexpected: %x", headerMagic)
	}
	if h.ContentID != contentIDCheckpoint {
		return fmt.Errorf("checkpoint message contentid is unexpected %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("checkpoint message data too short: %d", len(data))
	}
	checkpointsCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	for i := uint32(0); i < checkpointsCount; i++ {
		if len(data) < 72 {
			return fmt.Errorf("checkpoint message data too short")
		}
		var ci CheckpointItem
		ci.Height = binary.BigEndian.Uint64(data[0:8])
		copy(ci.Signature[:], data[8:72])
		data = data[72:]
		m.Checkpoints = append(m.Checkpoints, ci)
	}

	return nil
}

func (h *Handshake) marshalBinaryName() ([]byte, error) {
	if len(h.Name) > 255 {
		return nil, errors.New("handshake application name too long")
	}
	data := make([]byte, len(h.Name)+1)
	data[0] = byte(len(h.Name))
	copy(data[1:1+len(h.Name)], h.Name)

	return data, nil
}

func (h *Handshake) marshalBinaryVersion() ([]byte, error) {
	data := make([]byte, 12)

	binary.BigEndian.PutUint32(data[0:4], h.VersionMajor)
	binary.BigEndian.PutUint32(data[4:8], h.VersionMinor)
	binary.BigEndian.PutUint32(data[8:12], h.VersionPatch)

	return data, nil
}

func (h *Handshake) marshalBinaryNodeName() ([]byte, error) {
	if len(h.NodeName) > 255 {
		return nil, errors.New("handshake node name too long")
	}
	l := len(h.NodeName)
	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], h.NodeName)

	return data, nil
}

func (h *Handshake) marshalBinaryAddr() ([]byte, error) {
	data := make([]byte, 20+len(h.DeclaredAddrBytes))

	binary.BigEndian.PutUint64(data[0:8], h.NodeNonce)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(h.DeclaredAddrBytes)))

	copy(data[12:12+len(h.DeclaredAddrBytes)], h.DeclaredAddrBytes)
	binary.BigEndian.PutUint64(data[12+len(h.DeclaredAddrBytes):20+len(h.DeclaredAddrBytes)], h.Timestamp)

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

func (h *Handshake) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return errors.New("data too short")
	}
	appNameLen := data[0]
	data = data[1:]
	if len(data) < int(appNameLen) {
		return errors.New("data too short")
	}
	h.Name = string(data[:appNameLen])
	data = data[appNameLen:]
	if len(data) < 13 {
		return errors.New("data too short")
	}
	h.VersionMajor = binary.BigEndian.Uint32(data[0:4])
	h.VersionMinor = binary.BigEndian.Uint32(data[4:8])
	h.VersionPatch = binary.BigEndian.Uint32(data[8:12])

	nodeNameLen := data[12]
	data = data[13:]
	if len(data) < int(nodeNameLen) {
		return errors.New("data too short")
	}
	h.NodeName = string(data[:nodeNameLen])
	data = data[nodeNameLen:]
	if len(data) < 12 {
		return errors.New("data too short")
	}
	h.NodeNonce = binary.BigEndian.Uint64(data[:8])
	declAddrLen := binary.BigEndian.Uint32(data[8:12])
	data = data[12:]
	if len(data) < int(declAddrLen) {
		return errors.New("data too short")
	}
	h.DeclaredAddrBytes = append([]byte(nil), data[:declAddrLen]...)
	data = data[declAddrLen:]
	if len(data) < 8 {
		return errors.New("data too short")
	}
	h.Timestamp = binary.BigEndian.Uint64(data[:8])

	return nil
}
