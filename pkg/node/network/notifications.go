package network

import "github.com/wavesplatform/gowaves/pkg/p2p/peer"

// Notification represents messages produced by the Network service.
// The Network service issues notifications for the following events:
//   - QuorumMet: Triggered when the required number of peers connect.
//   - QuorumLost: Triggered when the count of connected peers falls below the required threshold.
//   - SyncPeerSelected: Activated when the Network selects a new peer for synchronization.
//   - NoSyncPeer: Triggered when the Network loses a peer used for synchronization and unable to select another one.
type Notification interface{ networkNotificationTypeMaker() }

// QuorumMetNotification signals when the required threshold of connected peers is reached.
type QuorumMetNotification struct{}

func (n QuorumMetNotification) networkNotificationTypeMaker() {}

// QuorumLostNotification signals when the count of connected peers drops below the required threshold.
type QuorumLostNotification struct{}

func (n QuorumLostNotification) networkNotificationTypeMaker() {}

// SyncPeerSelectedNotification signals the selection of a new peer for synchronization.
type SyncPeerSelectedNotification struct {
	Peer peer.Peer
}

func (n SyncPeerSelectedNotification) networkNotificationTypeMaker() {}
