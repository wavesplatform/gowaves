package node

import (
	"context"
	"math/big"
	"net"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

func notFound() state.StateError {
	return state.NewStateError(state.NotFoundError, proto.ErrNotFound)
}

type MockStateManager struct {
	state           []*proto.Block
	sig2Block       map[crypto.Signature]*proto.Block
	Peers_          []proto.TCPAddr
	blockIDToHeight map[crypto.Signature]proto.Height
}

func (a *MockStateManager) HeaderBytes(blockID crypto.Signature) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) HeaderBytesByHeight(height uint64) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) AddBlock([]byte) (*proto.Block, error) {
	panic("implement me")
}

func NewMockStateManager(blocks ...*proto.Block) (*MockStateManager, error) {
	m := &MockStateManager{
		blockIDToHeight: make(map[crypto.Signature]proto.Height),
	}
	for _, b := range blocks {
		if _, err := m.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return m, nil
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

func (a *MockStateManager) NewestHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	panic("implement me")
}

func (a *MockStateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	rs, err := a.BlockByHeight(height)
	if err != nil {
		return nil, err
	}
	return &rs.BlockHeader, nil
}

func (a *MockStateManager) AddingBlockHeight() (proto.Height, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestHeight() (proto.Height, error) {
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

func (a *MockStateManager) Mutex() *lock.RwMutex {
	return lock.NewRwMutex(&sync.RWMutex{})
}

func (a *MockStateManager) AddNewBlocks(blocks [][]byte) error {
	panic("implement me")
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

func (a *MockStateManager) EffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) ValidateSingleTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	panic("implement me")
}

func (a *MockStateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error {
	panic("implement me")
}

func (a *MockStateManager) ResetValidationList() {

}

func (a *MockStateManager) SavePeers([]proto.TCPAddr) error {
	panic("implement me")
}

func (a *MockStateManager) Peers() ([]proto.TCPAddr, error) {
	return a.Peers_, nil
}

func (a *MockStateManager) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	panic("implement me")
}
func (a *MockStateManager) RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestTransactionHeightByID(id []byte) (proto.Height, error) {
	panic("implement me")
}

func (a *MockStateManager) TransactionHeightByID(id []byte) (proto.Height, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	panic("implement me")
}

func (a *MockStateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetIsSponsored(assetID crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) IsNotFound(err error) bool {
	panic("implement me")
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

func (a *MockStateManager) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	panic("implement me")
}

func (a *MockStateManager) AddrByAlias(alias proto.Alias) (proto.Address, error) {
	panic("implement me")
}

func (a *MockStateManager) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	if _, ok := a.blockIDToHeight[block.BlockSignature]; ok {
		panic("duplicate block")
	}
	a.state = append(a.state, block)
	a.blockIDToHeight[block.BlockSignature] = proto.Height(len(a.state))
	return block, nil
}
func (a *MockStateManager) AddNewDeserializedBlocks(blocks []*proto.Block) error {
	for _, b := range blocks {
		if _, err := a.AddDeserializedBlock(b); err != nil {
			return err
		}
	}
	return nil
}

func (a *MockStateManager) AddOldDeserializedBlocks([]*proto.Block) error {
	panic("implement me")
}

func (a *MockStateManager) BlockBytes(blockID crypto.Signature) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) BlockBytesByHeight(height proto.Height) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActivated(featureID int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) ActivationHeight(featureID int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApproved(featureID int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) ApprovalHeight(featureID int16) (uint64, error) {
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
	connected map[peer.Peer]struct{}
}

func (a *mockPeerManager) IsSuspended(peer.Peer) bool {
	panic("implement me")
}

func (a *mockPeerManager) Suspend(peer.Peer) {
	panic("implement me")
}

func (a *mockPeerManager) Suspended() []string {
	panic("implement me")
}

func (a *mockPeerManager) Disconnect(peer.Peer) {
	panic("implement me")
}

func (a *mockPeerManager) Block(peer.Peer) {
	panic("implement me")
}

func (a *mockPeerManager) UpdateScore(p peer.Peer, score *proto.Score) {
	panic("implement me")
}

func (a *mockPeerManager) Score(p peer.Peer) (*proto.Score, error) {
	panic("implement me")
}

func (a *mockPeerManager) PeerWithHighestScore() (peer.Peer, *big.Int, bool) {
	panic("implement me")
}

func (a *mockPeerManager) Close() {
	panic("implement me")
}

func (*mockPeerManager) Banned(peer.Peer) bool {
	panic("implement me")
}

func (*mockPeerManager) SpawnOutgoingConnections(ctx context.Context) {
	panic("implement me")
}

func (*mockPeerManager) UpdateKnownPeers([]proto.TCPAddr) error {
	panic("implement me")
}

func (*mockPeerManager) AddConnected(p peer.Peer) {
	panic("implement me")
}

func (*mockPeerManager) AskPeers() {
	panic("implement me")
}

func (*mockPeerManager) EachConnected(func(peer.Peer, *big.Int)) {
	panic("implement me")
}

func (*mockPeerManager) SpawnIncomingConnection(ctx context.Context, n net.Conn) error {
	panic("implement me")
}

func NewMockPeerManagerWithDefaultPeer() (*mockPeerManager, *mock.Peer) {
	p := mock.NewPeer()
	m := make(map[peer.Peer]struct{})
	m[p] = struct{}{}

	return &mockPeerManager{
		connected: m,
	}, p
}

func (a *mockPeerManager) Connected(p peer.Peer) (peer.Peer, bool) {
	_, ok := a.connected[p]
	return p, ok
}

func (a *mockPeerManager) KnownPeers() ([]proto.TCPAddr, error) {
	panic("implement me")
}

func (a *mockPeerManager) Connect(context.Context, proto.TCPAddr) error {
	panic("implement me")
}
