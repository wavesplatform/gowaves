package state_fsm

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HaltFSM struct {
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

func (a HaltFSM) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) (FSM, Async, error) {
	return noop(a)
}

func (a HaltFSM) Signatures(peer peer.Peer, sigs []crypto.Signature) (FSM, Async, error) {
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

func HaltTransition(info BaseInfo) (FSM, Async, error) {
	info.peers.Close()
	locked := info.storage.Mutex().Lock()
	info.storage.Close()
	locked.Unlock()
	return HaltFSM{}, nil, nil
}
