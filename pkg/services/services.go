package services

import (
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlocksApplier interface {
	BlockExists(state state.State, block *proto.Block) (bool, error)
	Apply(state state.State, block []*proto.Block) (proto.Height, error)
	ApplyMicro(state state.State, block *proto.Block) (proto.Height, error)
}

type MicroBlockCache interface {
	Add(blockID proto.BlockID, micro *proto.MicroBlock)
	Get(proto.BlockID) (*proto.MicroBlock, bool)
}

type MicroBlockInvCache interface {
	Add(blockID proto.BlockID, micro *proto.MicroBlockInv)
	Get(proto.BlockID) (*proto.MicroBlockInv, bool)
}

type Services struct {
	NodeName        string
	State           state.State
	Peers           peers.PeerManager
	Scheduler       types.Scheduler
	BlocksApplier   BlocksApplier
	UtxPool         types.UtxPool
	Scheme          proto.Scheme
	Time            types.Time
	Wallet          types.EmbeddedWallet
	SkipMessageList *messages.SkipMessageList
}
