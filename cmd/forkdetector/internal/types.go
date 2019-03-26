package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"hash/fnv"
	"net"
	"strconv"
	"strings"
)

const (
	PeerAddrLen   = net.IPv6len + 2
	PeerAddrV4Len = 4 + 2
)

// PeerAddr is a `net.TCPAddr` with helper methods for marshalling and hash identification.
type PeerAddr net.TCPAddr

// ParsePeerAddr tries to create a PeerAddr from given string.
func ParsePeerAddr(s string) (PeerAddr, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to create PeerAddr from string")
	}
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return PeerAddr{}, wrapError(err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return PeerAddr{}, wrapError(err)
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return PeerAddr{}, errors.Errorf("failed to create PeerAddr from string: no IP")
	}
	return PeerAddr(net.TCPAddr{IP: ip, Port: port}), nil
}

func (pa PeerAddr) MarshalJSON() ([]byte, error) {
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(pa.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (pa *PeerAddr) UnmarshalJSON(data []byte) error {
	a, err := ParsePeerAddr(string(data))
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerAddr from JSON")
	}
	*pa = a
	return nil
}

func (pa PeerAddr) String() string {
	a := net.TCPAddr(pa)
	return a.String()
}

func (pa PeerAddr) Hash() uint64 {
	hash := fnv.New64()
	hash.Reset()
	_, err := hash.Write(pa.IP)
	if err != nil {
		panic("err should be always nil")
	}
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(pa.Port))
	_, err = hash.Write(buf)
	if err != nil {
		panic("err should be always nil")
	}
	return hash.Sum64()
}

func (pa PeerAddr) MarshalBinary() ([]byte, error) {
	buf := make([]byte, PeerAddrLen)
	copy(buf[0:net.IPv6len], pa.IP)
	binary.BigEndian.PutUint16(buf[net.IPv6len:], uint16(pa.Port))
	return buf, nil
}

func (pa *PeerAddr) UnmarshalBinary(data []byte) error {
	if l := len(data); l < PeerAddrLen {
		return errors.Errorf("%d is not enough bytes for PeerAddr", l)
	}
	pa.IP = make([]byte, net.IPv6len)
	copy(pa.IP, data[:net.IPv6len])
	pa.Port = int(binary.BigEndian.Uint16(data[net.IPv6len:]))
	return nil
}

// PeerAddrV4 represents PeerAddr which contains IP address of version 4. Should be used in marshalling of network messages.
type PeerAddrV4 PeerAddr

func (pa PeerAddrV4) MarshalBinary() ([]byte, error) {
	buf := make([]byte, PeerAddrV4Len)
	ip4 := net.TCPAddr(pa).IP.To4()
	if ip4 == nil {
		return nil, errors.Errorf("failed to marshal PeerAddrV4 to bytes: contains IPv6 address")
	}
	copy(buf[0:4], ip4)
	binary.BigEndian.PutUint16(buf[4:6], uint16(pa.Port))
	return buf, nil
}

func (pa *PeerAddrV4) UnmarshalBinary(data []byte) error {
	if l := len(data); l < PeerAddrV4Len {
		return errors.Errorf("%d is not enough bytes for PeerAddrV4", l)
	}
	pa.IP = net.IPv4(data[0], data[1], data[2], data[3])
	pa.Port = int(binary.BigEndian.Uint16(data[4:6]))
	return nil
}

func (pa PeerAddrV4) String() string {
	a := net.TCPAddr(pa)
	return a.String()
}

type PeerDesignation struct {
	Address net.IP
	Nonce   uint64
}

func NewPeerDesignation(address net.IP, nonce uint64) PeerDesignation {
	a := make(net.IP, len(address))
	copy(a, address)
	return PeerDesignation{Address: a, Nonce: nonce}
}

func (pd PeerDesignation) Hash() uint64 {
	hash := fnv.New64()
	hash.Reset()
	_, err := hash.Write(pd.Address)
	if err != nil {
		panic("err should be always nil")
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, pd.Nonce)
	_, err = hash.Write(buf)
	if err != nil {
		panic("err should be always nil")
	}
	return hash.Sum64()
}

func (pd PeerDesignation) String() string {
	var sb strings.Builder
	sb.WriteString(pd.Address.String())
	sb.WriteRune('-')
	sb.WriteString(strconv.FormatUint(pd.Nonce, 10))
	return sb.String()
}

func (pd PeerDesignation) MarshalJSON() ([]byte, error) {
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(pd.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

type PeerForkInfo struct {
	Peer PeerDesignation `json:"peer"`
	Lag  int             `json:"lag"`
}

func NewPeerForkInfo(peer PeerDesignation, lag int) PeerForkInfo {
	return PeerForkInfo{Peer: peer, Lag: lag}
}

type Fork struct {
	HeadBlock   crypto.Signature `json:"head_block"`
	Longest     bool             `json:"longest"`
	CommonBlock crypto.Signature `json:"common_block"`
	Height      int              `json:"height"`
	Length      int              `json:"length"`
	Peers       []PeerForkInfo   `json:"peers"`
}

type ForkByHeightLengthAndPeersCount []Fork

func (a ForkByHeightLengthAndPeersCount) Len() int {
	return len(a)
}

func (a ForkByHeightLengthAndPeersCount) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ForkByHeightLengthAndPeersCount) Less(i, j int) bool {
	if a[i].Height > a[j].Height {
		return true
	}
	if a[i].Length > a[j].Length {
		return true
	}
	return len(a[i].Peers) > len(a[j].Peers)
}

type PeerDescription struct {
	Address PeerAddr      `json:"address"`
	Nonce   int           `json:"nonce"`
	Name    string        `json:"name"`
	Version proto.Version `json:"version"`
}

func NewPeerDescription(a net.Addr, h proto.Handshake) (*PeerDescription, error) {
	addr, ok := a.(*net.TCPAddr)
	if !ok {
		return nil, errors.Errorf("address '%s' not a TCP address", a)
	}
	return &PeerDescription{
		Address: PeerAddr(*addr),
		Nonce:   int(h.NodeNonce),
		Name:    h.NodeName,
		Version: h.Version,
	}, nil
}

type NodeForkInfo struct {
	Address    net.IP        `json:"address"`
	Nonce      int           `json:"nonce"`
	Name       string        `json:"name"`
	Version    proto.Version `json:"version"`
	OnFork     Fork          `json:"on_fork"`
	OtherForks []Fork        `json:"other_forks"`
}
