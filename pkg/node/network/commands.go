package network

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Command interface{ networkCommandTypeMaker() }

type FollowGroupCommand struct{}

func (c FollowGroupCommand) networkCommandTypeMaker() {}

type FollowLeaderCommand struct{}

func (c FollowLeaderCommand) networkCommandTypeMaker() {}

type BlacklistPeerCommand struct {
	Peer    peer.Peer
	Message string
}

func (c BlacklistPeerCommand) networkCommandTypeMaker() {}

type BroadcastTransactionCommand struct {
	Transaction proto.Transaction
	Origin      peer.Peer
}

func (c BroadcastTransactionCommand) networkCommandTypeMaker() {}

type AnnounceScoreCommand struct{}

func (c AnnounceScoreCommand) networkCommandTypeMaker() {}

type BroadcastMicroBlockInvCommand struct {
	MicroBlockInv *proto.MicroBlockInv
	Origin        peer.Peer
}

func (c BroadcastMicroBlockInvCommand) networkCommandTypeMaker() {}
