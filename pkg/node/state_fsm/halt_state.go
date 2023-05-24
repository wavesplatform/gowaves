package state_fsm

import (
	"context"

	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	haltSkipMessageList = proto.PeerMessageIDs{
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
		proto.ContentIDGetBlockIds,
	}
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
	zap.S().Debugf("started HaltTransition")
	info.peers.Close()
	zap.S().Debugf("started HaltTransition peers closed")
	err := info.storage.Close()
	if err != nil {
		return nil, nil, err
	}
	zap.S().Debugf("storage closed")
	info.skipMessageList.SetList(haltSkipMessageList)
	return &HaltState{
		baseInfo: info,
	}, nil, nil
}

func initHaltStateInFSM(_ *StateData, fsm *stateless.StateMachine, info BaseInfo) {
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
		Ignore(HaltEvent)
}
