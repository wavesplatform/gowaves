package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type InternalMessage interface {
	Internal()
}

type MinedBlockInternalMessage struct {
	Block   *proto.Block
	Limits  proto.MiningLimits
	KeyPair proto.KeyPair
}

func (a *MinedBlockInternalMessage) Internal() {
}
