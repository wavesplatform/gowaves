package storage

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"time"
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
type knownPeers map[KnownPeer]struct{}

func fromUnixMillis(timestampMillis int64) time.Time {
	sec := timestampMillis / 1_000
	nsec := (timestampMillis % 1_000) * 1_000_000
	return time.Unix(sec, nsec)
}
