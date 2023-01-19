package blocks_applier

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
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

func (a *MockStateManager) HeaderBytes(_ proto.BlockID) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) Map(func(state.State) error) error {
	panic("not impl")
}

func (a *MockStateManager) HeaderBytesByHeight(_ uint64) ([]byte, error) {
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

func (a *MockStateManager) Header(_ proto.BlockID) (*proto.BlockHeader, error) {
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

func (a *MockStateManager) HeightToBlockID(_ uint64) (proto.BlockID, error) {
	panic("implement me")
}

func (a *MockStateManager) WavesAddressesNumber() (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AddressesNumber(_ bool) (uint64, error) {
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

func (a *MockStateManager) RollbackTo(_ proto.BlockID) error {
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

func (a *MockStateManager) EffectiveBalanceStable(_ proto.Recipient, _, _ uint64) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) ValidateNextTx(_ proto.Transaction, _, _ uint64, _ proto.BlockVersion, _ bool) error {
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

func (a *MockStateManager) RetrieveEntries(_ proto.Recipient) ([]proto.DataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveEntry(_ proto.Recipient, _ string) (proto.DataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveIntegerEntry(_ proto.Recipient, _ string) (*proto.IntegerDataEntry, error) {
	panic("implement me")
}
func (a *MockStateManager) RetrieveBooleanEntry(_ proto.Recipient, _ string) (*proto.BooleanDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveStringEntry(_ proto.Recipient, _ string) (*proto.StringDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) RetrieveBinaryEntry(_ proto.Recipient, _ string) (*proto.BinaryDataEntry, error) {
	panic("implement me")
}

func (a *MockStateManager) TransactionHeightByID(_ []byte) (proto.Height, error) {
	panic("implement me")
}

func (a *MockStateManager) NewAddrTransactionsIterator(_ proto.WavesAddress) (state.TransactionIterator, error) {
	panic("implement me")
}

func (a *MockStateManager) TransactionByID(_ []byte) (proto.Transaction, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetIsSponsored(_ crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) AssetInfo(_ crypto.Digest) (*proto.AssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) FullAssetInfo(_ crypto.Digest) (*proto.FullAssetInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) ScriptInfoByAccount(_ proto.Recipient) (*proto.ScriptInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) ScriptInfoByAsset(_ crypto.Digest) (*proto.ScriptInfo, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActiveLeasing(_ crypto.Digest) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) InvokeResultByID(_ crypto.Digest) (*proto.ScriptResult, error) {
	panic("implement me")
}

func (a *MockStateManager) ProvidesExtendedApi() (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) ProvidesStateHashes() (bool, error) {
	panic("not implemented")
}

func (a *MockStateManager) StateHashAtHeight(_ uint64) (*proto.StateHash, error) {
	panic("not implemented")
}

func (a *MockStateManager) IsNotFound(_ error) bool {
	panic("implement me")
}

func (a *MockStateManager) Close() error {
	panic("implement me")
}

func (a *MockStateManager) AddBlocks(_ [][]byte, _ bool) error {
	panic("implement me")
}

func (a *MockStateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	panic("implement me")
}

func (a *MockStateManager) AddrByAlias(_ proto.Alias) (proto.WavesAddress, error) {
	panic("implement me")
}

func (a *MockStateManager) FullWavesBalance(_ proto.Recipient) (*proto.FullWavesBalance, error) {
	panic("implement me")
}

func (a *MockStateManager) AccountBalance(_ proto.Recipient, _ []byte) (uint64, error) {
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
func (a *MockStateManager) AddDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	var out *proto.Block
	var err error
	for _, b := range blocks {
		if out, err = a.AddDeserializedBlock(b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (a *MockStateManager) BlockBytes(_ proto.BlockID) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) BlockBytesByHeight(_ proto.Height) ([]byte, error) {
	panic("implement me")
}

func (a *MockStateManager) VotesNumAtHeight(_ int16, _ proto.Height) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) VotesNum(_ int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActivated(_ int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) IsActiveAtHeight(_ int16, _ proto.Height) (bool, error) {
	panic("not implemented")
}

func (a *MockStateManager) ActivationHeight(_ int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApproved(_ int16) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) IsApprovedAtHeight(_ int16, _ uint64) (bool, error) {
	panic("implement me")
}

func (a *MockStateManager) ApprovalHeight(_ int16) (uint64, error) {
	panic("implement me")
}

func (a *MockStateManager) AllFeatures() ([]int16, error) {
	panic("implement me")
}

func (a *MockStateManager) StartProvidingExtendedApi() error {
	panic("implement me")
}

func (a *MockStateManager) HitSourceAtHeight(_ proto.Height) ([]byte, error) {
	panic("not implemented")
}
