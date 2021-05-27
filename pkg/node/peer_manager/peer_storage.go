package peer_manager

import (
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

type PeerStorage interface {
	All() ([]proto.TCPAddr, error)
	Known() ([]proto.TCPAddr, error)
	AddKnown(proto.TCPAddr) error
	Add([]proto.TCPAddr) error
}

type PersistentPeersStorage interface {
	Known() []storage.KnownPeer
	AddKnown(known []storage.KnownPeer) error
	DeleteKnown(known []storage.KnownPeer) error

	Suspended(now time.Time) []storage.SuspendedPeer
	AddSuspended(suspended []storage.SuspendedPeer) error
	IsSuspendedIP(ip storage.IP, now time.Time) bool
	IsSuspendedIPs(ips []storage.IP, now time.Time) []bool
	DeleteSuspendedByIP(suspended []storage.SuspendedPeer) error
	RefreshSuspended(now time.Time) error
	DropSuspended() error
}
