package messages

import "github.com/wavesplatform/gowaves/pkg/proto"

type MinedBlockInternalMessage struct {
	Block   *proto.Block
	Limits  proto.MiningLimits
	KeyPair proto.KeyPair
}

func NewMinedBlockInternalMessage(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) *MinedBlockInternalMessage {
	return &MinedBlockInternalMessage{Block: block, Limits: limits, KeyPair: keyPair}
}

func (a *MinedBlockInternalMessage) Internal() {
}
