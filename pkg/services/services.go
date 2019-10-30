package services

import (
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Services struct {
	State              state.State
	Peers              peer_manager.PeerManager
	Scheduler          types.Scheduler
	BlockApplier       types.BlockApplier
	UtxPool            types.UtxPool
	Scheme             proto.Scheme
	BlockAddedNotifier types.Handler
}
