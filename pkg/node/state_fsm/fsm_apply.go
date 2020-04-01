package state_fsm

//
//import (
//	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
//	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
//	"github.com/wavesplatform/gowaves/pkg/proto"
//)
//
//type blockFromPeer struct {
//	Peer  peer.Peer
//	Block *proto.Block
//}
//
//type ApplyFSM struct {
//	BaseInfo
//
//	receivedBlocks []blockFromPeer
//}
//
//func (a *ApplyFSM) NewPeer(p peer.Peer) (FSM, Async, error) {
//	return newPeer(a, p, a.peers)
//}
//
//func (a *ApplyFSM) PeerError(p peer.Peer, e error) (FSM, Async, error) {
//	return peerError(a, p, a.peers, e)
//}
//
//func (a *ApplyFSM) Score(p peer.Peer, score *proto.Score) (FSM, Async, error) {
//	err := a.peers.UpdateScore(p, score)
//	return a, nil, err
//}
//
//func (a *ApplyFSM) Block(peer peer.Peer, block *proto.Block) (FSM, Async, error) {
//	a.receivedBlocks = append(a.receivedBlocks, blockFromPeer{
//		Peer:  peer,
//		Block: block,
//	})
//	return a, nil, nil
//}
//
//func (a *ApplyFSM) MinedBlock(block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair) (FSM, Async, error) {
//	panic("implement me")
//}
//
//func (a *ApplyFSM) BlockIDs(peer.Peer, []proto.BlockID) (FSM, Async, error) {
//	panic("implement me")
//}
//
//func (a *ApplyFSM) Task(task tasks.AsyncTask) (FSM, Async, error) {
//	panic("implement me")
//}
//
//func (a *ApplyFSM) MicroBlock(p peer.Peer, micro *proto.MicroBlock) (FSM, Async, error) {
//	panic("implement me")
//}
//
//func (a *ApplyFSM) MicroBlockInv(p peer.Peer, inv *proto.MicroBlockInv) (FSM, Async, error) {
//	panic("implement me")
//}
//
//func (a *ApplyFSM) Halt() (FSM, Async, error) {
//	panic("implement me")
//}
