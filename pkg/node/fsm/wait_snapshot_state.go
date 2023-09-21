package fsm

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

type WaitSnapshotState struct {
	baseInfo   BaseInfo
	blockCache *blockStatesCache
}

func NewWaitSnapshotState(baseInfo BaseInfo, blockCache *blockStatesCache) *WaitSnapshotState {
	baseInfo.syncPeer.Clear()
	return &WaitSnapshotState{
		baseInfo:   baseInfo,
		blockCache: blockCache,
	}
}

func (a *WaitSnapshotState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func (a *WaitSnapshotState) String() string {
	return NGLightStateName
}

func (a *WaitSnapshotState) Block(peer peer.Peer, block *proto.Block) (State, Async, error) {
	ok, err := a.baseInfo.snapshotApplier.BlockSnapshotExists(a.baseInfo.storage, block.BlockID())
	if err != nil {
		return a, nil, a.Errorf(errors.Wrapf(err, "peer '%s'", peer.ID()))
	}
	if ok {
		return a, nil, a.Errorf(proto.NewInfoMsg(errors.Errorf("Block '%s' already exists", block.BlockID().String())))
	}
	//metrics.FSMKeyBlockReceived("ng", block, peer.Handshake().NodeName)

	top := a.baseInfo.storage.TopBlock()
	if top.BlockID() != block.Parent { // does block refer to last block
		zap.S().Named(logging.FSMNamespace).Debugf(
			"[%s] Key-block '%s' has parent '%s' which is not the top block '%s'",
			a, block.ID.String(), block.Parent.String(), top.ID.String(),
		)
		var blockFromCache *proto.Block
		if blockFromCache, ok = a.blockCache.Get(block.Parent); ok {
			zap.S().Named(logging.FSMNamespace).Debugf("[%s] Re-applying block '%s' from cache",
				a, blockFromCache.ID.String())
			if err = a.rollbackToStateFromCache(blockFromCache); err != nil {
				return a, nil, a.Errorf(err)
			}
		}
	}
	a.blockCache.Clear()
	a.blockCache.AddBlockState(block)
}

func (a *WaitSnapshotState) BlockSnapshot(peer peer.Peer, blockID proto.BlockID, snapshots state.TransactionSnapshot) (State, Async, error) {
	// check if this snapshot for our block
	if _, ok := a.blockCache.Get(blockID); !ok {
		return newNGLightState(a.baseInfo), nil, a.Errorf(errors.Errorf("Snapshot for the block '%s' doestn match", blockID))
	}

	_, err := a.baseInfo.snapshotApplier.Apply(a.baseInfo.storage, []state.TransactionSnapshot{snapshots}, []proto.BlockID{blockID})
	if err != nil {
		//metrics.FSMKeyBlockDeclined("ng", block, err)
		return a, nil, a.Errorf(errors.Wrapf(err, "peer '%s'", peer.ID()))
	}

	metrics.FSMKeyBlockApplied("ng", block)
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Handle received key block message: block '%s' applied to state",
		a, block.BlockID())

	a.baseInfo.scheduler.Reschedule()
	a.baseInfo.actions.SendScore(a.baseInfo.storage)
	a.baseInfo.CleanUtx()

	return newNGLightState(a.baseInfo), nil, nil
}
