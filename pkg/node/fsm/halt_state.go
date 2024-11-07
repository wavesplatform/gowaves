package fsm

import (
	"context"
	stderrs "errors"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HaltState struct {
	baseInfo BaseInfo
}

func (a *HaltState) String() string {
	return HaltStateName
}

func (a *HaltState) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func newHaltState(info BaseInfo) (State, Async, error) {
	zap.S().Named(logging.FSMNamespace).Debugf("[Halt] Entered the Halt state")
	var errs []error
	if err := info.peers.Close(); err != nil {
		errs = append(errs, errors.Wrap(err, "failed to close peers"))
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[Halt] Peers closed")
	err := info.storage.Close()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to close storage"))
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[Halt] Storage closed")
	info.syncPeer.Clear()
	return &HaltState{
		baseInfo: info,
	}, nil, stderrs.Join(errs...)
}

func initHaltStateInFSM(_ *StateData, fsm *stateless.StateMachine, info BaseInfo) {
	haltSkipMessageList := proto.PeerMessageIDs{
		proto.ContentIDGetPeers,
		proto.ContentIDPeers,
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
		proto.ContentIDScore,
		proto.ContentIDTransaction,
		proto.ContentIDInvMicroblock,
		proto.ContentIDCheckpoint,
		proto.ContentIDMicroblockRequest,
		proto.ContentIDMicroblock,
		proto.ContentIDPBBlock,
		proto.ContentIDPBMicroBlock,
		proto.ContentIDPBTransaction,
		proto.ContentIDGetBlockIDs,
		proto.ContentIDBlockSnapshot,
		proto.ContentIDGetBlockSnapshot,
		proto.ContentIDMicroBlockSnapshot,
		proto.ContentIDMicroBlockSnapshotRequest,
	}
	fsm.Configure(HaltStateName).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			info.skipMessageList.SetList(haltSkipMessageList)
			return nil
		}).
		Ignore(ScoreEvent).
		Ignore(BlockEvent).
		Ignore(MinedBlockEvent).
		Ignore(BlockIDsEvent).
		Ignore(TaskEvent).
		Ignore(MicroBlockEvent).
		Ignore(MicroBlockInvEvent).
		Ignore(TransactionEvent).
		Ignore(StopSyncEvent).
		Ignore(StartMiningEvent).
		Ignore(ChangeSyncPeerEvent).
		Ignore(StopMiningEvent).
		Ignore(HaltEvent).
		Ignore(BlockSnapshotEvent).
		Ignore(MicroBlockSnapshotEvent)
}
