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

	BlackList(now time.Time) []storage.BlackListedPeer
	AddToBlackList(blackListed []storage.BlackListedPeer) error
	IsBlackListedIP(ip storage.IP, now time.Time) bool
	IsBlackListedIPs(ips []storage.IP, now time.Time) []bool
	DeleteBlackListedByIP(blackListed []storage.BlackListedPeer) error
	RefreshBlackList(now time.Time) error
	DropBlackList() error

	DropStorage() error
}
