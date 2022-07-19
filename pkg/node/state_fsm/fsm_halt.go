package state_fsm

import (
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type HaltFSM struct {
}

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

func (a HaltFSM) Transaction(p peer.Peer, t proto.Transaction) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) Halt() (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) NewPeer(p peer.Peer) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) PeerError(peer.Peer, error) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) BlockIDs(peer peer.Peer, sigs []proto.BlockID) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) Task(task tasks.AsyncTask) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) String() string {
	return "Halt"
}

func (a HaltFSM) Errorf(err error) error {
	return fsmErrorf(a, err)
}

func HaltTransition(info BaseInfo) (FSM, Async, error) {
	zap.S().Debugf("started HaltTransition ")
	info.peers.Close()
	zap.S().Debugf("started HaltTransition peers closed")
	err := info.storage.Close()
	if err != nil {
		return nil, nil, err
	}
	zap.S().Debugf("storage closed")
	info.skipMessageList.SetList(haltSkipMessageList)
	return HaltFSM{}, nil, nil
}
