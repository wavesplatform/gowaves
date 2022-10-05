package state

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type ThreadSafeReadWrapper struct {
	mu *sync.RWMutex
	s  StateInfo
}

func (a *ThreadSafeReadWrapper) HitSourceAtHeight(height proto.Height) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.HitSourceAtHeight(height)
}

func (a *ThreadSafeReadWrapper) MapR(f func(StateInfo) (interface{}, error)) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return f(a.s)
}

func (a *ThreadSafeReadWrapper) TopBlock() *proto.Block {
	return a.s.TopBlock()
}

func (a *ThreadSafeReadWrapper) Block(blockID proto.BlockID) (*proto.Block, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.Block(blockID)
}

func (a *ThreadSafeReadWrapper) BlockByHeight(height proto.Height) (*proto.Block, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.BlockByHeight(height)
}

func (a *ThreadSafeReadWrapper) Header(blockID proto.BlockID) (*proto.BlockHeader, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.Header(blockID)
}

func (a *ThreadSafeReadWrapper) HeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.HeaderByHeight(height)
}

func (a *ThreadSafeReadWrapper) Height() (proto.Height, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.Height()
}

func (a *ThreadSafeReadWrapper) BlockIDToHeight(blockID proto.BlockID) (proto.Height, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.BlockIDToHeight(blockID)
}

func (a *ThreadSafeReadWrapper) HeightToBlockID(height proto.Height) (proto.BlockID, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.HeightToBlockID(height)
}

func (a *ThreadSafeReadWrapper) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.FullWavesBalance(account)
}

func (a *ThreadSafeReadWrapper) EffectiveBalance(account proto.Recipient, startHeight, endHeight proto.Height) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.EffectiveBalance(account, startHeight, endHeight)
}

func (a *ThreadSafeReadWrapper) WavesBalance(account proto.Recipient) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.WavesBalance(account)
}

func (a *ThreadSafeReadWrapper) AssetBalance(account proto.Recipient, asset proto.AssetID) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.AssetBalance(account, asset)
}

func (a *ThreadSafeReadWrapper) WavesAddressesNumber() (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.WavesAddressesNumber()
}

func (a *ThreadSafeReadWrapper) ScoreAtHeight(height proto.Height) (*big.Int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ScoreAtHeight(height)
}

func (a *ThreadSafeReadWrapper) CurrentScore() (*big.Int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.CurrentScore()
}

func (a *ThreadSafeReadWrapper) BlockchainSettings() (*settings.BlockchainSettings, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.BlockchainSettings()
}

func (a *ThreadSafeReadWrapper) VotesNum(featureID int16) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.VotesNum(featureID)
}

func (a *ThreadSafeReadWrapper) VotesNumAtHeight(featureID int16, height proto.Height) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.VotesNumAtHeight(featureID, height)
}

func (a *ThreadSafeReadWrapper) IsActivated(featureID int16) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.IsActivated(featureID)
}

func (a *ThreadSafeReadWrapper) IsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.IsActiveAtHeight(featureID, height)
}

func (a *ThreadSafeReadWrapper) ActivationHeight(featureID int16) (proto.Height, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ActivationHeight(featureID)
}

func (a *ThreadSafeReadWrapper) IsApproved(featureID int16) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.IsApproved(featureID)
}

func (a *ThreadSafeReadWrapper) IsApprovedAtHeight(featureID int16, height proto.Height) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.IsApprovedAtHeight(featureID, height)
}

func (a *ThreadSafeReadWrapper) ApprovalHeight(featureID int16) (proto.Height, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ApprovalHeight(featureID)
}

func (a *ThreadSafeReadWrapper) AllFeatures() ([]int16, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.AllFeatures()
}

func (a *ThreadSafeReadWrapper) EstimatorVersion() (int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.EstimatorVersion()
}

func (a *ThreadSafeReadWrapper) AddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.AddrByAlias(alias)
}

func (a *ThreadSafeReadWrapper) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveEntries(account)
}

func (a *ThreadSafeReadWrapper) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveEntry(account, key)
}

func (a *ThreadSafeReadWrapper) RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveIntegerEntry(account, key)
}

func (a *ThreadSafeReadWrapper) RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveBooleanEntry(account, key)
}

func (a *ThreadSafeReadWrapper) RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveStringEntry(account, key)
}

func (a *ThreadSafeReadWrapper) RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.RetrieveBinaryEntry(account, key)
}

func (a *ThreadSafeReadWrapper) TransactionByID(id []byte) (proto.Transaction, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.TransactionByID(id)
}

func (a *ThreadSafeReadWrapper) TransactionByIDWithStatus(id []byte) (proto.Transaction, bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.TransactionByIDWithStatus(id)
}

func (a *ThreadSafeReadWrapper) TransactionHeightByID(id []byte) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.TransactionHeightByID(id)
}

func (a *ThreadSafeReadWrapper) NewAddrTransactionsIterator(addr proto.Address) (TransactionIterator, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.NewAddrTransactionsIterator(addr)
}

func (a *ThreadSafeReadWrapper) AssetIsSponsored(assetID proto.AssetID) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.AssetIsSponsored(assetID)
}

func (a *ThreadSafeReadWrapper) AssetInfo(assetID proto.AssetID) (*proto.AssetInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.AssetInfo(assetID)
}

func (a *ThreadSafeReadWrapper) FullAssetInfo(assetID proto.AssetID) (*proto.FullAssetInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.FullAssetInfo(assetID)
}

func (a *ThreadSafeReadWrapper) NFTList(account proto.Recipient, limit uint64, afterAssetID *proto.AssetID) ([]*proto.FullAssetInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.NFTList(account, limit, afterAssetID)
}

func (a *ThreadSafeReadWrapper) ScriptBasicInfoByAccount(account proto.Recipient) (*proto.ScriptBasicInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ScriptBasicInfoByAccount(account)
}

func (a *ThreadSafeReadWrapper) ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ScriptInfoByAccount(account)
}

func (a *ThreadSafeReadWrapper) ScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ScriptInfoByAsset(assetID)
}

func (a *ThreadSafeReadWrapper) NewestScriptByAccount(recipient proto.Recipient) (*ast.Tree, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.NewestScriptByAccount(recipient)
}

func (a *ThreadSafeReadWrapper) NewestScriptBytesByAccount(recipient proto.Recipient) (proto.Script, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.NewestScriptBytesByAccount(recipient)
}

func (a *ThreadSafeReadWrapper) IsActiveLeasing(leaseID crypto.Digest) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.IsActiveLeasing(leaseID)
}

func (a *ThreadSafeReadWrapper) InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.InvokeResultByID(invokeID)
}

func (a *ThreadSafeReadWrapper) ProvidesStateHashes() (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ProvidesStateHashes()
}

func (a *ThreadSafeReadWrapper) StateHashAtHeight(height uint64) (*proto.StateHash, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.StateHashAtHeight(height)
}

func (a *ThreadSafeReadWrapper) ProvidesExtendedApi() (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ProvidesExtendedApi()
}

func (a *ThreadSafeReadWrapper) ShouldPersistAddressTransactions() (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.ShouldPersistAddressTransactions()
}

func NewThreadSafeReadWrapper(mu *sync.RWMutex, s StateInfo) StateInfo {
	return &ThreadSafeReadWrapper{
		mu: mu,
		s:  s,
	}
}

type ThreadSafeWriteWrapper struct {
	mu *sync.RWMutex
	i  *int32
	s  State
}

func (a *ThreadSafeWriteWrapper) Map(f func(state NonThreadSafeState) error) error {
	a.lock()
	defer a.unlock()
	return f(a.s)
}

func (a *ThreadSafeWriteWrapper) ValidateNextTx(_ proto.Transaction, _, _ uint64, _ proto.BlockVersion, _ bool) error {
	panic("Invalid ValidateNextTx usage on thread safe wrapper. Should call TxValidation")
}

func (a *ThreadSafeWriteWrapper) ResetValidationList() {
	panic("invalid ResetValidationList usage")
}

func (a *ThreadSafeWriteWrapper) AddBlock(block []byte) (*proto.Block, error) {
	a.lock()
	defer a.unlock()
	return a.s.AddBlock(block)
}

func (a *ThreadSafeWriteWrapper) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	a.lock()
	defer a.unlock()
	return a.s.AddDeserializedBlock(block)
}

func (a *ThreadSafeWriteWrapper) AddBlocks(blocks [][]byte) error {
	a.lock()
	defer a.unlock()
	return a.s.AddBlocks(blocks)
}

func (a *ThreadSafeWriteWrapper) AddDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	a.lock()
	defer a.unlock()
	return a.s.AddDeserializedBlocks(blocks)
}

func (a *ThreadSafeWriteWrapper) RollbackToHeight(height proto.Height) error {
	a.lock()
	defer a.unlock()
	return a.s.RollbackToHeight(height)
}

func (a *ThreadSafeWriteWrapper) RollbackTo(removalEdge proto.BlockID) error {
	a.lock()
	defer a.unlock()
	return a.s.RollbackTo(removalEdge)
}

func (a *ThreadSafeWriteWrapper) TxValidation(f func(validation TxValidation) error) error {
	a.lock()
	defer a.unlock()
	defer a.s.ResetValidationList()
	return f(a.s)
}

func (a *ThreadSafeWriteWrapper) StartProvidingExtendedApi() error {
	a.lock()
	defer a.unlock()
	return a.s.StartProvidingExtendedApi()
}

func (a *ThreadSafeWriteWrapper) PersistAddressTransactions() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.s.PersistAddressTransactions()
}

func (a *ThreadSafeWriteWrapper) Close() error {
	a.lock()
	defer a.unlock()
	return a.s.Close()
}

func NewThreadSafeWriteWrapper(i *int32, mu *sync.RWMutex, s State) StateModifier {
	return &ThreadSafeWriteWrapper{
		mu: mu,
		i:  i,
		s:  s,
	}
}

func (a *ThreadSafeWriteWrapper) lock() {
	if !atomic.CompareAndSwapInt32(a.i, 0, 1) {
		// previous value was not `0`, so it means we already locked
		// this should never happen, cause all write action happens in only 1 thread.
		// most likely than we change state in another thread and it's forbidden
		panic("already modifying state")
	}
	a.mu.Lock()
}

func (a *ThreadSafeWriteWrapper) unlock() {
	a.mu.Unlock()
	if !atomic.CompareAndSwapInt32(a.i, 1, 0) {
		panic("state was already unlocked")
	}
}

type ThreadSafeState struct {
	StateInfo
	StateModifier
}

func NewThreadSafeState(s State) *ThreadSafeState {
	mu := &sync.RWMutex{}
	var i int32 = 0
	r := NewThreadSafeReadWrapper(mu, s)
	w := NewThreadSafeWriteWrapper(&i, mu, s)
	return &ThreadSafeState{
		StateInfo:     r,
		StateModifier: w,
	}
}
