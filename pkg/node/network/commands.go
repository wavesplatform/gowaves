package network

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Command interface{ networkCommandTypeMarker() }

type FollowGroupCommand struct{}

func (c FollowGroupCommand) networkCommandTypeMarker() {}

type FollowLeaderCommand struct{}

func (c FollowLeaderCommand) networkCommandTypeMarker() {}

type BlacklistPeerCommand struct {
	Peer    peer.Peer
	Message string
}

func (c BlacklistPeerCommand) networkCommandTypeMarker() {}

type SuspendPeerCommand struct {
	Peer    peer.Peer
	Message string
}

func (c SuspendPeerCommand) networkCommandTypeMarker() {}

type BroadcastTransactionCommand struct {
	Transaction proto.Transaction
	Origin      peer.Peer
}

func (c BroadcastTransactionCommand) networkCommandTypeMarker() {}

type AnnounceScoreCommand struct{}

func (c AnnounceScoreCommand) networkCommandTypeMarker() {}

type BroadcastMicroBlockInvCommand struct {
	MicroBlockInv *proto.MicroBlockInv
	Origin        peer.Peer
}

func (c BroadcastMicroBlockInvCommand) networkCommandTypeMarker() {}

// RequestQuorumUpdate issued by the Node's FSM in order to receive QuorumMetNotification if there is a quorum.
type RequestQuorumUpdate struct{}

func (c RequestQuorumUpdate) networkCommandTypeMarker() {}
