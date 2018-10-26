package proto

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"io"
	"net"
	"strconv"
	"strings"
)

const (
	headerLength  = 17
	headerMagic   = 0x12345678
	headerCsumLen = 4
)

// Constants for message IDs
const (
	ContentIDGetPeers      = 0x1
	ContentIDPeers         = 0x2
	ContentIDGetSignatures = 0x14
	ContentIDSignatures    = 0x15
	ContentIDGetBlock      = 0x16
	ContentIDBlock         = 0x17
	ContentIDScore         = 0x18
	ContentIDTransaction   = 0x19
	ContentIDCheckpoint    = 0x64
)

// BlockSignature is a signature of a formed block
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

// Version represents the version of the protocol
type Version struct {
	Major, Minor, Patch uint32
}

// Handshake is the handshake structure of the waves protocol
type Handshake struct {
	Name              string
	Version           Version
	NodeName          string
	NodeNonce         uint64
	DeclaredAddrBytes []byte
	Timestamp         uint64
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

	binary.BigEndian.PutUint32(data[0:4], h.Version.Major)
	binary.BigEndian.PutUint32(data[4:8], h.Version.Minor)
	binary.BigEndian.PutUint32(data[8:12], h.Version.Patch)

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

// MarshalBinary encodes Handshake to binary form
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

// UnmarshalBinary decodes Handshake from binary from
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
	h.Version.Major = binary.BigEndian.Uint32(data[0:4])
	h.Version.Minor = binary.BigEndian.Uint32(data[4:8])
	h.Version.Patch = binary.BigEndian.Uint32(data[8:12])

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

// ReadFrom reads Handshake from io.Reader
func (h *Handshake) ReadFrom(r io.Reader) (int64, error) {
	buf := make([]byte, 1)

	nn, err := io.ReadFull(r, buf)
	if err != nil {
		return int64(nn), err
	}

	buf = append(buf, make([]byte, uint(buf[0]))...)
	n, err := io.ReadFull(r, buf[1:])
	if err != nil {
		return int64(n + nn), err
	}
	nn += n
	tmp := make([]byte, 13)
	n, err = io.ReadFull(r, tmp)
	if err != nil {
		return int64(n + nn), err
	}
	buf = append(buf, tmp...)
	nn += n
	tmp = make([]byte, uint(tmp[12]))
	n, err = io.ReadFull(r, tmp)
	if err != nil {
		return int64(n + nn), err
	}
	buf = append(buf, tmp...)
	nn += n
	tmp = make([]byte, 12)
	n, err = io.ReadFull(r, tmp)
	if err != nil {
		return int64(n + nn), err
	}
	buf = append(buf, tmp...)
	nn += n
	addrlen := binary.BigEndian.Uint32(tmp[8:12])
	buf = append(buf, tmp...)
	tmp = make([]byte, addrlen+8)
	n, err = io.ReadFull(r, tmp)
	if err != nil {
		return int64(n + nn), err
	}
	buf = append(buf, tmp...)
	nn += n

	return int64(nn), h.UnmarshalBinary(buf)
}

// WriteTo writes Handshake to io.Writer
func (h *Handshake) WriteTo(w io.Writer) (int64, error) {
	buf, err := h.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// GetPeersMessage implements the GetPeers message from the waves protocol
type GetPeersMessage struct{}

// MarshalBinary encodes GetPeersMessage to binary form
func (m *GetPeersMessage) MarshalBinary() ([]byte, error) {
	var header header

	header.Length = headerLength - 8
	header.Magic = headerMagic
	header.ContentID = ContentIDGetPeers
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

// UnmarshalBinary decodes GetPeersMessage from binary form
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
	if header.ContentID != ContentIDGetPeers {
		return fmt.Errorf("getpeers message ContentID is unexpected: want %x have %x", ContentIDGetPeers, header.ContentID)
	}
	if header.PayloadLength != 0 {
		return fmt.Errorf("getpeers message length is not zero")
	}

	return nil
}

// ReadFrom reads GetPeersMessage from io.Reader
func (m *GetPeersMessage) ReadFrom(r io.Reader) (int64, error) {
	var packetLen [4]byte
	nn, err := io.ReadFull(r, packetLen[:])
	if err != nil {
		return int64(nn), err
	}
	packet := make([]byte, binary.BigEndian.Uint32(packetLen[:]))
	n, err := io.ReadFull(r, packet)
	if err != nil {
		return int64(nn), err
	}
	nn += n
	packet = append(packetLen[:], packet...)

	return int64(nn), m.UnmarshalBinary(packet)
}

// WriteTo writes GetPeersMessage to io.Writer
func (m *GetPeersMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// PeerInfo represents the address of a single peer
type PeerInfo struct {
	Addr net.IP
	Port uint16
}

// MarshalBinary encodes PeerInfo message to binary form
func (m *PeerInfo) MarshalBinary() ([]byte, error) {
	buffer := make([]byte, 8)

	copy(buffer[0:4], m.Addr.To4())
	binary.BigEndian.PutUint32(buffer[4:8], uint32(m.Port))

	return buffer, nil
}

// UnmarshalBinary decodes PeerInfo message from binary form
func (m *PeerInfo) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("too short")
	}

	m.Addr = net.IPv4(data[0], data[1], data[2], data[3])
	m.Port = uint16(binary.BigEndian.Uint32(data[4:8]))

	return nil
}

// MarshalJSON writes PeerInfo Value as JSON string
func (m PeerInfo) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	if m.Addr == nil {
		return nil, errors.New("invalid addr")
	}
	if m.Port == 0 {
		return nil, errors.New("invalid port")
	}
	sb.WriteRune('"')
	sb.WriteString(m.Addr.String())
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(m.Port)))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads PeerInfo from JSON string
func (m *PeerInfo) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == "null" {
		return nil
	}

	s, err := strconv.Unquote(s)
	if err != nil {
		errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
	}

	splitted := strings.SplitN(s, "/", 2)
	if len(splitted) == 1 {
		s = splitted[0]
	} else {
		s = splitted[1]
	}

	splitted = strings.SplitN(s, ":", 2)
	var addr, port string
	if len(splitted) == 1 {
		addr = splitted[0]
		port = "0"
	} else {
		addr = splitted[0]
		port = splitted[1]
	}

	m.Addr = net.ParseIP(addr)
	port64, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
	}
	m.Port = uint16(port64)
	return nil
}

// PeersMessage represents the peers message
type PeersMessage struct {
	Peers []PeerInfo
}

// MarshalBinary encodes PeersMessage message to binary form
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
	h.ContentID = ContentIDPeers
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

// UnmarshalBinary decodes PeersMessage from binary form
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

func readPacket(r io.Reader) ([]byte, int64, error) {
	var packetLen [4]byte
	nn, err := io.ReadFull(r, packetLen[:])
	if err != nil {
		return nil, int64(nn), err
	}
	packet := make([]byte, binary.BigEndian.Uint32(packetLen[:]))
	n, err := io.ReadFull(r, packet)
	if err != nil {
		return nil, int64(nn + n), err
	}
	nn += n
	packet = append(packetLen[:], packet...)

	return packet, int64(nn), nil
}

// ReadFrom reads PeersMessage from io.Reader
func (m *PeersMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes PeersMessage to io.Writer
func (m *PeersMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// BlockID represents the ID of a block
type BlockID [64]byte

// GetSignaturesMessage represents the Get Signatures request
type GetSignaturesMessage struct {
	Blocks []BlockID
}

// MarshalBinary encodes GetSignaturesMessage to binary form
func (m *GetSignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Blocks)*64)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, b := range m.Blocks {
		body = append(body, b[:]...)
	}

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetSignatures
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

// UnmarshalBinary decodes GetSignaturesMessage from binary form
func (m *GetSignaturesMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDGetSignatures {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
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

// ReadFrom reads GetSignaturesMessage from io.Reader
func (m *GetSignaturesMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes GetSignaturesMessage to io.Writer
func (m *GetSignaturesMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// SignaturesMessage represents Signatures message
type SignaturesMessage struct {
	Signatures []BlockSignature
}

// MarshalBinary encodes SignaturesMessage to binary form
func (m *SignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures))
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDSignatures
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

// UnmarshalBinary decodes SignaturesMessage from binary form
func (m *SignaturesMessage) UnmarshalBinary(data []byte) error {
	var h header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDSignatures {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
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

// ReadFrom reads SignaturesMessage from binary form
func (m *SignaturesMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes SignaturesMessage to binary form
func (m *SignaturesMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// GetBlockMessage represents GetBlock message
type GetBlockMessage struct {
	BlockID BlockID
}

// MarshalBinary encodes GetBlockMessage to binary form
func (m *GetBlockMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 0, 64)
	body = append(body, m.BlockID[:]...)

	var h header
	h.Length = headerLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetBlock
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

// UnmarshalBinary decodes GetBlockMessage from binary form
func (m *GetBlockMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}

	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDGetBlock {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 64 {
		return fmt.Errorf("message too short %v", len(data))
	}

	copy(m.BlockID[:], data[:64])

	return nil
}

// ReadFrom reads GetBlockMessage from io.Reader
func (m *GetBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes GetBlockMessage to io.Writer
func (m *GetBlockMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// BlockMessage represents Block message
type BlockMessage struct {
	BlockBytes []byte
}

// MarshalBinary encodes BlockMessage to binary form
func (m *BlockMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.BlockBytes)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDBlock
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

// UnmarshalBinary decodes BlockMessage from binary from
func (m *BlockMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDBlock {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
	}

	m.BlockBytes = data[17:]

	return nil
}

// ReadFrom reads BlockMessage from io.Reader
func (m *BlockMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes BlockMessage to io.Writer
func (m *BlockMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// ScoreMessage represents Score message
type ScoreMessage struct {
	Score []byte
}

// MarshalBinary encodes ScoreMessage to binary form
func (m *ScoreMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.Score)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDScore
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

// UnmarshalBinary decodes ScoreMessage from binary form
func (m *ScoreMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDScore {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
	}

	m.Score = data[17:]

	return nil
}

// ReadFrom reads ScoreMessage from io.Reader
func (m *ScoreMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return 0, err
	}
	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes ScoreMessage to io.Writer
func (m *ScoreMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// TransactionMessage represents Transaction message
type TransactionMessage struct {
	Transaction []byte
}

// MarshalBinary encodes TransactionMessage to binary form
func (m *TransactionMessage) MarshalBinary() ([]byte, error) {
	var h header
	h.Length = headerLength + uint32(len(m.Transaction)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDTransaction
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

// UnmarshalBinary decodes TransactionMessage from binary form
func (m *TransactionMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in header: %x", h.Magic)
	}
	if h.ContentID != ContentIDTransaction {
		return fmt.Errorf("wrong ContentID in header: %x", h.ContentID)
	}

	m.Transaction = data[17:]

	return nil
}

// ReadFrom reads TransactionMessage from io.Reader
func (m *TransactionMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}
	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes TransactionMessage to io.Writer
func (m *TransactionMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// CheckpointItem represents a Checkpoint
type CheckpointItem struct {
	Height    uint64
	Signature BlockSignature
}

// CheckPointMessage represents a CheckPoint message
type CheckPointMessage struct {
	Checkpoints []CheckpointItem
}

// MarshalBinary encodes CheckPointMessage to binary form
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
	h.ContentID = ContentIDCheckpoint
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

// UnmarshalBinary decodes CheckPointMessage from binary form
func (m *CheckPointMessage) UnmarshalBinary(data []byte) error {
	var h header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("ckeckpoint message magic is unexpected: %x", headerMagic)
	}
	if h.ContentID != ContentIDCheckpoint {
		return fmt.Errorf("checkpoint message ContentID is unexpected %x", h.ContentID)
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

// ReadFrom reads CheckPointMessage from io.Reader
func (m *CheckPointMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes CheckPointMessage to io.Writer
func (m *CheckPointMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}
