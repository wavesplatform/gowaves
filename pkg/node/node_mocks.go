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
	id2Block        map[proto.BlockID]*proto.Block
	Peers_          []proto.TCPAddr
	blockIDToHeight map[proto.BlockID]proto.Height
}

func (a *MockStateManager) HeaderBytes(blockID proto.BlockID) ([]byte, error) {
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
		blockIDToHeight: make(map[proto.BlockID]proto.Height),
	}
	for _, b := range blocks {
		if _, err := m.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (a *MockStateManager) TopBlock() *proto.Block {
	if len(a.state) == 0 {
		panic("no top block")
	}
	return a.state[len(a.state)-1]
}

func (a *MockStateManager) Block(blockID proto.BlockID) (*proto.Block, error) {
	if block, ok := a.id2Block[blockID]; ok {
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

func (a *MockStateManager) Header(block proto.BlockID) (*proto.BlockHeader, error) {
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

func (a *MockStateManager) Height() (proto.Height, error) {
	return proto.Height(len(a.state)), nil
}

func (a *MockStateManager) BlockIDToHeight(blockID proto.BlockID) (uint64, error) {
	if height, ok := a.blockIDToHeight[blockID]; ok {
		return height, nil
	}
	return 0, notFound()
}

func (a *MockStateManager) HeightToBlockID(height uint64) (proto.BlockID, error) {
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
		delete(a.blockIDToHeight, block.BlockID())
	}
	return nil
}

func (a *MockStateManager) RollbackTo(removalEdge proto.BlockID) error {
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

func (a *MockStateManager) EffectiveBalanceStable(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, version proto.BlockVersion, vrf []byte) error {
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

func (a *MockStateManager) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	panic("implement me")
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

func (a *MockStateManager) TransactionHeightByID(id []byte) (proto.Height, error) {
	panic("implement me")
}

func (a *MockStateManager) NewAddrTransactionsIterator(addr proto.Address) (state.TransactionIterator, error) {
	panic("implement me")
}

func (a *MockStateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetIsSponsored(assetID crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) FullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) ScriptInfoByAsset(assetID crypto.Digest) (*proto.ScriptInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActiveLeasing(leaseID crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error) {
	panic("implement me")
}

func (a *MockStateManager) ProvidesExtendedApi() (bool, error) {
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

func (a *MockStateManager) AddrByAlias(alias proto.Alias) (proto.Address, error) {
	panic("implement me")
}

func (a *MockStateManager) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	panic("implement me")
}

func (a *MockStateManager) AccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	if _, ok := a.blockIDToHeight[block.BlockID()]; ok {
		panic("duplicate block")
	}
	a.state = append(a.state, block)
	a.blockIDToHeight[block.BlockID()] = proto.Height(len(a.state))
	return block, nil
}
func (a *MockStateManager) AddNewDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	var out *proto.Block
	var err error
	for _, b := range blocks {
		if out, err = a.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *MockStateManager) AddOldDeserializedBlocks([]*proto.Block) error {
	panic("implement me")
}

func (a *MockStateManager) BlockBytes(blockID proto.BlockID) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) BlockBytesByHeight(height proto.Height) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) VotesNumAtHeight(featureID int16, height proto.Height) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) VotesNum(featureID int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActivated(featureID int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	panic("not implemented")
}

func (a *MockStateManager) ActivationHeight(featureID int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApproved(featureID int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApprovedAtHeight(featureID int16, height uint64) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) ApprovalHeight(featureID int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AllFeatures() ([]int16, error) {
	panic("implement me")
}

func (a *MockStateManager) StartProvidingExtendedApi() error {
	panic("implement me")
}

func (a *MockStateManager) HitSourceAtHeight(height proto.Height) ([]byte, error) {
	panic("not implemented")
}

func (a *MockStateManager) BlockVRF(header *proto.BlockHeader, height proto.Height) ([]byte, error) {
	return nil, nil
}

func newMockStateWithGenesis() *MockStateManager {
	sig, _ := crypto.NewSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa")
	block := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: sig,
		},
	}
	id := proto.NewBlockIDFromSignature(sig)
	id2Block := map[proto.BlockID]*proto.Block{id: block}
	return &MockStateManager{
		id2Block: id2Block,
	}
}

type mockPeerManager struct {
	connected map[peer.Peer]struct{}
}

func (a *mockPeerManager) Spawned() []proto.IpPort {
	panic("implement me")
}

func (a *mockPeerManager) IsSuspended(peer.Peer) bool {
	panic("implement me")
}

func (a *mockPeerManager) Suspend(peer.Peer, string) {
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
