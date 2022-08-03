package internal

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type PeerForkInfo struct {
	Peer    net.IP        `json:"peer"`
	Lag     int           `json:"lag"`
	Name    string        `json:"name"`
	Version proto.Version `json:"version"`
}

type Fork struct {
	Longest          bool           `json:"longest"`            // Indicates that the fork is the longest
	Height           int            `json:"height"`             // The height of the last block in the fork
	HeadBlock        proto.BlockID  `json:"head_block"`         // The last block of the fork
	LastCommonHeight int            `json:"last_common_height"` // The height of the last common block
	LastCommonBlock  proto.BlockID  `json:"last_common_block"`  // The last common block with the longest fork
	Length           int            `json:"length"`             // The number of blocks since the last common block
	Peers            []PeerForkInfo `json:"peers"`              // Peers that seen on the fork
}

type ForkByHeightLengthAndPeersCount []Fork

func (a ForkByHeightLengthAndPeersCount) Len() int {
	return len(a)
}

func (a ForkByHeightLengthAndPeersCount) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ForkByHeightLengthAndPeersCount) Less(i, j int) bool {
	if a[i].Longest {
		return true
	}
	if a[i].Height > a[j].Height {
		return true
	}
	if a[i].Length > a[j].Length {
		return true
	}
	return len(a[i].Peers) > len(a[j].Peers)
}

type NodeForkInfo struct {
	Address    net.IP        `json:"address"`
	Nonce      int           `json:"nonce"`
	Name       string        `json:"name"`
	Version    proto.Version `json:"version"`
	OnFork     Fork          `json:"on_fork"`
	OtherForks []Fork        `json:"other_forks"`
}

type NodeState byte

const (
	NodeUnknown    = iota // Unknown node
	NodeDiscarded         // Network connection to the node failed
	NodeResponding        // Network connection to the node was successful
	NodeGreeted           // Handshake with the node was successful
	NodeHostile           // The node has different blockchain scheme
)

const (
	timeBinarySize     = 1 + 8 + 4 + 2
	peerNodeBinarySize = net.IPv6len + 2 + 8 + 1 + 3*4 + 4 + timeBinarySize + 1
)

type PeerNode struct {
	Address     net.IP        `json:"address"`
	Port        uint16        `json:"port"`
	Nonce       uint64        `json:"nonce"`
	Name        string        `json:"name"`
	Version     proto.Version `json:"version"`
	Attempts    int           `json:"attempts"`
	NextAttempt time.Time     `json:"next_attempt"`
	State       NodeState     `json:"state"`
}

func (a PeerNode) String() string {
	sb := strings.Builder{}
	sb.WriteString(net.JoinHostPort(a.Address.String(), strconv.Itoa(int(a.Port))))
	sb.WriteRune('|')
	sb.WriteString(strconv.Itoa(int(a.Nonce)))
	sb.WriteRune('|')
	sb.WriteString(a.Name)
	sb.WriteRune('|')
	switch a.State {
	case NodeUnknown:
		sb.WriteString("UNKNOWN")
	case NodeDiscarded:
		sb.WriteString("DISCARDED")
	case NodeResponding:
		sb.WriteString("RESPONDING")
	case NodeGreeted:
		sb.WriteString("GREETED")
	case NodeHostile:
		sb.WriteString("HOSTILE")
	}
	sb.WriteRune('|')
	sb.WriteRune('v')
	sb.WriteString(a.Version.String())
	sb.WriteRune('|')
	sb.WriteString(strconv.Itoa(a.Attempts))
	sb.WriteRune('|')
	sb.WriteString(a.NextAttempt.Format(time.RFC3339))
	return sb.String()
}

func (a PeerNode) MarshalBinary() ([]byte, error) {
	buf := make([]byte, peerNodeBinarySize+len(a.Name))
	pos := 0
	copy(buf[pos:], a.Address.To16())
	pos += net.IPv6len
	binary.BigEndian.PutUint16(buf[pos:], a.Port)
	pos += 2
	binary.BigEndian.PutUint64(buf[pos:], a.Nonce)
	pos += 8
	proto.PutStringWithUInt8Len(buf[pos:], a.Name)
	pos += 1 + len(a.Name)
	binary.BigEndian.PutUint32(buf[pos:], a.Version.Major())
	pos += 4
	binary.BigEndian.PutUint32(buf[pos:], a.Version.Minor())
	pos += 4
	binary.BigEndian.PutUint32(buf[pos:], a.Version.Patch())
	pos += 4
	binary.BigEndian.PutUint32(buf[pos:], uint32(a.Attempts))
	pos += 4
	tb, err := a.NextAttempt.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal PeerNode to bytes")
	}
	copy(buf[pos:], tb)
	pos += timeBinarySize
	buf[pos] = byte(a.State)
	return buf, nil
}

func (a *PeerNode) UnmarshalBinary(data []byte) error {
	if l := len(data); l < peerNodeBinarySize {
		return errors.Errorf("%d is not enough bytes for PeerNode", l)
	}
	a.Address = make([]byte, net.IPv6len)
	copy(a.Address[:], data[:net.IPv6len])
	data = data[net.IPv6len:]
	a.Port = binary.BigEndian.Uint16(data[:2])
	data = data[2:]
	a.Nonce = binary.BigEndian.Uint64(data[:8])
	data = data[8:]
	n, err := proto.StringWithUInt8Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerNode")
	}
	a.Name = n
	data = data[1+len(n):]
	major := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	minor := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	patch := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	a.Version = proto.NewVersion(major, minor, patch)
	a.Attempts = int(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]
	t := time.Time{}
	err = t.UnmarshalBinary(data[:timeBinarySize])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PeerNode")
	}
	a.NextAttempt = t
	data = data[timeBinarySize:]
	a.State = NodeState(data[0])
	return nil
}

type PeerNodesByName []PeerNode

func (a PeerNodesByName) Len() int {
	return len(a)
}

func (a PeerNodesByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a PeerNodesByName) Less(i, j int) bool {
	x := a[i].Name
	y := a[j].Name
	return strings.ToUpper(x) < strings.ToUpper(y)
}
