package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"math/big"
	"net"
)

type MockStateManager struct {
	sig2Block map[crypto.Signature]*proto.Block
	Peers_    []proto.TCPAddr
}

func (a *MockStateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	return a.sig2Block[blockID], nil
}

func (a *MockStateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	panic("implement me")
}

func (a *MockStateManager) Header(block crypto.Signature) (*proto.BlockHeader, error) {
	panic("implement me")
}

func (a *MockStateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	panic("implement me")
}

func (a *MockStateManager) Height() (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) HeightToBlockID(height uint64) (crypto.Signature, error) {
	panic("implement me")
}

func (a *MockStateManager) WavesAddressesNumber() (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AddressesNumber(wavesonly bool) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AddBlock(block []byte) error {
	panic("implement me")
}

func (a *MockStateManager) AddNewBlocks(blocks [][]byte) error {
	panic("implement me")
}

func (a *MockStateManager) AddOldBlocks(blocks [][]byte) error {
	panic("implement me")
}

func (a *MockStateManager) RollbackToHeight(height uint64) error {
	panic("implement me")
}

func (a *MockStateManager) RollbackTo(removalEdge crypto.Signature) error {
	panic("implement me")
}

func (a *MockStateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	panic("implement me")
}

func (a *MockStateManager) CurrentScore() (*big.Int, error) {
	panic("implement me")
}

func (a *MockStateManager) EffectiveBalance(addr proto.Address, startHeigth, endHeight uint64) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) ValidateSingleTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	panic("implement me")
}

func (a *MockStateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	panic("implement me")
}

func (a *MockStateManager) ResetValidationList() {
	panic("implement me")
}

func (a *MockStateManager) SavePeers([]proto.TCPAddr) error {
	panic("implement me")
}

func (a *MockStateManager) Peers() ([]proto.TCPAddr, error) {
	return a.Peers_, nil
}

func (a *MockStateManager) Close() error {
	panic("implement me")
}

func (a *MockStateManager) AddBlocks(blocks [][]byte, initialisation bool) error {
	panic("implement me")
}

func (a *MockStateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	panic("implement me")
}

func (a *MockStateManager) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	panic("implement me")
}

func newMockStateWithGenesis() *MockStateManager {
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	block := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: sig,
		},
	}
	sig2Block := map[crypto.Signature]*proto.Block{sig: block}
	return &MockStateManager{
		sig2Block: sig2Block,
	}
}

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

func (*mockPeerManager) AskPeers() {
	panic("implement me")
}

func (*mockPeerManager) Disconnect(string) {
	panic("implement me")
}

func (*mockPeerManager) EachConnected(func(peer.Peer, *big.Int)) {
	panic("implement me")
}

func (*mockPeerManager) SpawnIncomingConnection(ctx context.Context, n net.Conn) {
	panic("implement me")
}

func NewMockPeerManagerWithDefaultPeer() (*mockPeerManager, string, *mock.Peer) {
	peerName := "peer"
	p := mock.NewPeer()
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

func (a *mockPeerManager) Connect(context.Context, proto.TCPAddr) error {
	panic("implement me")
}
