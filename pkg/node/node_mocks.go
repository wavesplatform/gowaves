package node

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"math/big"
	"net"
	"sync"
)

func notFound() state.StateError {
	return state.NewStateError(0, keyvalue.ErrNotFound)
}

type AddBlockFunc func(a *MockStateManager, block []byte) (*proto.Block, error)

type MockStateManager struct {
	state           []*proto.Block
	sig2Block       map[crypto.Signature]*proto.Block
	Peers_          []proto.TCPAddr
	blockIDToHeight map[crypto.Signature]proto.Height
	addBlockFunc    AddBlockFunc
}

func DefaultAddBlockFunc(a *MockStateManager, block []byte) (*proto.Block, error) {
	b := &proto.Block{}
	err := b.UnmarshalBinary(block)
	if err != nil {
		return nil, err
	}
	a.addBlock(b)
	return b, nil
}

func NewMockStateManager(blocks ...*proto.Block) *MockStateManager {
	return NewMockStateManagerWithAddBlock(DefaultAddBlockFunc, blocks...)
}

func NewMockStateManagerWithAddBlock(addBlockFunc AddBlockFunc, blocks ...*proto.Block) *MockStateManager {
	m := &MockStateManager{
		blockIDToHeight: make(map[crypto.Signature]proto.Height),
		addBlockFunc:    addBlockFunc,
	}
	for _, b := range blocks {
		m.addBlock(b)
	}
	return m
}

func (a *MockStateManager) addBlock(block *proto.Block) {
	if (block.BlockSignature == crypto.Signature{}) {
		panic("empty signature")
	}
	if _, ok := a.blockIDToHeight[block.BlockSignature]; ok {
		panic("duplicate block")
	}
	a.state = append(a.state, block)
	a.blockIDToHeight[block.BlockSignature] = proto.Height(len(a.state))
}

func (a *MockStateManager) Block(blockID crypto.Signature) (*proto.Block, error) {
	if block, ok := a.sig2Block[blockID]; ok {
		return block, nil
	}
	return nil, notFound()
}

func (a *MockStateManager) BlockByHeight(height proto.Height) (*proto.Block, error) {
	if height > proto.Height(len(a.state)) {
		return nil, notFound()
	}
	return a.state[height-1], nil
}

func (a *MockStateManager) Header(block crypto.Signature) (*proto.BlockHeader, error) {
	panic("implement me")
}

func (a *MockStateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	panic("implement me")
}

func (a *MockStateManager) Height() (proto.Height, error) {
	return proto.Height(len(a.state)), nil
}

func (a *MockStateManager) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	if height, ok := a.blockIDToHeight[blockID]; ok {
		return height, nil
	}
	return 0, notFound()
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

func (a *MockStateManager) AddBlock(block []byte) (*proto.Block, error) {
	return a.addBlockFunc(a, block)
}

func (a *MockStateManager) Mutex() *sync.RWMutex {
	return &sync.RWMutex{}
}

func (a *MockStateManager) AddNewBlocks(blocks [][]byte) error {
	for _, bts := range blocks {
		block := proto.Block{}
		err := block.UnmarshalBinary(bts)
		if err != nil {
			return err
		}
		a.addBlock(&block)
	}
	return nil
}

func (a *MockStateManager) AddOldBlocks(blocks [][]byte) error {
	panic("implement me")
}

func (a *MockStateManager) RollbackToHeight(height uint64) error {
	if height > proto.Height(len(a.state)) {
		return notFound()
	}

	for i := proto.Height(len(a.state)); i > height; i-- {
		block := a.state[len(a.state)-1]
		a.state = a.state[:len(a.state)-1]
		delete(a.blockIDToHeight, block.BlockSignature)
	}
	return nil
}

func (a *MockStateManager) RollbackTo(removalEdge crypto.Signature) error {
	panic("implement me")
}

func (a *MockStateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	if height > uint64(len(a.state)) {
		return nil, notFound()
	}
	score := big.NewInt(0)
	for _, b := range a.state[:height] {
		n, err := state.CalculateScore(b.NxtConsensus.BaseTarget)
		if err != nil {
			panic(err)
		}
		score.Add(score, n)
	}
	return score, nil
}

func (a *MockStateManager) CurrentScore() (*big.Int, error) {
	return a.ScoreAtHeight(proto.Height(len(a.state)))
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
