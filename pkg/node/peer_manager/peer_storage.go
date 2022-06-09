package peer_manager

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
)

type PeerStorage interface {
	Known(limit int) []storage.KnownPeer
	AddOrUpdateKnown(known []storage.KnownPeer, now time.Time) error
	DeleteKnown(known []storage.KnownPeer) error
	DropKnown() error

	Suspended(now time.Time) []storage.SuspendedPeer
	AddSuspended(suspended []storage.SuspendedPeer) error
	IsSuspendedIP(ip storage.IP, now time.Time) bool
	IsSuspendedIPs(ips []storage.IP, now time.Time) []bool
	DeleteSuspendedByIP(suspended []storage.SuspendedPeer) error
	RefreshSuspended(now time.Time) error
	DropSuspended() error

	DropStorage() error
}
