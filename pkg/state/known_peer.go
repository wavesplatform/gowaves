package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
)

const KnownPeerLength = proto.IpPortLength

type KnownPeer proto.IpPort
type KnownPeerKey = [1 + proto.IpPortLength]byte

func NewKnownPeer(ip net.IP, port int) KnownPeer {
	return NewKnownPeerFromTcpAddr(proto.TCPAddr{
		IP:   ip,
		Port: port,
	})
}

func NewKnownPeerFromTcpAddr(a proto.TCPAddr) KnownPeer {
	out := KnownPeer{}
	buf := new(bytes.Buffer)
	_, _ = a.WriteTo(buf)
	copy(out[:], buf.Bytes())
	return out
}

func NewKnownPeerFromKey(key KnownPeerKey) KnownPeer {
	out := KnownPeer{}
	copy(out[:], key[1:])
	return out
}

func (a KnownPeer) key() KnownPeerKey {
	key := KnownPeerKey{}
	key[0] = knownPeersPrefix
	copy(key[1:1+KnownPeerLength], a[:])
	return key
}

func (a KnownPeer) Addr() net.IP {
	return net.IP(a[:16])
}

func (a KnownPeer) Port() int {
	b := binary.BigEndian.Uint64(a[16:24])
	return int(b)
}

func (a KnownPeer) String() string {
	return fmt.Sprintf("%s:%d", a.Addr(), a.Port())
}

//func (a *KnownPeer) fromKey(k KnownPeerKey) error {
//	copy(a.IP[:], k[1:17])
//	a.Port = binary.BigEndian.Uint16(k[17:])
//	return nil
//}

func (a *KnownPeer) UnmarshalBinary(b []byte) error {
	if len(b) < KnownPeerLength {
		return errors.Errorf("too low bytes to unmarshal KnownPeer, expected at least %d, got %d", KnownPeerLength, len(b))
	}

	k := KnownPeer{}
	copy(k[:], b)
	return nil
}

//func (a KnownPeer) String() string {
//	return proto.NodeAddr(a).String()
//}
