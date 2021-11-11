package storage

import (
	"net"
	"sort"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type IP [net.IPv6len]byte

func (i *IP) String() string {
	return net.IP(i[:]).String()
}

func IPFromString(s string) IP {
	parsed := net.ParseIP(s)
	ip := IP{}
	copy(ip[:], parsed[:net.IPv6len])
	return ip
}

func IpFromIpPort(ipPort proto.IpPort) IP {
	ip := IP{}
	copy(ip[:], ipPort[:net.IPv6len])
	return ip
}

type KnownPeer proto.IpPort

func (kp *KnownPeer) IP() IP {
	return IpFromIpPort(proto.IpPort(*kp))
}

func (kp *KnownPeer) IpPort() proto.IpPort {
	return proto.IpPort(*kp)
}

func (kp *KnownPeer) String() string {
	ipPort := kp.IpPort()
	return ipPort.String()
}

type SuspendedPeer struct {
	IP                     IP            `cbor:"0,keyasint,omitemtpy"`
	SuspendTimestampMillis int64         `cbor:"1,keyasint,omitemtpy"`
	SuspendDuration        time.Duration `cbor:"2,keyasint,omitemtpy"`
	Reason                 string        `cbor:"3,keyasint,omitemtpy"`
}

func (sp *SuspendedPeer) SuspendTime() time.Time {
	return fromUnixMillis(sp.SuspendTimestampMillis)
}

func (sp *SuspendedPeer) AwakeTime() time.Time {
	return sp.SuspendTime().Add(sp.SuspendDuration)
}

func (sp *SuspendedPeer) IsSuspended(now time.Time) bool {
	awakeTime := sp.AwakeTime()
	return awakeTime.After(now)
}

type suspendedPeers map[IP]SuspendedPeer

type pair struct {
	peer KnownPeer
	ts   int64
}

type pairs []pair

func (p pairs) Len() int { return len(p) }

func (p pairs) Less(i, j int) bool { return p[i].ts < p[j].ts }

func (p pairs) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type knownPeers map[KnownPeer]int64

func (a knownPeers) OldestFirst(limit int) []KnownPeer {
	ps := make(pairs, len(a))
	i := 0
	for k, v := range a {
		ps[i] = pair{k, v}
		i++
	}
	sort.Sort(ps)
	l := len(ps)
	if l > limit {
		l = limit
	}
	r := make([]KnownPeer, l)
	for i := 0; i < l; i++ {
		r[i] = ps[i].peer
	}
	return r
}

func fromUnixMillis(timestampMillis int64) time.Time {
	sec := timestampMillis / 1_000
	nsec := (timestampMillis % 1_000) * 1_000_000
	return time.Unix(sec, nsec)
}
