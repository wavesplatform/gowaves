package network

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

// Notification represents messages produced by the Network service.
// The Network service issues notifications for the following events:
//   - QuorumMet: Triggered when the required number of peers connect.
//   - QuorumLost: Triggered when the count of connected peers falls below the required threshold.
//   - SyncPeerChanged: Activated when the Network selects a new peer for synchronization.
//   - NoSyncPeer: Triggered when the Network loses a peer used for synchronization and unable to select another one.
type Notification interface{ networkNotificationTypeMarker() }

// QuorumMetNotification signals when the required threshold of connected peers is reached.
type QuorumMetNotification struct {
	Peer peer.Peer
}

func (n QuorumMetNotification) networkNotificationTypeMarker() {}

// QuorumLostNotification signals when the count of connected peers drops below the required threshold.
type QuorumLostNotification struct{}

func (n QuorumLostNotification) networkNotificationTypeMarker() {}

// SyncPeerChangedNotification signals the selection of a new peer for synchronization.
type SyncPeerChangedNotification struct {
	Peer  peer.Peer
	Score *big.Int
}

func (n SyncPeerChangedNotification) networkNotificationTypeMarker() {}
