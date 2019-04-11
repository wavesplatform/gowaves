package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"math/big"
)

type mockPeer struct {
	messages []proto.Message
}

func (a *mockPeer) RemoteAddr() proto.TCPAddr {
	panic("implement me")
}

func (*mockPeer) Direction() peer.Direction {
	panic("implement me")
}

func (*mockPeer) Close() error {
	panic("implement me")
}

func (a *mockPeer) SendMessage(m proto.Message) {
	a.messages = append(a.messages, m)
}

func (*mockPeer) ID() string {
	return "mocked"
}

func (*mockPeer) Connection() conn.Connection {
	panic("implement me")
}

func (*mockPeer) Handshake() proto.Handshake {
	panic("implement me")
}

type mockStateManager struct {
	sig2Block map[crypto.Signature]*proto.Block
}

func (a *mockStateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	return a.sig2Block[blockID], nil
}

func (a *mockStateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	panic("implement me")
}

func (a *mockStateManager) Height() (uint64, error) {
	panic("implement me")
}

func (a *mockStateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	panic("implement me")
}

func (a *mockStateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	panic("implement me")
}

func (a *mockStateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	panic("implement me")
}

func (a *mockStateManager) AddressesNumber() (uint64, error) {
	panic("implement me")
}

func (a *mockStateManager) AddBlock(block []byte) error {
	panic("implement me")
}

func (a *mockStateManager) AddNewBlocks(blocks [][]byte) error {
	panic("implement me")
}

func (a *mockStateManager) AddOldBlocks(blocks [][]byte) error {
	panic("implement me")
}

func (a *mockStateManager) RollbackToHeight(height uint64) error {
	panic("implement me")
}

func (a *mockStateManager) RollbackTo(removalEdge crypto.Signature) error {
	panic("implement me")
}

func (a *mockStateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	panic("implement me")
}

func (a *mockStateManager) CurrentScore() (*big.Int, error) {
	panic("implement me")
}

func (a *mockStateManager) SavePeers([]state.KnownPeer) error {
	panic("implement me")
}

func (a *mockStateManager) Peers() ([]state.KnownPeer, error) {
	panic("implement me")
}

func (a *mockStateManager) Close() error {
	panic("implement me")
}

func (a *mockStateManager) AddBlocks(blocks [][]byte, initialisation bool) error {
	panic("implement me")
}

//func (a *mockStateManager) AddBlock(block *proto.Block) error {
//	panic("implement me")
//}

func newMockStateWithGenesis() *mockStateManager {
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	block := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: sig,
		},
	}
	sig2Block := map[crypto.Signature]*proto.Block{sig: block}
	return &mockStateManager{
		sig2Block: sig2Block,
	}
}

//
//func (a *mockStateManager) GetBlock(blockID crypto.Signature) (*proto.Block, error) {
//	return a.sig2Block[blockID], nil
//}

type mockPeerManager struct {
	connected map[string]peer.Peer
}

func (a *mockPeerManager) PeerWithHighestScore() (peer.Peer, *big.Int, bool) {
	panic("implement me")
}

func (a *mockPeerManager) Close() {
	panic("implement me")
}

func (*mockPeerManager) Banned(unique string) bool {
	panic("implement me")
}

func (*mockPeerManager) SpawnOutgoingConnections(ctx context.Context) {
	panic("implement me")
}

func (*mockPeerManager) UpdateKnownPeers([]proto.TCPAddr) error {
	panic("implement me")
}

func (*mockPeerManager) UpdateScore(string, *big.Int) {
	panic("implement me")
}

func (*mockPeerManager) AddConnected(p peer.Peer) {
	panic("implement me")
}

func NewMockPeerWithDefaultPeer() (*mockPeerManager, string, *mockPeer) {
	peerName := "peer"
	p := &mockPeer{}
	m := make(map[string]peer.Peer)
	m[peerName] = p

	return &mockPeerManager{
		connected: m,
	}, peerName, p
}

func (a *mockPeerManager) Connected(id string) (peer.Peer, bool) {
	p, ok := a.connected[id]
	return p, ok
}

func (a *mockPeerManager) KnownPeers() ([]proto.TCPAddr, error) {
	panic("implement me")
}
