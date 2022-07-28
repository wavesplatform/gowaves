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
	HeaderContentIDPosition = 8

	headerSizeWithPayload    = 17
	headerSizeWithoutPayload = 13
	maxHeaderLength          = headerSizeWithPayload
	headerMagic              = 0x12345678
	headerChecksumLen        = 4
)

type (
	PeerMessageID  byte
	PeerMessageIDs []PeerMessageID
)

// Constants for message IDs
const (
	ContentIDGetPeers          PeerMessageID = 0x1
	ContentIDPeers             PeerMessageID = 0x2
	ContentIDGetSignatures     PeerMessageID = 0x14
	ContentIDSignatures        PeerMessageID = 0x15
	ContentIDGetBlock          PeerMessageID = 0x16
	ContentIDBlock             PeerMessageID = 0x17
	ContentIDScore             PeerMessageID = 0x18
	ContentIDTransaction       PeerMessageID = 0x19
	ContentIDInvMicroblock     PeerMessageID = 0x1A
	ContentIDCheckpoint        PeerMessageID = 0x64
	ContentIDMicroblockRequest PeerMessageID = 27
	ContentIDMicroblock        PeerMessageID = 28
	ContentIDPBBlock           PeerMessageID = 29
	ContentIDPBMicroBlock      PeerMessageID = 30
	ContentIDPBTransaction     PeerMessageID = 31
	ContentIDGetBlockIds       PeerMessageID = 32
	ContentIDBlockIds          PeerMessageID = 33
)

var ProtocolVersion = NewVersion(1, 4, 0)

type Message interface {
	io.ReaderFrom
	io.WriterTo
	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}

type Header struct {
	Length          uint32
	Magic           uint32
	ContentID       PeerMessageID
	PayloadLength   uint32
	PayloadChecksum [headerChecksumLen]byte
}

func (h *Header) MarshalBinary() ([]byte, error) {
	data := make([]byte, h.HeaderLength())
	if _, err := h.Copy(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (h *Header) WriteTo(w io.Writer) (int64, error) {
	buf := [headerSizeWithPayload]byte{}
	n, err := h.Copy(buf[:])
	if err != nil {
		return 0, err
	}
	rs, err := w.Write(buf[:n])
	return int64(rs), err
}

func (h *Header) HeaderLength() uint32 {
	if h.PayloadLength > 0 {
		return headerSizeWithPayload
	}
	return headerSizeWithoutPayload
}

func (h *Header) ReadFrom(r io.Reader) (int64, error) {
	body := [headerSizeWithPayload]byte{}
	n, err := io.ReadFull(r, body[:headerSizeWithoutPayload])
	if err != nil {
		return int64(n), err
	}

	payloadLength := binary.BigEndian.Uint32(body[9:headerSizeWithoutPayload])
	nn := 0
	if payloadLength > 0 {
		nn, err = io.ReadFull(r, body[headerSizeWithoutPayload:headerSizeWithPayload])
		if err != nil {
			return int64(n), err
		}
		return int64(n + nn), h.UnmarshalBinary(body[:])
	}

	return int64(n + nn), h.UnmarshalBinary(body[:headerSizeWithoutPayload])
}

func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) < headerSizeWithoutPayload {
		return fmt.Errorf("data is to short to unmarshal Header: len=%d", len(data))
	}
	h.Length = binary.BigEndian.Uint32(data[0:4])
	h.Magic = binary.BigEndian.Uint32(data[4:8])
	if h.Magic != headerMagic {
		return fmt.Errorf("received wrong magic: want %x, have %x", headerMagic, h.Magic)
	}
	h.ContentID = PeerMessageID(data[HeaderContentIDPosition])
	h.PayloadLength = binary.BigEndian.Uint32(data[9:headerSizeWithoutPayload])
	if h.PayloadLength > 0 {
		if uint32(len(data)) < headerSizeWithPayload {
			return errors.New("Header UnmarshalBinary: invalid data size")
		}
		copy(h.PayloadChecksum[:], data[headerSizeWithoutPayload:headerSizeWithPayload])
	}

	return nil
}

func (h *Header) Copy(data []byte) (int, error) {
	if len(data) < headerSizeWithoutPayload {
		return 0, errors.New("Header Copy: invalid data size")
	}
	binary.BigEndian.PutUint32(data[0:4], h.Length)
	binary.BigEndian.PutUint32(data[4:8], headerMagic)
	data[HeaderContentIDPosition] = byte(h.ContentID)
	binary.BigEndian.PutUint32(data[9:headerSizeWithoutPayload], h.PayloadLength)
	if h.PayloadLength > 0 {
		if len(data) < headerSizeWithPayload {
			return 0, errors.New("Header Copy: invalid data size")
		}
		copy(data[headerSizeWithoutPayload:headerSizeWithPayload], h.PayloadChecksum[:])
		return headerSizeWithPayload, nil
	}
	return headerSizeWithoutPayload, nil
}

// Version represents the version of the protocol
type Version struct {
	_                   struct{} // this field disallows raw struct initialization
	major, minor, patch uint32
}

func NewVersion(major, minor, patch uint32) Version {
	return Version{
		major: major,
		minor: minor,
		patch: patch,
	}
}

func (v Version) Major() uint32 {
	return v.major
}

func (v Version) Minor() uint32 {
	return v.minor
}

func (v Version) Patch() uint32 {
	return v.patch
}

func (v Version) Cmp(other Version) int {
	if v.major < other.major {
		return -1
	}
	if v.major > other.major {
		return 1
	}
	if v.minor < other.minor {
		return -1
	}
	if v.minor > other.minor {
		return 1
	}
	if v.patch < other.patch {
		return -1
	}
	if v.patch > other.patch {
		return 1
	}
	return 0
}

// CmpMinor compares minor version.
// If equal return 0.
// If diff only 1 version (for example 1.14 and 1.13), then 1
// If more then 1 version, then return 2.
func (v Version) CmpMinor(other Version) int {
	if v.major != other.major {
		return 2
	}
	if v.minor == other.minor {
		return 0
	}
	rs := v.minor - other.minor
	if rs*rs == 1 {
		return 1
	}
	return 2
}

func NewVersionFromString(version string) (Version, error) {
	parts := strings.Split(version, ".")
	if l := len(parts); l <= 0 || l > 3 {
		return Version{}, errors.Errorf("invalid version string '%s'", version)
	}
	r := Version{}
	for n, p := range parts {
		i, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return Version{}, errors.Wrapf(err, "invalid version string '%s'", version)
		}
		switch n {
		case 0:
			r.major = uint32(i)
		case 1:
			r.minor = uint32(i)
		case 2:
			r.patch = uint32(i)
		}
	}
	return r, nil
}

func (v Version) WriteTo(writer io.Writer) (int64, error) {
	b := [12]byte{}
	binary.BigEndian.PutUint32(b[:4], v.major)
	binary.BigEndian.PutUint32(b[4:8], v.minor)
	binary.BigEndian.PutUint32(b[8:], v.patch)
	n, err := writer.Write(b[:])
	return int64(n), err
}

func (v *Version) ReadFrom(r io.Reader) (int64, error) {
	b := [12]byte{}
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}
	v.major = binary.BigEndian.Uint32(b[0:4])
	v.minor = binary.BigEndian.Uint32(b[4:8])
	v.patch = binary.BigEndian.Uint32(b[8:12])
	return int64(n), nil
}

func (v Version) String() string {
	sb := strings.Builder{}
	sb.WriteString(strconv.Itoa(int(v.major)))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(v.minor)))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(v.patch)))
	return sb.String()
}

func (v Version) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(v.String())
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
	x := cmp(a[i].major, a[j].major)
	y := cmp(a[i].minor, a[j].minor)
	z := cmp(a[i].patch, a[j].patch)
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
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}

func (a TCPAddr) Empty() bool {
	return len(a.IP) == 0 || a.IP.IsUnspecified()
}

func (a TCPAddr) WriteTo(w io.Writer) (int64, error) {
	b := []byte(a.IP.To16())
	n1, err := w.Write(b)
	if err != nil {
		return int64(n1), err
	}
	b8 := [8]byte{}
	binary.BigEndian.PutUint64(b8[:], uint64(a.Port))
	n2, err := w.Write(b8[:])
	return int64(n1 + n2), err
}

// ToUint64 converts TCPAddr to uint64 number.
func (a TCPAddr) ToUint64() uint64 {
	ip := uint64(a.ipToUint32()) << 32
	ip = ip | uint64(a.Port)
	return ip
}

func (a TCPAddr) ipToUint32() uint32 {
	if len(a.IP) == 16 {
		return binary.BigEndian.Uint32(a.IP[12:16])
	}
	return binary.BigEndian.Uint32(a.IP)
}

// Equal checks if ip address and port are equal.
func (a TCPAddr) Equal(other TCPAddr) bool {
	return a.IP.Equal(other.IP) && a.Port == other.Port
}

func NewTCPAddrFromString(s string) TCPAddr {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return TCPAddr{}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err == nil {
			ip = ips[0]
		}
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return TCPAddr{}
	}
	return NewTCPAddr(ip, int(p))
}

func NewTcpAddrFromUint64(value uint64) TCPAddr {
	var (
		ip    = make([]byte, 4)
		port  = uint32(value)
		ipVal = uint32(value >> 32)
	)
	binary.BigEndian.PutUint32(ip, ipVal)
	return TCPAddr{
		IP:   ip,
		Port: int(port),
	}
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
	n, err := io.ReadFull(r, size[:])
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
	n2, err := io.ReadFull(r, b[:])
	if err != nil {
		return 0, err
	}
	a.IP = net.IPv4(b[0], b[1], b[2], b[3])

	n3, err := io.ReadFull(r, b[:])
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

// WriteTo writes U8String into io.Writer w in binary form.
func (a U8String) WriteTo(w io.Writer) (int64, error) {
	l := len(a.S)
	if l > 255 {
		return 0, errors.New("too long string")
	}

	data := make([]byte, l+1)
	data[0] = byte(l)
	copy(data[1:1+l], a.S)
	n, err := w.Write(data)
	return int64(n), err
}

func (a *U8String) ReadFrom(r io.Reader) (int64, error) {
	size := [1]byte{}
	n1, err := io.ReadFull(r, size[:])
	if err != nil {
		return int64(n1), err
	}
	str := make([]byte, size[0])
	n2, err := io.ReadFull(r, str)
	if err != nil {
		return int64(n1 + n2), err
	}
	a.S = string(str)
	return int64(n1 + n2), nil
}

type U64 uint64

func (a U64) WriteTo(w io.Writer) (int64, error) {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(a))
	n, err := w.Write(b[:])
	return int64(n), err
}

func (a *U64) ReadFrom(r io.Reader) (int64, error) {
	b := [8]byte{}
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}
	*a = U64(binary.BigEndian.Uint64(b[:]))
	return int64(n), nil
}

type U32 uint32

func (a U32) WriteTo(w io.Writer) (int64, error) {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], uint32(a))
	n, err := w.Write(b[:])
	return int64(n), err
}

func (a *U32) ReadFrom(r io.Reader) (int64, error) {
	b := [4]byte{}
	n, err := io.ReadFull(r, b[:])
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

// ReadFrom reads Handshake from io.Reader
func (a *Handshake) ReadFrom(r io.Reader) (int64, error) {
	appName := U8String{}
	n1, err := appName.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "appName")
	}
	a.AppName = appName.S

	n2, err := a.Version.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "version")
	}

	nodeName := U8String{}
	n3, err := nodeName.ReadFrom(r)
	if err != nil {
		return 0, errors.Wrap(err, "nodeName")
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
	return n1 + n2 + n3 + n4 + n5 + n6, nil
}

// GetPeersMessage implements the GetPeers message from the waves protocol
type GetPeersMessage struct{}

// MarshalBinary encodes GetPeersMessage to binary form
func (m *GetPeersMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = maxHeaderLength - 8
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
	return nn, m.UnmarshalBinary(packet)
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
	return a[:net.IPv6len]
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

func NewPeerInfoFromString(addr string) (PeerInfo, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return PeerInfo{}, errors.Errorf("invalid addr %s", addr)
	}

	ip := net.ParseIP(parts[0])
	port, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return PeerInfo{}, errors.Errorf("invalid port %s", parts[1])
	}
	return PeerInfo{
		Addr: ip,
		Port: uint16(port),
	}, nil
}

func (p PeerInfo) WriteTo(w io.Writer) (int64, error) {
	b := [8]byte{}
	copy(b[:4], p.Addr.To4())
	binary.BigEndian.PutUint32(b[4:8], uint32(p.Port))
	n, err := w.Write(b[:])
	return int64(n), err
}

func (p *PeerInfo) ReadFrom(r io.Reader) (int64, error) {
	b := [8]byte{}
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}
	p.Addr = net.IPv4(b[0], b[1], b[2], b[3])
	p.Port = uint16(binary.BigEndian.Uint32(b[:4]))

	return int64(n), nil
}

// MarshalBinary encodes PeerInfo message to binary form
func (p *PeerInfo) MarshalBinary() ([]byte, error) {
	buffer := make([]byte, 8)

	copy(buffer[0:4], p.Addr.To4())
	binary.BigEndian.PutUint32(buffer[4:8], uint32(p.Port))

	return buffer, nil
}

// UnmarshalBinary decodes PeerInfo message from binary form
func (p *PeerInfo) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("too short")
	}

	p.Addr = net.IPv4(data[0], data[1], data[2], data[3])
	p.Port = uint16(binary.BigEndian.Uint32(data[4:8]))

	return nil
}

// String() implements Stringer interface for PeerInfo
func (p PeerInfo) String() string {
	var sb strings.Builder
	sb.WriteString(p.Addr.String())
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(p.Port)))

	return sb.String()
}

// MarshalJSON writes PeerInfo Value as JSON string
func (p PeerInfo) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	if p.Addr == nil {
		return nil, errors.New("invalid addr")
	}
	if p.Port == 0 {
		return nil, errors.New("invalid port")
	}
	sb.WriteRune('"')
	sb.WriteString(p.Addr.String())
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(p.Port)))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads PeerInfo from JSON string
func (p *PeerInfo) UnmarshalJSON(value []byte) error {
	s := string(value)
	if s == jsonNull {
		return nil
	}

	s, err := strconv.Unquote(s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 1 {
		s = parts[0]
	} else {
		s = parts[1]
	}

	parts = strings.SplitN(s, ":", 2)
	var addr, port string
	if len(parts) == 1 {
		addr = parts[0]
		port = "0"
	} else {
		addr = parts[0]
		port = parts[1]
	}

	p.Addr = net.ParseIP(addr)
	port64, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerInfo from JSON")
	}
	p.Port = uint16(port64)
	return nil
}

func (p *PeerInfo) Empty() bool {
	if p.Addr == nil || p.Addr.String() == "0.0.0.0" {
		return true
	}

	if p.Port == 0 {
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

	peers := m.Peers

	if len(peers) > 1000 {
		peers = peers[:1000]
	}

	length := U32(len(peers))
	c.W(length.WriteTo(buf))

	for _, k := range peers {
		c.W(k.WriteTo(buf))
	}

	n, err := c.Ret()
	if err != nil {
		return n, err
	}

	h.Length = maxHeaderLength + uint32(len(buf.Bytes())) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDPeers
	h.PayloadLength = uint32(len(buf.Bytes()))
	dig, err := crypto.FastHash(buf.Bytes())
	if err != nil {
		return 0, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

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
	if uint32(len(data)) < maxHeaderLength {
		return errors.New("PeersMessage UnmarshalBinary: invalid data size")
	}
	data = data[maxHeaderLength:]
	if len(data) < 4 {
		return errors.New("peers message has insufficient length")
	}
	peersCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	for i := uint32(0); i < peersCount; i++ {
		var peer PeerInfo
		if uint32(len(data)) < 8 {
			return errors.Errorf("PeersMessage UnmarshalBinary: invalid peers count: expected %d, found %d", peersCount, len(m.Peers))
		}
		if err := peer.UnmarshalBinary(data[:8]); err != nil {
			return err
		}
		m.Peers = append(m.Peers, peer)
		data = data[8:]
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
	Signatures []crypto.Signature
}

// MarshalBinary encodes GetSignaturesMessage to binary form
func (m *GetSignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures)*64)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	var h Header
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetSignatures
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

// UnmarshalBinary decodes GetSignaturesMessage from binary form
func (m *GetSignaturesMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return errors.New("GetSignaturesMessage UnmarshalBinary: invalid data size")
	}
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

	pos := 0
	for i := uint32(0); i < blockCount; i++ {
		var b crypto.Signature
		if len(data[pos:]) < 64 {
			return fmt.Errorf("message too short %v", len(data))
		}
		copy(b[:], data[pos:pos+64])
		m.Signatures = append(m.Signatures, b)
		pos += 64
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
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDSignatures
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

// UnmarshalBinary decodes SignaturesMessage from binary form
func (m *SignaturesMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return errors.New("SignaturesMessage UnmarshalBinary: invalid data size")
	}
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
	BlockID BlockID
}

// MarshalBinary encodes GetBlockMessage to binary form
func (m *GetBlockMessage) MarshalBinary() ([]byte, error) {
	body := m.BlockID.Bytes()

	var h Header
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetBlock
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

// UnmarshalBinary decodes GetBlockMessage from binary form
func (m *GetBlockMessage) UnmarshalBinary(data []byte) error {
	return parsePacket(data, ContentIDGetBlock, "GetBlockMessage", func(payload []byte) error {
		blockID, err := NewBlockIDFromBytes(payload)
		if err != nil {
			return err
		}
		m.BlockID = blockID
		return nil
	})
}

func parsePacket(data []byte, ContentID PeerMessageID, name string, f func(payload []byte) error) error {
	if len(data) < 17 {
		return errors.Errorf("%s: invalid data size %d, expected at least 17", name, len(data))
	}
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return errors.Wrap(err, name)
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("%s: wrong magic in Header: %x", name, h.Magic)
	}
	if h.ContentID != ContentID {
		return fmt.Errorf("%s: wrong ContentID in Header: %x", name, h.ContentID)
	}
	if len(data) < int(17+h.PayloadLength) {
		return fmt.Errorf("%s: expected data at least %d, found %d", name, 17+h.PayloadLength, len(data))
	}
	err := f(data[17 : 17+h.PayloadLength])
	if err != nil {
		return errors.Wrapf(err, "%s payload error", name)
	}
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

func MessageByBlock(block *Block, scheme Scheme) (Message, error) {
	bts, err := block.Marshal(scheme)
	if err != nil {
		return nil, err
	}
	if block.Version >= ProtobufBlockVersion {
		return &PBBlockMessage{bts}, nil
	} else {
		return &BlockMessage{bts}, nil
	}
}

// BlockMessage represents Block message
type BlockMessage struct {
	BlockBytes []byte
}

// MarshalBinary encodes BlockMessage to binary form
func (m *BlockMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = maxHeaderLength + uint32(len(m.BlockBytes)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDBlock
	h.PayloadLength = uint32(len(m.BlockBytes))
	dig, err := crypto.FastHash(m.BlockBytes)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.BlockBytes...)
	return hdr, nil
}

func MakeHeader(contentID PeerMessageID, payload []byte) (Header, error) {
	var h Header
	h.Length = maxHeaderLength + uint32(len(payload)) - 4
	h.Magic = headerMagic
	h.ContentID = contentID
	h.PayloadLength = uint32(len(payload))
	dig, err := crypto.FastHash(payload)
	if err != nil {
		return Header{}, err
	}
	copy(h.PayloadChecksum[:], dig[:4])
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

	if uint32(len(data)) < 17+h.PayloadLength {
		return errors.New("BlockMessage UnmarshalBinary: invalid data size")
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
	h.Length = maxHeaderLength + uint32(len(m.Score)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDScore
	h.PayloadLength = uint32(len(m.Score))
	dig, err := crypto.FastHash(m.Score)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

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

	if uint32(len(data)) < 17+h.PayloadLength {
		return errors.New("invalid data size")
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
	h.Length = maxHeaderLength + uint32(len(m.Transaction)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDTransaction
	h.PayloadLength = uint32(len(m.Transaction))
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

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
	if uint32(len(data)) < maxHeaderLength+h.PayloadLength {
		return errors.New("invalid data size")
	}
	m.Transaction = make([]byte, h.PayloadLength)
	copy(m.Transaction, data[maxHeaderLength:maxHeaderLength+h.PayloadLength])
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return err
	}

	if !bytes.Equal(dig[:4], h.PayloadChecksum[:]) {
		return fmt.Errorf("invalid checksum: expected %x, found %x", dig[:4], h.PayloadChecksum[:])
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
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDCheckpoint
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	hdr = append(hdr, body...)
	return hdr, nil

}

// UnmarshalBinary decodes CheckPointMessage from binary form
func (m *CheckPointMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return errors.New("invalid data size")
	}
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

// PBBlockMessage represents Protobuf Block message
type PBBlockMessage struct {
	PBBlockBytes []byte
}

// MarshalBinary encodes PBBlockMessage to binary form
func (m *PBBlockMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = maxHeaderLength + uint32(len(m.PBBlockBytes)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDPBBlock
	h.PayloadLength = uint32(len(m.PBBlockBytes))
	dig, err := crypto.FastHash(m.PBBlockBytes)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.PBBlockBytes...)
	return hdr, nil
}

// UnmarshalBinary decodes PBBlockMessage from binary from
func (m *PBBlockMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDPBBlock {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}

	m.PBBlockBytes = make([]byte, h.PayloadLength)
	if uint32(len(data)) < 17+h.PayloadLength {
		return errors.New("PBBlockMessage UnmarshalBinary: invalid data size")
	}
	copy(m.PBBlockBytes, data[17:17+h.PayloadLength])

	return nil
}

// ReadFrom reads PBBlockMessage from io.Reader
func (m *PBBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes PBBlockMessage to io.Writer
func (m *PBBlockMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// PBTransactionMessage represents Protobuf TransactionsSend message
type PBTransactionMessage struct {
	Transaction []byte
}

// MarshalBinary encodes PBTransactionMessage to binary form
func (m *PBTransactionMessage) MarshalBinary() ([]byte, error) {
	var h Header
	h.Length = maxHeaderLength + uint32(len(m.Transaction)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDPBTransaction
	h.PayloadLength = uint32(len(m.Transaction))
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.Transaction...)
	return hdr, nil
}

// UnmarshalBinary decodes PBTransactionMessage from binary form
func (m *PBTransactionMessage) UnmarshalBinary(data []byte) error {
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.ContentID != ContentIDPBTransaction {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	// TODO check max length
	m.Transaction = make([]byte, h.PayloadLength)
	if uint32(len(data)) < maxHeaderLength+h.PayloadLength {
		return errors.New("PBTransactionMessage UnmarshalBinary: invalid data size")
	}
	copy(m.Transaction, data[maxHeaderLength:maxHeaderLength+h.PayloadLength])
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return err
	}

	if !bytes.Equal(dig[:4], h.PayloadChecksum[:]) {
		return fmt.Errorf("invalid checksum: expected %x, found %x", dig[:4], h.PayloadChecksum[:])
	}
	return nil
}

// ReadFrom reads PBTransactionMessage from io.Reader
func (m *PBTransactionMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}
	return nn, m.UnmarshalBinary(packet)
}

// WriteTo writes PBTransactionMessage to io.Writer
func (m *PBTransactionMessage) WriteTo(w io.Writer) (int64, error) {
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
	if len(b) < headerSizeWithoutPayload {
		return nil, errors.Errorf("message is too short")
	}

	var m Message
	switch messageID := b[HeaderContentIDPosition]; PeerMessageID(messageID) {
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
	case ContentIDPBBlock:
		m = &PBBlockMessage{}
	case ContentIDPBMicroBlock:
		m = &PBMicroBlockMessage{}
	case ContentIDPBTransaction:
		m = &PBTransactionMessage{}
	case ContentIDGetBlockIds:
		m = &GetBlockIdsMessage{}
	case ContentIDBlockIds:
		m = &BlockIdsMessage{}
	default:
		return nil, errors.Errorf(
			"received unknown content id byte %d 0x%x", b[HeaderContentIDPosition], b[HeaderContentIDPosition])
	}

	err := m.UnmarshalBinary(b)
	return m, err

}

// GetBlockIdsMessage is used for Signatures or hashes block ids
type GetBlockIdsMessage struct {
	Blocks []BlockID
}

func (m *GetBlockIdsMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, bl := range m.Blocks {
		b := bl.Bytes()
		idLen := len(b)
		body = append(body, byte(idLen))
		body = append(body, b...)
	}

	var h Header
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDGetBlockIds
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *GetBlockIdsMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return errors.New("GetBlockIdsMessage UnmarshalBinary: invalid data size")
	}
	var h Header
	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDGetBlockIds {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	blockCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	pos := 0
	for i := uint32(0); i < blockCount; i++ {
		if len(data) < pos+1 {
			return fmt.Errorf("message too short %v", len(data))
		}
		idLen := int(data[pos])
		pos += 1
		if len(data[pos:]) < idLen {
			return fmt.Errorf("message too short %v", len(data))
		}
		id, err := NewBlockIDFromBytes(data[pos : pos+idLen])
		if err != nil {
			return errors.Wrap(err, "bad block id bytes")
		}
		m.Blocks = append(m.Blocks, id)
		pos += idLen
	}

	return nil
}

func (m *GetBlockIdsMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

func (m *GetBlockIdsMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

// BlockIdsMessage is used for Signatures or hashes block ids.
type BlockIdsMessage struct {
	Blocks []BlockID
}

func (m *BlockIdsMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, bl := range m.Blocks {
		b := bl.Bytes()
		idLen := len(b)
		body = append(body, byte(idLen))
		body = append(body, b...)
	}

	var h Header
	h.Length = maxHeaderLength + uint32(len(body)) - 4
	h.Magic = headerMagic
	h.ContentID = ContentIDBlockIds
	h.PayloadLength = uint32(len(body))
	dig, err := crypto.FastHash(body)
	if err != nil {
		return nil, err
	}
	copy(h.PayloadChecksum[:], dig[:4])

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *BlockIdsMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return errors.New("BlockIdsMessage UnmarshalBinary: invalid data size")
	}
	var h Header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDBlockIds {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	data = data[17:]
	if len(data) < 4 {
		return fmt.Errorf("message too short %v", len(data))
	}
	idsCount := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]

	offset := 0
	for i := uint32(0); i < idsCount; i++ {
		if len(data) < offset+1 {
			return fmt.Errorf("message too short: %v", len(data))
		}
		idLen := int(data[offset])
		offset += 1
		if len(data[offset:]) < idLen {
			return fmt.Errorf("message too short: %v", len(data))
		}
		id, err := NewBlockIDFromBytes(data[offset : offset+idLen])
		if err != nil {
			return errors.Wrap(err, "bad block id bytes")
		}
		m.Blocks = append(m.Blocks, id)
		offset += idLen
	}

	return nil
}

func (m *BlockIdsMessage) ReadFrom(r io.Reader) (int64, error) {
	packet, nn, err := readPacket(r)
	if err != nil {
		return nn, err
	}

	return nn, m.UnmarshalBinary(packet)
}

func (m *BlockIdsMessage) WriteTo(w io.Writer) (int64, error) {
	buf, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	nn, err := w.Write(buf)
	n := int64(nn)
	return n, err
}

type BulkMessage []Message

func (BulkMessage) ReadFrom(_ io.Reader) (n int64, err error) {
	panic("implement me")
}

func (BulkMessage) WriteTo(_ io.Writer) (n int64, err error) {
	panic("implement me")
}

func (BulkMessage) UnmarshalBinary(_ []byte) error {
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

type MiningLimits struct {
	MaxScriptRunsInBlock        int
	MaxScriptsComplexityInBlock int
	ClassicAmountOfTxsInBlock   int
	MaxTxsSizeInBytes           int
}
