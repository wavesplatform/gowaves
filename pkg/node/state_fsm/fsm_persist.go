package state_fsm

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Save transactions by address from temporary file into storage.
// Only read operations permitted.
type PersistFsm struct {
	BaseInfo
}

var (
	persistSkipMessageList = proto.PeerMessageIDs{
		proto.ContentIDGetSignatures,
		proto.ContentIDSignatures,
		proto.ContentIDGetBlock,
		proto.ContentIDBlock,
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

func (a *PersistFsm) NewPeer(p peer.Peer) (FSM, Async, error) {
	return newPeer(a, p, a.peers)
}

func (a *PersistFsm) PeerError(p peer.Peer, e error) (FSM, Async, error) {
	return peerError(a, p, a.peers, e)
}

func (a *PersistFsm) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	err := a.peers.UpdateScore(p, score)
	if err != nil {
		return a, nil, a.Errorf(proto.NewInfoMsg(err))
	}
	return a, nil, nil
}

func (a *PersistFsm) Block(p peer.Peer, block *proto.Block) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) BlockIDs(peer peer.Peer, ids []proto.BlockID) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) Task(t tasks.AsyncTask) (FSM, Async, error) {
	switch t.TaskType {
	case tasks.PersistComplete:
		return NewIdleFsm(a.BaseInfo), nil, nil
	default:
		return noop(a)
	}
}

func (a *PersistFsm) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	return noop(a)
}

func (a *PersistFsm) Halt() (FSM, Async, error) {
	return HaltTransition(a.BaseInfo)
}

func (a *PersistFsm) String() string {
	return "Persist"
}

func (a *PersistFsm) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func NewPersistTransition(info BaseInfo) (FSM, Async, error) {
	t := tasks.NewFuncTask(func(ctx context.Context, output chan tasks.AsyncTask) error {
		err := info.storage.PersistAddressTransactions()
		tasks.SendAsyncTask(output, tasks.AsyncTask{
			TaskType: tasks.PersistComplete,
		})
		return err
	}, tasks.PersistComplete)

	info.skipMessageList.SetList(persistSkipMessageList)
	return &PersistFsm{
		info,
	}, tasks.Tasks(t), nil
}
