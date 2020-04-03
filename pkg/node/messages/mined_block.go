package messages

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type MinedBlockInternalMessage struct {
	Block   *proto.Block
	Limits  proto.MiningLimits
	KeyPair proto.KeyPair
	vrf     []byte
}

func NewMinedBlockInternalMessage(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) *MinedBlockInternalMessage {
	return &MinedBlockInternalMessage{
		Block:   block,
		Limits:  limits,
		KeyPair: keyPair,
		vrf:     common.Dup(vrf),
	}
}

func (a *MinedBlockInternalMessage) Internal() {
}
