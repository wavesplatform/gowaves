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

type restrictedPeer struct {
	IP                      IP            `cbor:"0,keyasint,omitemtpy"`
	RestrictTimestampMillis int64         `cbor:"1,keyasint,omitemtpy"`
	RestrictDuration        time.Duration `cbor:"2,keyasint,omitemtpy"`
	Reason                  string        `cbor:"3,keyasint,omitemtpy"`
}

func (sp *restrictedPeer) RestrictTime() time.Time {
	return time.UnixMilli(sp.RestrictTimestampMillis)
}

func (sp *restrictedPeer) AwakeTime() time.Time {
	return sp.RestrictTime().Add(sp.RestrictDuration)
}

func (sp *restrictedPeer) IsRestricted(now time.Time) bool {
	awakeTime := sp.AwakeTime()
	return awakeTime.After(now)
}

type restrictedPeers map[IP]restrictedPeer

type SuspendedPeer = restrictedPeer

func NewSuspendedPeer(ip IP, suspendTimestampMillis int64, suspendDuration time.Duration, reason string) SuspendedPeer {
	return SuspendedPeer{
		IP:                      ip,
		RestrictTimestampMillis: suspendTimestampMillis,
		RestrictDuration:        suspendDuration,
		Reason:                  reason,
	}
}

type suspendedPeers = restrictedPeers

type BlackListedPeer = restrictedPeer

func NewBlackListedPeer(ip IP, blackListTimestampMillis int64, blackListDuration time.Duration, reason string) BlackListedPeer {
	return BlackListedPeer{
		IP:                      ip,
		RestrictTimestampMillis: blackListTimestampMillis,
		RestrictDuration:        blackListDuration,
		Reason:                  reason,
	}
}

type blackListedPeers = restrictedPeers

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
