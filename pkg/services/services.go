package services

import (
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlocksApplier interface {
	Apply(state state.State, block []*proto.Block) error
}

type MicroBlockCache interface {
	Add(micro *proto.MicroBlock)
	Get(sig proto.MicroblockTotalSig) (*proto.MicroBlock, bool)
}

type Services struct {
	State              state.State
	Peers              peer_manager.PeerManager
	Scheduler          types.Scheduler
	BlocksApplier      BlocksApplier
	UtxPool            types.UtxPool
	Scheme             proto.Scheme
	BlockAddedNotifier types.Handler
	InvRequester       types.InvRequester
	ScoreSender        types.Handler
	LoggableRunner     runner.LogRunner
	Time               types.Time
	Wallet             types.EmbeddedWallet
	MicroBlockCache    MicroBlockCache
}
