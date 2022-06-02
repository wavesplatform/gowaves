package services

import (
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
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
	Peers           peer_manager.PeerManager
	Scheduler       types.Scheduler
	BlocksApplier   BlocksApplier
	UtxPool         types.UtxPool
	Scheme          proto.Scheme
	InvRequester    types.InvRequester
	LoggableRunner  runner.LogRunner
	Time            types.Time
	Wallet          types.EmbeddedWallet
	MicroBlockCache MicroBlockCache
	InternalChannel chan messages.InternalMessage
	MinPeersMining  int
	SkipMessageList *messages.SkipMessageList
}
