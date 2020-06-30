package state_fsm

import (
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

func newPeer(fsm FSM, p Peer, peers peer_manager.PeerManager) (FSM, Async, error) {
	err := peers.NewConnection(p)
	return fsm, nil, err
}

// TODO handle no peers
func peerError(fsm FSM, p Peer, peers peer_manager.PeerManager, _ error) (FSM, Async, error) {
	peers.Disconnect(p)
	return fsm, nil, nil
}

func noop(fsm FSM) (FSM, Async, error) {
	return fsm, nil, nil
}

func IsOutdate(period proto.Timestamp, lastBlock *proto.Block, tm types.Time) bool {
	curTime := proto.NewTimestampFromTime(tm.Now())
	return curTime-lastBlock.Timestamp > period
}

func handleScore(fsm FSM, info BaseInfo, p Peer, score *proto.Score) (FSM, Async, error) {
	err := info.peers.UpdateScore(p, score)
	if err != nil {
		return fsm, nil, err
	}

	myScore, err := info.storage.CurrentScore()
	if err != nil {
		return NewIdleFsm(info), nil, err
	}

	if score.Cmp(myScore) == 1 { // remote score > my score
		return NewIdleToSyncTransition(info, p)
	}
	return fsm, nil, nil
}

func sendScore(p Peer, storage state.State) {
	curScore, err := storage.CurrentScore()
	if err != nil {
		zap.S().Error(err)
		return
	}

	bts := curScore.Bytes()
	p.SendMessage(&proto.ScoreMessage{Score: bts})
}

// TODO send micro block
//func handleMineMicro(a FromBaseInfo, base BaseInfo, minedBlock *proto.Block, rest miner.MiningLimits, blocks ng.Blocks, keyPair proto.KeyPair) (FSM, Async, error) {
//	block, micro, rest, err := base.microMiner.Micro(rest, minedBlock, blocks, keyPair)
//	if err != nil {
//		return a, nil, err
//	}
//	base.
//	return a.FromBaseInfo()
//}
