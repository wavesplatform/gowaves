package proto

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"

	"github.com/ccoveille/go-safecast"
	"golang.org/x/crypto/blake2b"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/collect_writes"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

/*
Messages are sent over the network in the following format:
+---------+-----------+----------+-------------+------------------+---------+
| MSG_LEN | MSG_MAGIC | MSG_TYPE | PAYLOAD_LEN | PAYLOAD_CHECKSUM | PAYLOAD |
+---------+-----------+----------+-------------+------------------+---------+

* MSG_LEN (4 bytes, uint32) - message length. It includes lengths of all fields except the length of MSG_LEN itself.
* MSG_MAGIC (4 bytes, "0x12345678") - magic number, constant value.
* MSG_TYPE (1 byte) - message type.
* PAYLOAD_LEN (4 bytes, uin32) - payload length.
* PAYLOAD_CHECKSUM (4 bytes) - payload checksum, optional, may be omitted if PAYLOAD_LEN == 0.
* PAYLOAD (variable) - payload, optional, omitted if PAYLOAD_LEN == 0.

Payload checksum calculated as first 4 bytes of blake2b-256 (crypto.FastHash) digest of payload.
*/

const (
	HeaderContentIDPosition = 8

	msgLenSize          uint32 = 4
	msgMagicSize        uint32 = 4
	msgTypeSize         uint32 = 1
	payloadLenSize      uint32 = 4
	payloadChecksumSize uint32 = 4

	headerSizeWithPayload    = msgLenSize + msgMagicSize + msgTypeSize + payloadLenSize + payloadChecksumSize
	headerSizeWithoutPayload = msgLenSize + msgMagicSize + msgTypeSize + payloadLenSize
	maxHeaderLength          = headerSizeWithPayload
	headerMagic              = 0x12345678
)

type (
	PeerMessageID  byte
	PeerMessageIDs []PeerMessageID
)

// Constants for message IDs
const (
	ContentIDGetPeers                  PeerMessageID = 0x1
	ContentIDPeers                     PeerMessageID = 0x2
	ContentIDGetSignatures             PeerMessageID = 0x14
	ContentIDSignatures                PeerMessageID = 0x15
	ContentIDGetBlock                  PeerMessageID = 0x16
	ContentIDBlock                     PeerMessageID = 0x17
	ContentIDScore                     PeerMessageID = 0x18
	ContentIDTransaction               PeerMessageID = 0x19
	ContentIDInvMicroblock             PeerMessageID = 0x1A
	ContentIDMicroblockRequest         PeerMessageID = 27
	ContentIDMicroblock                PeerMessageID = 28
	ContentIDPBBlock                   PeerMessageID = 29
	ContentIDPBMicroBlock              PeerMessageID = 30
	ContentIDPBTransaction             PeerMessageID = 31
	ContentIDGetBlockIDs               PeerMessageID = 32
	ContentIDBlockIDs                  PeerMessageID = 33
	ContentIDGetBlockSnapshot          PeerMessageID = 34
	ContentIDMicroBlockSnapshotRequest PeerMessageID = 35
	ContentIDBlockSnapshot             PeerMessageID = 36
	ContentIDMicroBlockSnapshot        PeerMessageID = 37
)

func ProtocolVersion() Version {
	const major, minor, patch = 1, 5, 0
	return NewVersion(major, minor, patch)
}

// ParseMessage parses a message from the given data. The 'f' function parameter is used to parse the message payload.
func ParseMessage(data []byte, contentID PeerMessageID, name string, f func(payload []byte) error) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if l < headerSizeWithoutPayload {
		return fmt.Errorf("%s: invalid data size %d, expected at least %d",
			name, len(data), headerSizeWithoutPayload)
	}
	var h Header
	if ubErr := h.UnmarshalBinary(data); ubErr != nil {
		return fmt.Errorf("%s: %w", name, ubErr)
	}
	if vErr := h.Validate(contentID); vErr != nil {
		return fmt.Errorf("%s: %w", name, vErr)
	}
	if exp, act := int(h.Length+msgLenSize), len(data); act < exp { // Add the length of the MSG_LEN field itself.
		return fmt.Errorf("%s: expected data at least %d, found %d", name, exp, act)
	}
	if h.payloadLength > 0 {
		if fErr := f(data[headerSizeWithPayload : headerSizeWithPayload+h.payloadLength]); fErr != nil {
			return fmt.Errorf("%s: payload error: %w", name, fErr)
		}
	}
	return nil
}

type ChecksumReader struct {
	r io.Reader
	h hash.Hash
}

func NewChecksumReader(r io.Reader) (*ChecksumReader, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChecksumReader: %w", err)
	}
	return &ChecksumReader{
		r: r,
		h: h,
	}, nil
}

// Read reads data from the underlying reader and updates the checksum.
func (r *ChecksumReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		if wn, wErr := r.h.Write(p[:n]); wErr != nil {
			return wn, fmt.Errorf("failed to update hasher: %w", wErr)
		}
	}
	return n, err
}

// Checksum returns the checksum of the data read so far.
func (r *ChecksumReader) Checksum() [payloadChecksumSize]byte {
	var d crypto.Digest
	r.h.Sum(d[:0])
	var cs [4]byte
	copy(cs[:], d[:4])
	return cs
}

// ReadMessage reads message from io.Reader and parses its payload using the given [f] function.
// While reading payload, checksum is calculated and compared with the checksum from the message header.
func ReadMessage(r io.Reader, contentID PeerMessageID, name string, payload Payload) (int64, error) {
	var h Header
	n1, err := h.ReadFrom(r)
	if err != nil {
		return n1, fmt.Errorf("%s: failed to read header: %w", name, err)
	}
	if vErr := h.Validate(contentID); vErr != nil {
		return n1, fmt.Errorf("%s: message header is not valid: %w", name, vErr)
	}
	if h.payloadLength > 0 && payload == nil {
		return n1, fmt.Errorf("%s: empty payload while length is %d", name, h.payloadLength)
	}
	if h.payloadLength == 0 || payload == nil { // Fast exit for messages without payload.
		return n1, nil
	}
	pr, err := NewChecksumReader(io.LimitReader(r, int64(h.payloadLength)))
	if err != nil {
		return n1, fmt.Errorf("%s: failed to create checksum reader: %w", name, err)
	}
	n2, err := payload.ReadFrom(pr)
	if err != nil {
		return n1 + n2, fmt.Errorf("%s: failed to read payload: %w", name, err)
	}
	if pr.Checksum() != h.PayloadChecksum {
		return n1 + n2, fmt.Errorf("%s: payload checksum mismatch", name)
	}
	return n1 + n2, nil
}

func writeEmptyMessage(w io.Writer, contentID PeerMessageID, name string) (int64, error) {
	h, err := NewHeader(contentID, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create header: %w", name, err)
	}
	n, err := h.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("%s: failed to write header: %w", name, err)
	}
	return n, nil
}

// WriteMessage writes a message with the given content ID, name, and payload to the writer.
func WriteMessage(w io.Writer, contentID PeerMessageID, name string, payload io.WriterTo) (int64, error) {
	// TODO: Think about implementing a MessageWriter that does sequential payload write and
	//  header calculation (sizes and checksum). Looks like it has to update some of header fields after payload write.
	//  Don't know if it's possible.
	if payload == nil {
		return writeEmptyMessage(w, contentID, name)
	}
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	if _, err := payload.WriteTo(buf); err != nil {
		return 0, fmt.Errorf("%s: failed to write payload: %w", name, err)
	}
	h, err := NewHeader(contentID, buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("%s: failed to create header: %w", name, err)
	}
	n1, err := h.WriteTo(w)
	if err != nil {
		return n1, fmt.Errorf("%s: failed to write header: %w", name, err)
	}
	n2, err := buf.WriteTo(w)
	if err != nil {
		return n1 + n2, fmt.Errorf("%s: failed to write payload: %w", name, err)
	}
	return n1 + n2, nil
}

type messageTag interface {
	IsMessage()
}
type Message interface {
	messageTag
	io.ReaderFrom
	io.WriterTo
	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
	SetPayload(Payload) (Message, error)
}

type Header struct {
	Length          uint32
	Magic           uint32
	ContentID       PeerMessageID
	payloadLength   uint32
	PayloadChecksum [payloadChecksumSize]byte
}

func NewHeader(contentID PeerMessageID, body []byte) (Header, error) {
	bl, err := safecast.ToUint32(len(body))
	if err != nil {
		return Header{}, fmt.Errorf("failed to create header: %w", err)
	}
	msgLen := msgMagicSize + msgTypeSize + payloadLenSize // For empty Header.
	cs := [payloadChecksumSize]byte{}
	if bl > 0 {
		msgLen = msgMagicSize + msgTypeSize + payloadLenSize + payloadChecksumSize + bl
		dig, fhErr := crypto.FastHash(body)
		if fhErr != nil {
			return Header{}, fmt.Errorf("failed to create header: %w", fhErr)
		}
		copy(cs[:], dig[:payloadChecksumSize])
	}
	return Header{
		Length:          msgLen,
		Magic:           headerMagic,
		ContentID:       contentID,
		payloadLength:   bl,
		PayloadChecksum: cs,
	}, nil
}

// Validate checks the header for correctness. It checks the magic number, ContentID, and lengths.
// Returns an error with a description of the problem.
func (h *Header) Validate(contentID PeerMessageID) error {
	if h.Magic != headerMagic {
		return fmt.Errorf("invalid header: wrong magic: want %x, have %x", headerMagic, h.Magic)
	}
	if h.ContentID != contentID {
		return fmt.Errorf("invalid header: wrong ContentID: want %x, have %x", contentID, h.ContentID)
	}
	// h.Length is the length of the message after the MSG_LEN field itself. So, we need to add 4 bytes to check
	// the total length of the message
	if exp := h.HeaderLength() + h.payloadLength - msgLenSize; h.Length != exp {
		return fmt.Errorf("invalid header: incorrect message length in header (%d), expected  %d", h.Length, exp)
	}
	return nil
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
	if h.payloadLength > 0 {
		return headerSizeWithPayload
	}
	return headerSizeWithoutPayload
}

func (h *Header) ReadFrom(r io.Reader) (int64, error) {
	var msgLen U32
	n1, err := msgLen.ReadFrom(r)
	if err != nil {
		return n1, fmt.Errorf("failed to read message lenght: %w", err)
	}
	h.Length = uint32(msgLen)

	var magic U32
	n2, err := magic.ReadFrom(r)
	if err != nil {
		return n1 + n2, fmt.Errorf("failed to read message magic: %w", err)
	}
	h.Magic = uint32(magic)

	var msgType [msgTypeSize]byte
	n3, err := io.ReadFull(r, msgType[:])
	if err != nil {
		return n1 + n2 + int64(n3), fmt.Errorf("failed to read message type: %w", err)
	}
	h.ContentID = PeerMessageID(msgType[0])

	var payloadLen U32
	n4, err := payloadLen.ReadFrom(r)
	if err != nil {
		return n1 + n2 + int64(n3) + n4, fmt.Errorf("failed to read payload length: %w", err)
	}
	h.payloadLength = uint32(payloadLen)

	if payloadLen == 0 { // Fast exit for messages without payload.
		return n1 + n2 + int64(n3) + n4, nil
	}

	var payloadChecksum [payloadChecksumSize]byte
	n5, err := io.ReadFull(r, payloadChecksum[:])
	if err != nil {
		return n1 + n2 + int64(n3) + n4 + int64(n5), fmt.Errorf("failed to read payload checksum: %w", err)
	}
	h.PayloadChecksum = payloadChecksum
	return n1 + n2 + int64(n3) + n4 + int64(n5), nil
}

func (h *Header) UnmarshalBinary(data []byte) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("failed to unmarshal Header: %w", err)
	}
	if l < headerSizeWithoutPayload {
		return fmt.Errorf("data is to short to unmarshal Header: len=%d", len(data))
	}
	h.Length = binary.BigEndian.Uint32(data[0:4])
	h.Magic = binary.BigEndian.Uint32(data[4:8])
	if h.Magic != headerMagic {
		return fmt.Errorf("received wrong magic: want %x, have %x", headerMagic, h.Magic)
	}
	h.ContentID = PeerMessageID(data[HeaderContentIDPosition])
	h.payloadLength = binary.BigEndian.Uint32(data[9:headerSizeWithoutPayload])
	if h.payloadLength > 0 {
		if uint32(len(data)) < headerSizeWithPayload {
			return errors.New("Header UnmarshalBinary: invalid data size")
		}
		copy(h.PayloadChecksum[:], data[headerSizeWithoutPayload:headerSizeWithPayload])
	}

	return nil
}

func (h *Header) Copy(data []byte) (int, error) {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return 0, fmt.Errorf("failed to copy Header: %w", err)
	}
	if l < headerSizeWithoutPayload {
		return 0, errors.New("failed to copy Header: invalid data size")
	}
	binary.BigEndian.PutUint32(data[:msgLenSize], h.Length)
	binary.BigEndian.PutUint32(data[msgLenSize:msgLenSize+msgMagicSize], h.Magic)
	data[HeaderContentIDPosition] = byte(h.ContentID)
	binary.BigEndian.PutUint32(data[HeaderContentIDPosition+1:headerSizeWithoutPayload], h.payloadLength)
	if h.payloadLength > 0 {
		if l < headerSizeWithPayload {
			return 0, errors.New("failed to copy Header: invalid data size")
		}
		copy(data[headerSizeWithoutPayload:headerSizeWithPayload], h.PayloadChecksum[:])
		return int(headerSizeWithPayload), nil
	}
	return int(headerSizeWithoutPayload), nil
}

func (h *Header) PayloadLength() uint32 {
	return h.payloadLength
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

// NewTCPAddrFromString creates TCPAddr from string.
// Returns empty TCPAddr if string can't be parsed.
func NewTCPAddrFromString(s string) TCPAddr {
	pi, err := NewPeerInfoFromString(s)
	if err != nil {
		return TCPAddr{} // return empty TCPAddr in case of error
	}
	return NewTCPAddr(pi.Addr, int(pi.Port))
}

func (a TCPAddr) String() string {
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}

// Empty checks if IP of TCPAddr is empty or unspecified (e.g., 0.0.0.0 or ::).
func (a TCPAddr) Empty() bool {
	return len(a.IP) == 0 || a.IP.IsUnspecified()
}

// EmptyNoPort checks if IP of TCPAddr is empty AND port is 0.
func (a TCPAddr) EmptyNoPort() bool {
	return a.Empty() && a.Port == 0
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
// Deprecated: will be removed in future versions.
func (a TCPAddr) ToUint64() uint64 {
	ip := uint64(a.ipToUint32()) << 32
	ip = ip | uint64(a.Port)
	return ip
}

// TODO: remove after removing of ToUint64.
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

// Deprecated: will be removed in future versions.
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
	h.payloadLength = 0
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
	if header.payloadLength != 0 {
		return fmt.Errorf("getpeers message length is not zero")
	}

	return nil
}

// ReadFrom reads GetPeersMessage from io.Reader
func (m *GetPeersMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDGetPeers, "GetPeersMessage", nil)
}

// WriteTo writes GetPeersMessage to io.Writer
func (m *GetPeersMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDGetPeers, "GetPeersMessage", nil)
}

func (m *GetPeersMessage) IsMessage() {}

func (m *GetPeersMessage) SetPayload(Payload) (Message, error) {
	return m, nil
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

func filterToIPV4(ips []net.IP) []net.IP {
	for i := 0; i < len(ips); i++ {
		ipV4 := ips[i].To4()
		if ipV4 == nil { // for now we support only IPv4
			iLast := len(ips) - 1
			ips[i], ips[iLast] = ips[iLast], nil // move last address to the current position, order is not important
			ips = ips[:iLast]                    // remove last address
			i--                                  // move back to check the previously last address
		} else {
			ips[i] = ipV4 // replace with exact IPv4 form (ipV4 can be in both forms: ipv4 and ipV4 in ipv6)
		}
	}
	return ips
}

func resolveHostToIPsv4(host string) ([]net.IP, error) {
	if host == "" {
		host = "0.0.0.0" // set default host to 0.0.0.0
	}
	if ip := net.ParseIP(host); ip != nil { // try to parse host as IP address
		ipV4 := ip.To4() // try to convert to IPv4
		if ipV4 == nil {
			return nil, errors.Errorf("non-IPv4 address %q", host)
		}
		return []net.IP{ipV4}, nil // host is already an IP address
	}
	ips, err := net.LookupIP(host) // try to resolve host
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve host %q", host)
	}
	ips = filterToIPV4(ips)
	if len(ips) == 0 {
		return nil, errors.Errorf("no IPv4 addresses found for host %q", host)
	}
	return ips, nil
}

// PeerInfo represents the address of a single peer
type PeerInfo struct {
	Addr net.IP
	Port uint16
}

func ipsV4PortFromString(addr string) ([]net.IP, uint16, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to split host and port")
	}
	portNum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, 0, errors.Errorf("invalid port %q", port)
	}
	if portNum == 0 {
		return nil, 0, errors.Errorf("invalid port %q", port)
	}
	ips, err := resolveHostToIPsv4(host)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to resolve host")
	}
	return ips, uint16(portNum), nil
}

// NewPeerInfosFromString creates PeerInfo slice from string 'host:port'.
// It resolves host to IPv4 addresses and creates PeerInfo for each of them.
func NewPeerInfosFromString(addr string) ([]PeerInfo, error) {
	ips, portNum, err := ipsV4PortFromString(addr)
	if err != nil {
		return nil, err
	}
	res := make([]PeerInfo, 0, len(ips))
	for _, ip := range ips {
		res = append(res, PeerInfo{
			Addr: ip,
			Port: portNum,
		})
	}
	return res, nil
}

// NewPeerInfoFromString creates PeerInfo from string 'host:port'.
// It resolves host to IPv4 addresses and selects the random one using math/rand/v2.
func NewPeerInfoFromString(addr string) (PeerInfo, error) {
	ips, portNum, err := ipsV4PortFromString(addr)
	if err != nil {
		return PeerInfo{}, err
	}
	n := rand.IntN(len(ips)) // #nosec: it's ok to use math/rand/v2 here
	ip := ips[n]             // Select random IPv4 from the list
	return PeerInfo{
		Addr: ip,
		Port: portNum,
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
	p.Port, err = safecast.ToUint16(binary.BigEndian.Uint32(b[4:8]))
	if err != nil {
		return int64(n), fmt.Errorf("PeerInfo: invalid port value: %w", err)
	}
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
	Peers PeerInfos
}

func (m *PeersMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDPeers, "PeersMessage", &m.Peers)
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
	for range peersCount {
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

// ReadFrom reads PeersMessage from io.Reader.
func (m *PeersMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDPeers, "PeersMessage", &m.Peers)
}

func (m *PeersMessage) IsMessage() {}

func (m *PeersMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*PeerInfos); ok {
		m.Peers = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// GetSignaturesMessage represents the Get Signatures request
type GetSignaturesMessage struct {
	Signatures Signatures
}

// MarshalBinary encodes GetSignaturesMessage to binary form
func (m *GetSignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures)*64)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	h, err := NewHeader(ContentIDGetSignatures, body)
	if err != nil {
		return nil, err
	}

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
	for range blockCount {
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
	return ReadMessage(r, ContentIDGetSignatures, "GetSignaturesMessage", &m.Signatures)
}

// WriteTo writes GetSignaturesMessage to io.Writer
func (m *GetSignaturesMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDGetSignatures, "GetSignaturesMessage", &m.Signatures)
}

func (m *GetSignaturesMessage) IsMessage() {}

func (m *GetSignaturesMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*Signatures); ok {
		m.Signatures = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// SignaturesMessage represents Signatures message
type SignaturesMessage struct {
	Signatures Signatures
}

// MarshalBinary encodes SignaturesMessage to binary form
func (m *SignaturesMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4, 4+len(m.Signatures))
	binary.BigEndian.PutUint32(body[0:4], common.SafeIntToUint32(len(m.Signatures)))
	for _, b := range m.Signatures {
		body = append(body, b[:]...)
	}

	h, err := NewHeader(ContentIDSignatures, body)
	if err != nil {
		return nil, err
	}

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

	for i := range sigCount {
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
	return ReadMessage(r, ContentIDSignatures, "SignaturesMessage", &m.Signatures)
}

// WriteTo writes SignaturesMessage to binary form
func (m *SignaturesMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDSignatures, "SignaturesMessage", &m.Signatures)
}

func (m *SignaturesMessage) IsMessage() {}

func (m *SignaturesMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*Signatures); ok {
		m.Signatures = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// GetBlockMessage represents GetBlock message
type GetBlockMessage struct {
	BlockID BlockID
}

// MarshalBinary encodes GetBlockMessage to binary form
func (m *GetBlockMessage) MarshalBinary() ([]byte, error) {
	body := m.BlockID.Bytes()

	h, err := NewHeader(ContentIDGetBlock, body)
	if err != nil {
		return nil, err
	}
	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

// UnmarshalBinary decodes GetBlockMessage from binary form
func (m *GetBlockMessage) UnmarshalBinary(data []byte) error {
	return ParseMessage(data, ContentIDGetBlock, "GetBlockMessage", func(payload []byte) error {
		blockID, err := NewBlockIDFromBytes(payload)
		if err != nil {
			return err
		}
		m.BlockID = blockID
		return nil
	})
}

// ReadFrom reads GetBlockMessage from io.Reader
func (m *GetBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDGetBlock, "GetBlockMessage", &m.BlockID)
}

// WriteTo writes GetBlockMessage to io.Writer
func (m *GetBlockMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDGetBlock, "GetBlockMessage", &m.BlockID)
}
func (m *GetBlockMessage) IsMessage() {}

func (m *GetBlockMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockID); ok {
		m.BlockID = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
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
	BlockBytes BytesPayload
}

// MarshalBinary encodes BlockMessage to binary form
func (m *BlockMessage) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(ContentIDBlock, m.BlockBytes)
	if err != nil {
		return nil, err
	}
	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hdr = append(hdr, m.BlockBytes...)
	return hdr, nil
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

	if common.SafeIntToUint32(len(data)) < headerSizeWithPayload+h.payloadLength {
		return errors.New("BlockMessage UnmarshalBinary: invalid data size")
	}
	m.BlockBytes = make([]byte, h.payloadLength)
	copy(m.BlockBytes, data[headerSizeWithPayload:headerSizeWithPayload+h.payloadLength])

	return nil
}

// ReadFrom reads BlockMessage from io.Reader
func (m *BlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDBlock, "BlockMessage", &m.BlockBytes)
}

// WriteTo writes BlockMessage to io.Writer
func (m *BlockMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDBlock, "BlockMessage", &m.BlockBytes)
}

func (m *BlockMessage) IsMessage() {}

func (m *BlockMessage) SetPayload(payload Payload) (Message, error) {
	if b, ok := payload.(*BytesPayload); ok {
		m.BlockBytes = *b
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// ScoreMessage represents Score message
type ScoreMessage struct {
	Score BytesPayload
}

// MarshalBinary encodes ScoreMessage to binary form
func (m *ScoreMessage) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(ContentIDScore, m.Score)
	if err != nil {
		return nil, err
	}
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

	if common.SafeIntToUint32(len(data)) < 17+h.payloadLength {
		return errors.New("invalid data size")
	}
	m.Score = make([]byte, h.payloadLength)
	copy(m.Score, data[17:17+h.payloadLength])
	return nil
}

// ReadFrom reads ScoreMessage from io.Reader
func (m *ScoreMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDScore, "ScoreMessage", &m.Score)
}

// WriteTo writes ScoreMessage to io.Writer
func (m *ScoreMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDScore, "ScoreMessage", &m.Score)
}

func (m *ScoreMessage) IsMessage() {}

func (m *ScoreMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Score = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// TransactionMessage represents TransactionsSend message
type TransactionMessage struct {
	Transaction BytesPayload
}

// MarshalBinary encodes TransactionMessage to binary form
func (m *TransactionMessage) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(ContentIDTransaction, m.Transaction)
	if err != nil {
		return nil, err
	}
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
	if common.SafeIntToUint32(len(data)) < maxHeaderLength+h.payloadLength {
		return errors.New("invalid data size")
	}
	m.Transaction = make([]byte, h.payloadLength)
	copy(m.Transaction, data[maxHeaderLength:maxHeaderLength+h.payloadLength])
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
	return ReadMessage(r, ContentIDTransaction, "TransactionMessage", &m.Transaction)
}

// WriteTo writes TransactionMessage to io.Writer
func (m *TransactionMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDTransaction, "TransactionMessage", &m.Transaction)
}

func (m *TransactionMessage) IsMessage() {}

func (m *TransactionMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Transaction = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// PBBlockMessage represents Protobuf Block message
type PBBlockMessage struct {
	PBBlockBytes BytesPayload
}

// MarshalBinary encodes PBBlockMessage to binary form
func (m *PBBlockMessage) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(ContentIDPBBlock, m.PBBlockBytes)
	if err != nil {
		return nil, err
	}
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

	m.PBBlockBytes = make([]byte, h.payloadLength)
	if common.SafeIntToUint32(len(data)) < 17+h.payloadLength {
		return errors.New("PBBlockMessage UnmarshalBinary: invalid data size")
	}
	copy(m.PBBlockBytes, data[17:17+h.payloadLength])

	return nil
}

// ReadFrom reads PBBlockMessage from io.Reader
func (m *PBBlockMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDPBBlock, "PBBlockMessage", &m.PBBlockBytes)
}

// WriteTo writes PBBlockMessage to io.Writer
func (m *PBBlockMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDPBBlock, "PBBlockMessage", &m.PBBlockBytes)
}

func (m *PBBlockMessage) IsMessage() {}

func (m *PBBlockMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.PBBlockBytes = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// PBTransactionMessage represents Protobuf TransactionsSend message
type PBTransactionMessage struct {
	Transaction BytesPayload
}

// MarshalBinary encodes PBTransactionMessage to binary form
func (m *PBTransactionMessage) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(ContentIDPBTransaction, m.Transaction)
	if err != nil {
		return nil, err
	}
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
	m.Transaction = make([]byte, h.payloadLength)
	if common.SafeIntToUint32(len(data)) < maxHeaderLength+h.payloadLength {
		return errors.New("PBTransactionMessage UnmarshalBinary: invalid data size")
	}
	copy(m.Transaction, data[maxHeaderLength:maxHeaderLength+h.payloadLength])
	dig, err := crypto.FastHash(m.Transaction)
	if err != nil {
		return err
	}

	if !bytes.Equal(dig[:4], h.PayloadChecksum[:]) {
		return fmt.Errorf("invalid checksum: expected %x, found %x", dig[:4], h.PayloadChecksum[:])
	}
	return nil
}

// ReadFrom reads PBTransactionMessage from io.Reader.
func (m *PBTransactionMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDPBTransaction, "PBTransactionMessage", &m.Transaction)
}

// WriteTo writes PBTransactionMessage to io.Writer.
func (m *PBTransactionMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDPBTransaction, "PBTransactionMessage", &m.Transaction)
}

func (m *PBTransactionMessage) IsMessage() {}

func (m *PBTransactionMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Transaction = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// UnmarshalMessage tries unmarshal bytes to proper Message type.
// It uses CreateMessageByContentID to create a message type.
// And can be used to unmarshal messages defined in this package only.
// Function returns error if message type is not supported.
func UnmarshalMessage(b []byte) (Message, error) {
	return UnmarshalMessageWith(b, CreateMessageByContentID)
}

// UnmarshalMessageWith tries unmarshal bytes to proper Message type with a given MessageProducer
// to create a message type.
// Use this function to unmarshal custom sets of messages.
func UnmarshalMessageWith(b []byte, mp MessageProducer) (Message, error) {
	l, err := safecast.ToUint32(len(b))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	if l < headerSizeWithoutPayload {
		return nil, errors.New("message is too short")
	}
	m, err := mp(PeerMessageID(b[HeaderContentIDPosition]))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	err = m.UnmarshalBinary(b)
	return m, err
}

// GetBlockIDsMessage is used for Signatures or hashes block IDs.
type GetBlockIDsMessage struct {
	Blocks BlockIDsPayload
}

func (m *GetBlockIDsMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, bl := range m.Blocks {
		b := bl.Bytes()
		idLen := len(b)
		body = append(body, byte(idLen))
		body = append(body, b...)
	}

	h, err := NewHeader(ContentIDGetBlockIDs, body)
	if err != nil {
		return nil, err
	}

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *GetBlockIDsMessage) UnmarshalBinary(data []byte) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("GetBlockIDsMessage UnmarshalBinary: %w", err)
	}
	if l < headerSizeWithoutPayload {
		return errors.New("GetBlockIDsMessage UnmarshalBinary: invalid data size")
	}
	var h Header
	if ubErr := h.UnmarshalBinary(data); ubErr != nil {
		return ubErr
	}
	if vErr := h.Validate(ContentIDGetBlockIDs); vErr != nil {
		return fmt.Errorf("GetBlockIDsMessage UnmarshalBinary: %w", vErr)
	}
	data = data[headerSizeWithPayload:]
	m.Blocks, err = unmarshalBlockIDs(data)
	if err != nil {
		return fmt.Errorf("GetBlockIDsMessage UnmarshalBinary: %w", err)
	}
	return nil
}

func (m *GetBlockIDsMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDGetBlockIDs, "GetBlockIDsMessage", &m.Blocks)
}

func (m *GetBlockIDsMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDGetBlockIDs, "GetBlockIDsMessage", &m.Blocks)
}

func (m *GetBlockIDsMessage) IsMessage() {}

func (m *GetBlockIDsMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockIDsPayload); ok {
		m.Blocks = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// BlockIDsMessage is used for Signatures or hashes block ids.
type BlockIDsMessage struct {
	Blocks BlockIDsPayload
}

func (m *BlockIDsMessage) MarshalBinary() ([]byte, error) {
	body := make([]byte, 4)
	binary.BigEndian.PutUint32(body[0:4], uint32(len(m.Blocks)))
	for _, bl := range m.Blocks {
		b := bl.Bytes()
		idLen := len(b)
		body = append(body, byte(idLen))
		body = append(body, b...)
	}

	h, err := NewHeader(ContentIDBlockIDs, body)
	if err != nil {
		return nil, err
	}

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	body = append(hdr, body...)

	return body, nil
}

func (m *BlockIDsMessage) UnmarshalBinary(data []byte) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("BlockIDsMessage UnmarshalBinary: %w", err)
	}
	if l < headerSizeWithPayload {
		return errors.New("BlockIDsMessage UnmarshalBinary: invalid data size")
	}
	var h Header
	if ubErr := h.UnmarshalBinary(data); ubErr != nil {
		return ubErr
	}
	if vErr := h.Validate(ContentIDBlockIDs); vErr != nil {
		return fmt.Errorf("BlockIDsMessage UnmarshalBinary: %w", vErr)
	}
	data = data[headerSizeWithPayload:]
	m.Blocks, err = unmarshalBlockIDs(data)
	if err != nil {
		return fmt.Errorf("BlockIDsMessage UnmarshalBinary: %w", err)
	}
	return nil
}

func (m *BlockIDsMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDBlockIDs, "BlockIDsMessage", &m.Blocks)
}

func (m *BlockIDsMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDBlockIDs, "BlockIDsMessage", &m.Blocks)
}

func (m *BlockIDsMessage) IsMessage() {}

func (m *BlockIDsMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockIDsPayload); ok {
		m.Blocks = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type GetBlockSnapshotMessage struct {
	BlockID BlockID
}

func (m *GetBlockSnapshotMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDGetBlockSnapshot, "GetBlockSnapshotMessage", &m.BlockID)
}

func (m *GetBlockSnapshotMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDGetBlockSnapshot, "GetBlockSnapshotMessage", &m.BlockID)
}

func (m *GetBlockSnapshotMessage) UnmarshalBinary(data []byte) error {
	return ParseMessage(data, ContentIDGetBlockSnapshot, "GetBlockSnapshotMessage", func(payload []byte) error {
		blockID, err := NewBlockIDFromBytes(payload)
		if err != nil {
			return err
		}
		m.BlockID = blockID
		return nil
	})
}

func (m *GetBlockSnapshotMessage) MarshalBinary() ([]byte, error) {
	body := m.BlockID.Bytes()

	h, err := NewHeader(ContentIDGetBlockSnapshot, body)
	if err != nil {
		return nil, err
	}
	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

func (m *GetBlockSnapshotMessage) IsMessage() {}

func (m *GetBlockSnapshotMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockID); ok {
		m.BlockID = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type BlockSnapshotMessage struct {
	Bytes BytesPayload
}

func (m *BlockSnapshotMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDBlockSnapshot, "BlockSnapshotMessage", &m.Bytes)
}

func (m *BlockSnapshotMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDBlockSnapshot, "BlockSnapshotMessage", &m.Bytes)
}

func (m *BlockSnapshotMessage) UnmarshalBinary(data []byte) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("BlockSnapshotMessage UnmarshalBinary: %w", err)
	}
	if l < maxHeaderLength {
		return errors.New("BlockSnapshotMessage UnmarshalBinary: invalid data size")
	}
	var h Header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDBlockSnapshot {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	m.Bytes = make([]byte, h.payloadLength)
	copy(m.Bytes, data[maxHeaderLength:maxHeaderLength+h.payloadLength])
	return nil
}

func (m *BlockSnapshotMessage) MarshalBinary() ([]byte, error) {
	body := m.Bytes

	h, err := NewHeader(ContentIDBlockSnapshot, body)
	if err != nil {
		return nil, err
	}

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

func (m *BlockSnapshotMessage) IsMessage() {}

func (m *BlockSnapshotMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Bytes = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type MicroBlockSnapshotMessage struct {
	Bytes BytesPayload
}

func (m *MicroBlockSnapshotMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroBlockSnapshot, "MicroBlockSnapshotMessage", &m.Bytes)
}

func (m *MicroBlockSnapshotMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDMicroBlockSnapshot, "MicroBlockSnapshotMessage", &m.Bytes)
}

func (m *MicroBlockSnapshotMessage) UnmarshalBinary(data []byte) error {
	l, err := safecast.ToUint32(len(data))
	if err != nil {
		return fmt.Errorf("MicroBlockSnapshotMessage UnmarshalBinary: %w", err)
	}
	if l < maxHeaderLength {
		return errors.New("MicroBlockSnapshotMessage UnmarshalBinary: invalid data size")
	}
	var h Header

	if err := h.UnmarshalBinary(data); err != nil {
		return err
	}
	if h.Magic != headerMagic {
		return fmt.Errorf("wrong magic in Header: %x", h.Magic)
	}
	if h.ContentID != ContentIDMicroBlockSnapshot {
		return fmt.Errorf("wrong ContentID in Header: %x", h.ContentID)
	}
	m.Bytes = make([]byte, h.payloadLength)
	copy(m.Bytes, data[maxHeaderLength:maxHeaderLength+h.payloadLength])
	return nil
}

func (m *MicroBlockSnapshotMessage) MarshalBinary() ([]byte, error) {
	body := m.Bytes

	h, err := NewHeader(ContentIDMicroBlockSnapshot, body)
	if err != nil {
		return nil, err
	}

	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

func (m *MicroBlockSnapshotMessage) IsMessage() {}

func (m *MicroBlockSnapshotMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BytesPayload); ok {
		m.Bytes = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

type MicroBlockSnapshotRequestMessage struct {
	BlockID BlockID
}

func (m *MicroBlockSnapshotRequestMessage) ReadFrom(r io.Reader) (int64, error) {
	return ReadMessage(r, ContentIDMicroBlockSnapshotRequest, "MicroBlockSnapshotRequestMessage", &m.BlockID)
}

func (m *MicroBlockSnapshotRequestMessage) WriteTo(w io.Writer) (int64, error) {
	return WriteMessage(w, ContentIDMicroBlockSnapshotRequest, "MicroBlockSnapshotRequestMessage", &m.BlockID)
}

func (m *MicroBlockSnapshotRequestMessage) UnmarshalBinary(data []byte) error {
	return ParseMessage(
		data,
		ContentIDMicroBlockSnapshotRequest,
		"MicroBlockSnapshotRequestMessage",
		func(payload []byte) error {
			id, err := NewBlockIDFromBytes(payload)
			if err != nil {
				return fmt.Errorf("failed to unmarshal MicroBlockSnapshotRequestMessage: %w", err)
			}
			m.BlockID = id
			return nil
		})
}

func (m *MicroBlockSnapshotRequestMessage) MarshalBinary() ([]byte, error) {
	body := m.BlockID.Bytes()
	h, err := NewHeader(ContentIDMicroBlockSnapshotRequest, body)
	if err != nil {
		return nil, err
	}
	hdr, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}
	body = append(hdr, body...)
	return body, nil
}

func (m *MicroBlockSnapshotRequestMessage) IsMessage() {}

func (m *MicroBlockSnapshotRequestMessage) SetPayload(payload Payload) (Message, error) {
	if p, ok := payload.(*BlockID); ok {
		m.BlockID = *p
		return m, nil
	}
	return nil, fmt.Errorf("invalid payload type %T", payload)
}

// MessageProducer is a function that creates a message by provided content ID.
type MessageProducer func(PeerMessageID) (Message, error)

func CreateMessageByContentID(contentID PeerMessageID) (Message, error) {
	switch contentID {
	case ContentIDGetPeers:
		return &GetPeersMessage{}, nil
	case ContentIDPeers:
		return &PeersMessage{}, nil
	case ContentIDGetSignatures:
		return &GetSignaturesMessage{}, nil
	case ContentIDSignatures:
		return &SignaturesMessage{}, nil
	case ContentIDGetBlock:
		return &GetBlockMessage{}, nil
	case ContentIDBlock:
		return &BlockMessage{}, nil
	case ContentIDScore:
		return &ScoreMessage{}, nil
	case ContentIDTransaction:
		return &TransactionMessage{}, nil
	case ContentIDMicroblock:
		return &MicroBlockMessage{}, nil
	case ContentIDMicroblockRequest:
		return &MicroBlockRequestMessage{}, nil
	case ContentIDInvMicroblock:
		return &MicroBlockInvMessage{}, nil
	case ContentIDPBBlock:
		return &PBBlockMessage{}, nil
	case ContentIDPBMicroBlock:
		return &PBMicroBlockMessage{}, nil
	case ContentIDPBTransaction:
		return &PBTransactionMessage{}, nil
	case ContentIDGetBlockIDs:
		return &GetBlockIDsMessage{}, nil
	case ContentIDBlockIDs:
		return &BlockIDsMessage{}, nil
	case ContentIDGetBlockSnapshot:
		return &GetBlockSnapshotMessage{}, nil
	case ContentIDMicroBlockSnapshotRequest:
		return &MicroBlockSnapshotRequestMessage{}, nil
	case ContentIDBlockSnapshot:
		return &BlockSnapshotMessage{}, nil
	case ContentIDMicroBlockSnapshot:
		return &MicroBlockSnapshotMessage{}, nil
	default:
		return nil, fmt.Errorf("unexpected content ID %d", contentID)
	}
}

// ReadMessageFrom reads message from io.Reader.
// Use this function to read messages defined in this package only.
func ReadMessageFrom(r io.Reader) (Message, int64, error) {
	return ReadMessageFromWith(r, CreatePayloadByContentID, CreateMessageByContentID)
}

// ReadMessageFromWith reads message from io.Reader with custom payload and message producers.
// Use this function to read custom sets of messages.
func ReadMessageFromWith(r io.Reader, pp PayloadProducer, mp MessageProducer) (Message, int64, error) {
	var h Header
	n1, err := h.ReadFrom(r)
	if err != nil {
		return nil, n1, fmt.Errorf("failed to read header: %w", err)
	}
	if vErr := h.Validate(h.ContentID); vErr != nil {
		return nil, n1, fmt.Errorf("message header is not valid: %w", vErr)
	}
	msg, err := mp(h.ContentID)
	if err != nil {
		return nil, n1, fmt.Errorf("failed to create message: %w", err)
	}
	if h.payloadLength == 0 { // Fast exit for messages without payload.
		return msg, n1, nil
	}
	payload, err := pp(h.ContentID)
	if err != nil {
		return nil, n1, fmt.Errorf("failed to create payload: %w", err)
	}
	pr, err := NewChecksumReader(io.LimitReader(r, int64(h.payloadLength)))
	if err != nil {
		return nil, n1, fmt.Errorf("failed to create checksum reader: %w", err)
	}
	n2, err := payload.ReadFrom(pr)
	if err != nil {
		return nil, n1 + n2, fmt.Errorf("failed to read payload: %w", err)
	}
	if pr.Checksum() != h.PayloadChecksum {
		return nil, n1 + n2, errors.New("payload checksum mismatch")
	}
	if msg, err = msg.SetPayload(payload); err != nil {
		return nil, n1 + n2, fmt.Errorf("failed to set payload: %w", err)
	}
	return msg, n1 + n2, nil
}

func unmarshalBlockIDs(data []byte) ([]BlockID, error) {
	if len(data) < uint32Size {
		return nil, fmt.Errorf("message too short %v", len(data))
	}
	count := binary.BigEndian.Uint32(data[0:uint32Size])
	data = data[uint32Size:]
	pos := 0
	dl := len(data)
	ids := make([]BlockID, count)
	for i := range count {
		if pos+1 > dl {
			return nil, fmt.Errorf("message too short %v", dl)
		}
		l := int(data[pos])
		pos++ // Skip length byte.
		if pos+l > dl {
			return nil, fmt.Errorf("message too short %v", dl)
		}
		id, err := NewBlockIDFromBytes(data[pos : pos+l])
		if err != nil {
			return nil, errors.Wrap(err, "bad block ID bytes")
		}
		ids[i] = id
		pos += l
	}
	return ids, nil
}
