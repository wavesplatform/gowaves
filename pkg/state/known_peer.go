package state

import (
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const KnownPeerKeyLength = proto.IpPortLength + 1

func IntoBytes(p proto.TCPAddr) []byte {
	out := make([]byte, KnownPeerKeyLength)
	out[0] = knownPeersPrefix
	ipPort := p.ToIpPort()
	copy(out[1:], ipPort[:])
	return out
}

func FromBytes(b []byte) (proto.TCPAddr, error) {
	i := proto.IpPort{}
	if len(b) < KnownPeerKeyLength {
		return i.ToTcpAddr(), errors.Errorf("not enough bytes to decode to KnownPeerKey")
	}
	copy(i[:], b[1:])
	return i.ToTcpAddr(), nil
}
