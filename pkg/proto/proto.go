package proto

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/collect_writes"
)

const (
	MaxHeaderLength = 17
	headerMagic     = 0x12345678
	headerCsumLen   = 4

	HeaderSizeWithPayload    = 17
	HeaderSizeWithoutPayload = 13
)

// Constants for message IDs
const (
	ContentIDGetPeers          = 0x1
	ContentIDPeers             = 0x2
	ContentIDGetSignatures     = 0x14
	ContentIDSignatures        = 0x15
	ContentIDGetBlock          = 0x16
	ContentIDBlock             = 0x17
	ContentIDScore             = 0x18
	ContentIDTransaction       = 0x19
	ContentIDInvMicroblock     = 0x1A
	ContentIDCheckpoint        = 0x64
	ContentIDMicroblockRequest = 27
	ContentIDMicroblock        = 28

	HeaderContentIDPosition = 8
)

type Message interface {
	io.ReaderFrom
	io.WriterTo
	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}

type Header struct {
	Length        uint32
	Magic         uint32
	ContentID     uint8
	PayloadLength uint32
	PayloadCsum   [headerCsumLen]byte
}

func (h *Header) MarshalBinary() ([]byte, error) {
	data := make([]byte, h.HeaderLength())
	h.Copy(data)
	return data, nil
}

func (h *Header) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 17)
	n := h.Copy(buf)
	rs, err := w.Write(buf[:n])
	if err != nil {
		return 0, err
	}
	return int64(rs), nil
}

func (h *Header) HeaderLength() uint32 {
	if h.PayloadLength > 0 {
		return HeaderSizeWithPayload
	}
	return HeaderSizeWithoutPayload
}

func (h *Header) ReadFrom(r io.Reader) (int64, error) {
	body := [HeaderSizeWithPayload]byte{}
	n, err := io.ReadFull(r, body[:HeaderSizeWithoutPayload])
	if err != nil {
		return int64(n), err
	}

	payloadLength := binary.BigEndian.Uint32(body[9:13])
	nn := 0
	if payloadLength > 0 {
		nn, err = io.ReadFull(r, body[HeaderSizeWithoutPayload:HeaderSizeWithPayload])
		if err != nil {
			return int64(n), err
		}
		return int64(n + nn), h.UnmarshalBinary(body[:])
	}

	return int64(n + nn), h.UnmarshalBinary(body[:HeaderSizeWithoutPayload])
}

func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) < HeaderSizeWithoutPayload {
		return fmt.Errorf("data is to short to unmarshal Header: len=%d", len(data))
	}
	h.Length = binary.BigEndian.Uint32(data[0:4])
	h.Magic = binary.BigEndian.Uint32(data[4:8])
	if h.Magic != headerMagic {
		return fmt.Errorf("received wrong magic: want %x, have %x", headerMagic, h.Magic)
	}
	h.ContentID = data[8]
	h.PayloadLength = binary.BigEndian.Uint32(data[9:13])
	if h.PayloadLength > 0 {
		copy(h.PayloadCsum[:], data[13:17])
	}

	return nil
}

func (h *Header) Copy(data []byte) int {
	binary.BigEndian.PutUint32(data[0:4], h.Length)
	binary.BigEndian.PutUint32(data[4:8], headerMagic)
	data[8] = h.ContentID
	binary.BigEndian.PutUint32(data[9:13], h.PayloadLength)
	if h.PayloadLength > 0 {
		copy(data[13:17], h.PayloadCsum[:])
		return HeaderSizeWithPayload
	}
	return HeaderSizeWithoutPayload
}

// Version represents the version of the protocol
type Version struct {
	Major, Minor, Patch uint32
}

func NewVersionFromString(version string) (*Version, error) {
	parts := strings.Split(version, ".")
	if l := len(parts); l <= 0 || l > 3 {
		return nil, errors.Errorf("invalid version string '%s'", version)
	}
	r := &Version{}
	for n, p := range parts {
		i, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid version string '%s'", version)
		}
		switch n {
		case 0:
			r.Major = uint32(i)
		case 1:
			r.Minor = uint32(i)
		case 2:
			r.Patch = uint32(i)
		}
	}
	return r, nil
}

func (a Version) WriteTo(writer io.Writer) (int64, error) {
	b := make([]byte, 12)
	binary.BigEndian.PutUint32(b[:4], a.Major)
	binary.BigEndian.PutUint32(b[4:8], a.Minor)
	binary.BigEndian.PutUint32(b[8:], a.Patch)
	n, err := writer.Write(b)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func (a *Version) ReadFrom(r io.Reader) (int64, error) {
	b := make([]byte, 12)
	n, err := r.Read(b)
	if err != nil {
		return int64(n), err
	}
	a.Major = binary.BigEndian.Uint32(b[0:4])
	a.Minor = binary.BigEndian.Uint32(b[4:8])
	a.Patch = binary.BigEndian.Uint32(b[8:12])
	return int64(n), nil
}

func (a Version) String() string {
	sb := strings.Builder{}
	sb.WriteString(strconv.Itoa(int(a.Major)))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(a.Minor)))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(a.Patch)))
	return sb.String()
}

func (a *Version) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(a.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

type ByVersion []Version

func (a ByVersion) Len() int {
	return len(a)
}

func (a ByVersion) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByVersion) Less(i, j int) bool {
	cmp := func(a, b uint32) int {
		if a < b {
			return -1
		} else if a == b {
			return 0
		} else {
			return 1
		}
	}
	x := cmp(a[i].Major, a[j].Major)
	y := cmp(a[i].Minor, a[j].Minor)
	z := cmp(a[i].Patch, a[j].Patch)
	if x < 0 {
		return true
	} else if x == 0 {
		if y < 0 {
			return true
		} else if y == 0 {
			return z < 0
		} else {
			return false
		}
	} else {
		return false
	}
}

type TCPAddr net.TCPAddr

func NewTCPAddr(ip net.IP, port int) TCPAddr {
	return TCPAddr{
		IP:   ip,
		Port: port,
	}
}

func (a TCPAddr) String() string {
	return fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
}

func (a TCPAddr) Empty() bool {
	return len(a.IP) == 0 || a.IP.IsUnspecified()
}

func (a TCPAddr) WriteTo(w io.Writer) (int64, error) {
	b := []byte(a.IP.To16())
	n, err := w.Write(b)
	if err != nil {
		return int64(n), err
	}

	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, uint64(a.Port))
	n2, err := w.Write(b8)
	if err != nil {
		return 0, err
	}

	return int64(n + n2), nil
}

func NewTCPAddrFromString(s string) TCPAddr {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return TCPAddr{}
	}
	ip := net.ParseIP(host)
	p, err := strconv.ParseUint(port, 10, 64)
	if err != nil {
		return TCPAddr{}
	}
	return NewTCPAddr(ip, int(p))
}

func (a TCPAddr) ToIpPort() IpPort {
	return NewIpPortFromTcpAddr(a)
}

// Handshake is the handshake structure of the waves protocol
type Handshake struct {
	AppName      string
	Version      Version
	NodeName     string
	NodeNonce    uint64
	DeclaredAddr HandshakeTCPAddr
	Timestamp    uint64
}

type HandshakeTCPAddr TCPAddr

func NewHandshakeTCPAddr(ip net.IP, port int) HandshakeTCPAddr {
	return HandshakeTCPAddr{
		IP:   ip,
		Port: port,
	}
}

func (a HandshakeTCPAddr) Empty() bool {
	return TCPAddr(a).Empty()
}

func (a HandshakeTCPAddr) WriteTo(w io.Writer) (int64, error) {
	if a.Empty() {
		n, err := w.Write([]byte{0, 0, 0, 0})
		if err != nil {
			return 0, err
		}
		return int64(n), nil
	}

	b := [12]byte{}
	binary.BigEndian.PutUint32(b[:4], 8)
	copy(b[4:8], a.IP.To4())
	binary.BigEndian.PutUint32(b[8:12], uint32(a.Port))
	n, err := w.Write(b[:])
	if err != nil {
		return 0, err
	}
	return int64(n), nil
}

func (a *HandshakeTCPAddr) ReadFrom(r io.Reader) (int64, error) {
	size := [4]byte{}
	n, err := r.Read(size[:])
	if err != nil {
		return int64(n), err
	}
	s := binary.BigEndian.Uint32(size[:])
	if s > 8 {
		return 0, errors.Errorf("tcp addr is too large: expected size to be 8 or lower, found %d", s)
	}

	if s == 0 {
		return int64(n), nil
	}

	b := [4]byte{}
	n2, err := r.Read(b[:])
	if err != nil {
		return 0, err
	}
	a.IP = net.IPv4(b[0], b[1], b[2], b[3])

	n3, err := r.Read(b[:])
	if err != nil {
		return 0, err
	}
	port := binary.BigEndian.Uint32(b[:])
	a.Port = int(port)

	return int64(n + n2 + n3), nil
}

func (a HandshakeTCPAddr) ToIpPort() IpPort {
	return NewIpPortFromTcpAddr(TCPAddr(a))
}

func (a HandshakeTCPAddr) String() string {
	return TCPAddr(a).String()
}

func (a HandshakeTCPAddr) Network() string {
	return "tcp"
}

func ParseHandshakeTCPAddr(s string) HandshakeTCPAddr {
	return HandshakeTCPAddr(NewTCPAddrFromString(s))
}

type U8String struct {
	S string
}

func NewU8String(s string) U8String {
	return U8String{S: s}
}

// MarshalBinary encodes U8String to binary form
func (a U8String) MarshalBinary() ([]byte, error) {
	l := len(a.S)
	if l > 255 {
		return nil, errors.New("too long string")
	}

	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], a.S)
	return data, nil
}

// MarshalBinary encodes U8String to binary form
func (a U8String) WriteTo(w io.Writer) (int64, error) {
	l := len(a.S)
	if l > 255 {
		return 0, errors.New("too long string")
	}

	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], a.S)
	n, err := w.Write(data)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func (a *U8String) ReadFrom(r io.Reader) (int64, error) {
	size := make([]byte, 1)
	n, err := r.Read(size)
	if err != nil {
		return 0, err
	}
	str := make([]byte, size[0])
	n2, err := r.Read(str)
	if err != nil {
		return 0, err
	}
	a.S = string(str)
	return int64(n + n2), nil
}

type U64 uint64

func (a U64) WriteTo(w io.Writer) (int64, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(a))
	n, err := w.Write(b)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func (a *U64) ReadFrom(r io.Reader) (int64, error) {
	b := [8]byte{}
	n, err := r.Read(b[:])
	if err != nil {
		return int64(n), err
	}
	*a = U64(binary.BigEndian.Uint64(b[:]))
	return int64(n), nil
}

type U32 uint32

func (a U32) WriteTo(w io.Writer) (int64, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(a))
	n, err := w.Write(b)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func (a *U32) ReadFrom(r io.Reader) (int64, error) {
	b := [4]byte{}
	n, err := r.Read(b[:])
	if err != nil {
		return int64(n), err
	}
	*a = U32(binary.BigEndian.Uint32(b[:]))
	return int64(n), nil
}

func (a *Handshake) WriteTo(w io.Writer) (int64, error) {
	c := collect_writes.CollectInt64{}
	c.W(NewU8String(a.AppName).WriteTo(w))
	c.W(a.Version.WriteTo(w))
	c.W(NewU8String(a.NodeName).WriteTo(w))
	c.W(U64(a.NodeNonce).WriteTo(w))
	c.W(a.DeclaredAddr.WriteTo(w))
	c.W(U64(a.Timestamp).WriteTo(w))
	return c.Ret()
}

/* TODO: unused code, need to write tests if it is needed or otherwise remove it.
func (h *Handshake) readApplicationName(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[0:1])
	if err != nil {
		return 0, err
	}

	length := uint(buf[0])
	n2, err := io.ReadFull(r, buf[1:1+length])
	if err != nil {
		return 0, err
	}

	return n + n2, nil
}

func (h *Handshake) readVersion(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[:12])
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (h *Handshake) readNodeName(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[0:1])
	if err != nil {
		return 0, err
	}

	length := uint(buf[0])
	n2, err := io.ReadFull(r, buf[1:1+length])
	if err != nil {
		return 0, err
	}

	return n + n2, nil
}

func (h *Handshake) readNonce(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[:8])
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (h *Handshake) readDeclAddr(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[:4])
	if err != nil {
		return n, err
	}

	addrlen := binary.BigEndian.Uint32(buf[:4])
	if addrlen > 8 {
		return n, errors.Errorf("invalid declared address length, expected 0 or 8, got %d", addrlen)
	}

	if addrlen == 0 {
		return n, nil
	}

	n2, err := io.ReadFull(r, buf[4:4+addrlen])
	if err != nil {
		return n + n2, err
	}

	return n + n2, nil
}

func (h *Handshake) readTimestamp(buf []byte, r io.Reader) (int, error) {
	n, err := io.ReadFull(r, buf[:8])
	if err != nil {
		return n, err
	}

	return n, nil
}
*/

// ReadFrom reads Handshake from io.Reader
func (a *Handshake) ReadFrom(r io.Reader) (int64, error) {
	// max Header size based on fields
	//buf := [556]byte{}
	appName := U8String{}
	n1, err := appName.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "appname")
	}
	a.AppName = appName.S

	n2, err := a.Version.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "version")
	}

	nodeName := U8String{}
	n3, err := nodeName.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "nodename")
	}
	a.NodeName = nodeName.S

	nonce := U64(0)
	n4, err := nonce.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "nonce")
	}
	a.NodeNonce = uint64(nonce)

	addr := HandshakeTCPAddr{}
	n5, err := addr.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "can't read HandshakeTCPAddr")
	}
	a.DeclaredAddr = addr

	tm := U64(0)
	n6, err := tm.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "timestamp")
	}

	a.Timestamp = uint64(tm)
	return int64(n1 + n2 + n3 + n4 + n5 + n6), nil
}

// GetPeersMessage implements the GetPeers message from the waves protocol
type GetPeersMessage struct{}

// MarshalBinary encodes GetPeersMessage to binary form
func (m *GetPeersMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = MaxHeaderLength - 8
	h.Magic = headerMagic
	h.ContentID = ContentIDGetPeers
	h.PayloadLength = 0
	return h.MarshalBinary()
}

// UnmarshalBinary decodes GetPeersMessage from binary form
func (m *GetPeersMessage) UnmarshalBinary(b []byte) error {
	var header Header

	err := header.UnmarshalBinary(b)
	if err != nil {
		return err
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
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}
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

const IpPortLength = net.IPv6len + 8

type IpPort [IpPortLength]byte

func NewIpPortFromTcpAddr(a TCPAddr) IpPort {
	out := IpPort{}
	buf := new(bytes.Buffer)
	_, _ = a.WriteTo(buf)
	copy(out[:], buf.Bytes())
	return out
}

func (a IpPort) Addr() net.IP {
	return net.IP(a[:net.IPv6len])
}

func (a IpPort) Port() int {
	b := binary.BigEndian.Uint64(a[net.IPv6len : net.IPv6len+8])
	return int(b)
}

func (a IpPort) ToTcpAddr() TCPAddr {
	return NewTCPAddr(a.Addr(), a.Port())
}

func (a *IpPort) UnmarshalBinary(b []byte) error {
	if len(b) < IpPortLength {
		return errors.Errorf("too low bytes to unmarshal IpPort, expected at least %d, got %d", IpPortLength, len(b))
	}

	k := IpPort{}
	copy(k[:], b)
	return nil
}

func (a *IpPort) String() string {
	return NewTCPAddr(a.Addr(), a.Port()).String()
}

// PeerInfo represents the address of a single peer
type PeerInfo struct {
	Addr net.IP
	Port uint16
}

func (a PeerInfo) WriteTo(w io.Writer) (int64, error) {
	b := [8]byte{}
	copy(b[:4], a.Addr.To4())
	binary.BigEndian.PutUint32(b[4:8], uint32(a.Port))
	n, err := w.Write(b[:])
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func (a *PeerInfo) ReadFrom(r io.Reader) (int64, error) {
	b := [8]byte{}
	n, err := r.Read(b[:])
	if err != nil {
		return int64(n), err
	}
	a.Addr = net.IPv4(b[0], b[1], b[2], b[3])
	a.Port = uint16(binary.BigEndian.Uint32(b[:4]))

	return int64(n), nil
}

func NewPeerInfoFromString(addr string) (PeerInfo, error) {
	strs := strings.Split(addr, ":")
	if len(strs) != 2 {
		return PeerInfo{}, errors.Errorf("invalid addr %s", addr)
	}

	ip := net.ParseIP(string(strs[0]))
	port, err := strconv.ParseUint(strs[1], 10, 64)
	if err != nil {
		return PeerInfo{}, errors.Errorf("invalid port %s", strs[1])
	}
	return PeerInfo{
		Addr: ip,
		Port: uint16(port),
	}, nil
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

// String() implements Stringer interface for PeerInfo
func (m PeerInfo) String() string {
	var sb strings.Builder
	sb.WriteString(m.Addr.String())
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(m.Port)))

	return sb.String()
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
	if s == jsonNull {
		return nil
	}

	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
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
		return errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
	}
	m.Port = uint16(port64)
	return nil
}

func (m *PeerInfo) Empty() bool {
	if m.Addr == nil || m.Addr.String() == "0.0.0.0" {
		return true
	}

	if m.Port == 0 {
		return true
	}

	return false
}

// PeersMessage represents the peers message
type PeersMessage struct {
	Peers []PeerInfo
}

func (m *PeersMessage) WriteTo(w io.Writer) (int64, error) {
	var h Header

	buf := new(bytes.Buffer)

	c := collect_writes.CollectInt64{}
	length := U32(len(m.Peers))
	c.W(length.WriteTo(buf))

	for _, k := range m.Peers {
		c.W(k.WriteTo(buf))
	}

	n, err := c.Ret()
	if err != nil {
		return n, err
	}

	h.Length = MaxHeaderLength + uint32(len(buf.Bytes())) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDPeers
	h.PayloadLength = uint32(len(buf.Bytes()))
	dig, err := crypto.FastHash(buf.Bytes())
	if err != nil {
		return 0, err
	}
	copy(h.PayloadCsum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return 0, err
	}

	out := append(hdr, buf.Bytes()...)

	n2, err := w.Write(out)
	if err != nil {
		return 0, err
	}

	return int64(n2), nil
}

// MarshalBinary encodes PeersMessage message to binary form
func (m *PeersMessage) MarshalBinary() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, err := m.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

// UnmarshalBinary decodes PeersMessage from binary form
func (m *PeersMessage) UnmarshalBinary(data []byte) error {
	var header Header
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	data = data[MaxHeaderLength:]
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
	l := binary.BigEndian.Uint32(packetLen[:])
	packet := make([]byte, l)
	for i := 0; i < len(packet); i++ {
		packet[i] = 0x88
	}
	n, err := io.ReadFull(r, packet)
	if err != nil {
		return nil, int64(nn + n), err
	}
	nn += n
	packet = append(packetLen[:], packet...)

	return packet, int64(nn), nil
}

func ReadPacket(buf []byte, r io.Reader) (int64, error) {
	packetLen := buf[:4]
	nn, err := io.ReadFull(r, packetLen)
	if err != nil {
		return int64(nn), err
	}
	l := binary.BigEndian.Uint32(packetLen)
	buf = buf[4:]
	packet := buf[:l]
	n, err := io.ReadFull(r, packet)
	if err != nil {
		return int64(nn + n), err
	}
	nn += n
	return int64(nn), nil
}

func ReadPayload(buf []byte, r io.Reader) (int64, error) {
	nn, err := io.ReadFull(r, buf)
	if err != nil {
		return int64(nn), err
	}
	return int64(nn), nil
}

// ReadFrom reads PeersMessage from io.Reader
func (m *PeersMessage) ReadFrom(r io.Reader) (int64, error) {
	//packet, nn, err := readPacket(r)
	//if err != nil {
	//	return nn, err
	//}

	h := Header{}
	n, err := h.ReadFrom(r)
	if err != nil {
		return n, err
	}

	length := U32(0)
	n2, err := length.ReadFrom(r)
	if err != nil {
		return 0, err
	}

	Peers := make([]PeerInfo, length)

	n3 := n + n2
	for i := 0; i < int(length); i++ {
		p := PeerInfo{}
		n4, err := p.ReadFrom(r)
		if err != nil {
			return 0, err
		}
		n3 += n4
		Peers[i] = p
	}

	return n3, nil
}

// GetSignaturesMessage represents the Get Signatures request
type GetSignaturesMessage struct {
	Blocks []crypto.Signature
}

// MarshalBinary encodes GetSignaturesMessage to binary form
func (m *GetSignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Blocks)*64)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, b := range m.Blocks {
		body = append(body, b[:]...)
	}

	var h Header
	h.Length = MaxHeaderLength + uint32(len(body)) - 4
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
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDGetSignatures {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	blockCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]

	for i := uint32(0); i < blockCount; i++ {
		var b crypto.Signature
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
	Signatures []crypto.Signature
}

// MarshalBinary encodes SignaturesMessage to binary form
func (m *SignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures))
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	var h Header
	h.Length = MaxHeaderLength + uint32(len(body)) - 4
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
	var h Header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDSignatures {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	sigCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]

	for i := uint32(0); i < sigCount; i++ {
		var sig crypto.Signature
		offset := i * 64
		if len(data[offset:]) < 64 {
			return fmt.Errorf("message too short: %v", len(data))
		}
		copy(sig[:], data[offset:offset+64])
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
	BlockID crypto.Signature
}

// MarshalBinary encodes GetBlockMessage to binary form
func (m *GetBlockMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 0, 64)
	body = append(body, m.BlockID[:]...)

	var h Header
	h.Length = MaxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetBlock
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

// UnmarshalBinary decodes GetBlockMessage from binary form
func (m *GetBlockMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}

	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDGetBlock {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
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
	var h Header
	h.Length = MaxHeaderLength + uint32(len(m.BlockBytes)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDBlock
	h.PayloadLength = uint32(len(m.BlockBytes))
	dig, err := crypto.FastHash(m.BlockBytes)
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

func MakeHeader(contentID uint8, payload []byte) (Header, error) {
	var h Header
	h.Length = MaxHeaderLength + uint32(len(payload)) - 4
	h.Magic = headerMagic
	h.ContentID = contentID
	h.PayloadLength = uint32(len(payload))
	dig, err := crypto.FastHash(payload)
	if err != nil {
		return Header{}, err
	}
	copy(h.PayloadCsum[:], dig[:4])
	return h, nil
}

// UnmarshalBinary decodes BlockMessage from binary from
func (m *BlockMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDBlock {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}

	m.BlockBytes = make([]byte, h.PayloadLength)
	copy(m.BlockBytes, data[17:17+h.PayloadLength])

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
	var h Header
	h.Length = MaxHeaderLength + uint32(len(m.Score)) - 4
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
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDScore {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}

	m.Score = make([]byte, h.PayloadLength)
	copy(m.Score, data[17:17+h.PayloadLength])
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

// TransactionMessage represents TransactionsSend message
type TransactionMessage struct {
	Transaction []byte
}

// MarshalBinary encodes TransactionMessage to binary form
func (m *TransactionMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = MaxHeaderLength + uint32(len(m.Transaction)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDTransaction
	h.PayloadLength = uint32(len(m.Transaction))
	dig, err := crypto.FastHash(m.Transaction)
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
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDTransaction {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	// TODO check max length
	m.Transaction = make([]byte, h.PayloadLength)
	copy(m.Transaction, data[MaxHeaderLength:MaxHeaderLength+h.PayloadLength])
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return err
	}

	if !bytes.Equal(dig[:4], h.PayloadCsum[:]) {
		return fmt.Errorf("invalid checksum: expected %x, found %x", dig[:4], h.PayloadCsum[:])
	}
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
	Signature crypto.Signature
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

	var h Header
	h.Length = MaxHeaderLength + uint32(len(body)) - 4
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
	var h Header
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

// UnmarshalMessage tries unmarshal bytes to proper type
func UnmarshalMessage(b []byte) (Message, error) {
	if len(b) < HeaderSizeWithoutPayload {
		return nil, errors.Errorf("message is too short")
	}

	var m Message
	switch b[HeaderContentIDPosition] {
	case ContentIDGetPeers:
		m = &GetPeersMessage{}
	case ContentIDPeers:
		m = &PeersMessage{}
	case ContentIDGetSignatures:
		m = &GetSignaturesMessage{}
	case ContentIDSignatures:
		m = &SignaturesMessage{}
	case ContentIDGetBlock:
		m = &GetBlockMessage{}
	case ContentIDBlock:
		m = &BlockMessage{}
	case ContentIDScore:
		m = &ScoreMessage{}
	case ContentIDTransaction:
		m = &TransactionMessage{}
	case ContentIDCheckpoint:
		m = &CheckPointMessage{}
	case ContentIDMicroblock:
		m = &MicroBlockMessage{}
	case ContentIDMicroblockRequest:
		m = &MicroBlockRequestMessage{}
	case ContentIDInvMicroblock:
		m = &MicroBlockInvMessage{}
	default:
		return nil, errors.Errorf(
			"received unknown content id byte %d 0x%x", b[HeaderContentIDPosition], b[HeaderContentIDPosition])
	}

	err := m.UnmarshalBinary(b)
	return m, err

}

type BulkMessage []Message

func (BulkMessage) ReadFrom(r io.Reader) (n int64, err error) {
	panic("implement me")
}

func (BulkMessage) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (BulkMessage) UnmarshalBinary(data []byte) error {
	panic("implement me")
}

func (a BulkMessage) MarshalBinary() (data []byte, err error) {
	var out bytes.Buffer
	for _, row := range a {
		_, err := row.WriteTo(&out)
		if err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}
