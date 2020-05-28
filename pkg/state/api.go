package state

import (
	"math/big"
	"runtime"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// TransactionIterator can be used to iterate through transactions of given address.
// One instance is only valid for iterating once.
// Transaction() returns current transaction.
// Next() moves iterator to next position, it must be called first time at the beginning.
// Release() must be called after using iterator.
// Error() should return nil if iterating was successful.
type TransactionIterator interface {
	Transaction() (proto.Transaction, bool, error)
	Next() bool
	Release()
	Error() error
}

// StateInfo returns information that corresponds to latest fully applied block.
// This should be used for APIs and other modules where stable, fully verified state is needed.
// Methods of this interface are thread-safe.
type StateInfo interface {
	// Block getters.
	TopBlock() *proto.Block
	Block(blockID proto.BlockID) (*proto.Block, error)
	BlockByHeight(height proto.Height) (*proto.Block, error)
	// Header getters.
	Header(blockID proto.BlockID) (*proto.BlockHeader, error)
	HeaderByHeight(height proto.Height) (*proto.BlockHeader, error)
	// Height returns current blockchain height.
	Height() (proto.Height, error)
	// Height <---> blockID converters.
	BlockIDToHeight(blockID proto.BlockID) (proto.Height, error)
	HeightToBlockID(height proto.Height) (proto.BlockID, error)
	// FullWavesBalance returns complete Waves balance record.
	FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error)
	EffectiveBalanceStable(account proto.Recipient, startHeight, endHeight proto.Height) (uint64, error)
	// AccountBalance retrieves balance of account in specific currency, asset is asset's ID.
	// nil asset = Waves.
	AccountBalance(account proto.Recipient, asset []byte) (uint64, error)
	// WavesAddressesNumber returns total number of Waves addresses in state.
	// It is extremely slow, so it is recommended to only use for testing purposes.
	WavesAddressesNumber() (uint64, error)

	// Get cumulative blocks score at given height.
	ScoreAtHeight(height proto.Height) (*big.Int, error)
	// Get current blockchain score (at top height).
	CurrentScore() (*big.Int, error)

	// Retrieve current blockchain settings.
	BlockchainSettings() (*settings.BlockchainSettings, error)

	Peers() ([]proto.TCPAddr, error)

	// Features.
	VotesNum(featureID int16) (uint64, error)
	VotesNumAtHeight(featureID int16, height proto.Height) (uint64, error)
	IsActivated(featureID int16) (bool, error)
	IsActiveAtHeight(featureID int16, height proto.Height) (bool, error)
	ActivationHeight(featureID int16) (proto.Height, error)
	IsApproved(featureID int16) (bool, error)
	IsApprovedAtHeight(featureID int16, height proto.Height) (bool, error)
	ApprovalHeight(featureID int16) (proto.Height, error)
	AllFeatures() ([]int16, error)

	// Aliases.
	AddrByAlias(alias proto.Alias) (proto.Address, error)

	// Accounts data storage.
	RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error)
	RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error)
	RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)

	// Transactions.
	TransactionByID(id []byte) (proto.Transaction, error)
	TransactionByIDWithStatus(id []byte) (proto.Transaction, bool, error)
	TransactionHeightByID(id []byte) (uint64, error)
	// NewAddrTransactionsIterator() returns iterator to iterate all transactions that affected
	// given address.
	// Iterator will move in range from most recent to oldest transactions.
	NewAddrTransactionsIterator(addr proto.Address) (TransactionIterator, error)

	// Asset fee sponsorship.
	AssetIsSponsored(assetID crypto.Digest) (bool, error)
	AssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
	FullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error)

	// Script information.
	ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error)
	ScriptInfoByAsset(assetID crypto.Digest) (*proto.ScriptInfo, error)

	// Leases.
	IsActiveLeasing(leaseID crypto.Digest) (bool, error)

	// Invoke results.
	InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error)

	// True if state stores additional information in order to provide extended API.
	ProvidesExtendedApi() (bool, error)

	// True if state stores and calculates state hashes for each block height.
	ProvidesStateHashes() (bool, error)

	// State hashes.
	StateHashAtHeight(height uint64) (*proto.StateHash, error)

	// Map on readable state. Way to apply multiple operations under same lock.
	MapR(func(StateInfo) (interface{}, error)) (interface{}, error)

	// HitSourceAtHeight reads hit source stored in state.
	HitSourceAtHeight(height proto.Height) ([]byte, error)

	// BlockVRF calculates VRF for given block.
	BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error)

	// ShouldPersisAddressTransactions checks if PersisAddressTransactions
	// should be called.
	ShouldPersistAddressTransactions() (bool, error)
}

// StateModifier contains all the methods needed to modify node's state.
// Methods of this interface are not thread-safe.
type StateModifier interface {
	// AddBlock adds single block to state.
	// It's not recommended to use this function when you are able to accumulate big blocks batch,
	// since it's much more efficient to add many blocks at once.
	AddBlock(block []byte) (*proto.Block, error)
	AddDeserializedBlock(block *proto.Block) (*proto.Block, error)
	// AddNewBlocks adds batch of new blocks to state.
	// Use it when blocks are logically new.
	AddNewBlocks(blocks [][]byte) error
	// AddNewDeserializedBlocks marshals blocks to binary and calls AddNewBlocks().
	AddNewDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error)
	// AddOldBlocks adds batch of old blocks to state.
	// Use it when importing historical blockchain.
	// It is faster than AddNewBlocks but it is only safe when importing from scratch when no rollbacks are possible at all.
	AddOldBlocks(blocks [][]byte) error
	// AddOldDeserializedBlocks marshals blocks to binary and calls AddOldBlocks().
	AddOldDeserializedBlocks(blocks []*proto.Block) error
	// Rollback functionality.
	RollbackToHeight(height proto.Height) error
	RollbackTo(removalEdge proto.BlockID) error

	// -------------------------
	// Validation functionality (for UTX).
	// -------------------------
	// ValidateNextTx() validates transaction against state, taking into account all the previous changes from transactions
	// that were added using ValidateNextTx() until you call ResetValidationList().
	// checkScripts specifies if scripts for Exchange and Invoke transactions
	// should be checked.
	// Returns TxValidationError or nil.
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, blockVersion proto.BlockVersion, checkScripts bool) error
	// ResetValidationList() resets the validation list, so you can ValidateNextTx() from scratch after calling it.
	ResetValidationList()

	// Func internally calls ResetValidationList.
	TxValidation(func(validation TxValidation) error) error

	// Way to call multiple operations under same lock.
	Map(func(state NonThreadSafeState) error) error

	// Create or replace Peers.
	SavePeers([]proto.TCPAddr) error

	// State will provide extended API data after returning.
	StartProvidingExtendedApi() error

	// PersisAddressTransactions sorts and saves transactions to storage.
	PersistAddressTransactions() error

	Close() error
}

type NonThreadSafeState = State

type TxValidation interface {
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, blockVersion proto.BlockVersion, checkScripts bool) error
}

type State interface {
	StateInfo
	StateModifier
}

// NewState() creates State.
// dataDir is path to directory to store all data, it's also possible to provide folder with existing data,
// and state will try to sync and use it in this case.
// params are state parameters (see below).
// settings are blockchain settings (settings.MainNetSettings, settings.TestNetSettings or custom settings).
func NewState(dataDir string, params StateParams, settings *settings.BlockchainSettings) (State, error) {
	s, err := newStateManager(dataDir, params, settings)
	if err != nil {
		return nil, err
	}
	return NewThreadSafeState(s), nil
}

type StorageParams struct {
	OffsetLen       int
	HeaderOffsetLen int
	DbParams        keyvalue.KeyValParams
}

func DefaultStorageParams() StorageParams {
	dbParams := keyvalue.KeyValParams{
		CacheParams: keyvalue.CacheParams{Size: DefaultCacheSize},
		BloomFilterParams: keyvalue.BloomFilterParams{
			N:                        DefaultBloomFilterSize,
			FalsePositiveProbability: DefaultBloomFilterFalsePositiveProbability,
			Store:                    keyvalue.NewStore(""),
		},
		WriteBuffer:         DefaultWriteBuffer,
		CompactionTableSize: DefaultCompactionTableSize,
		CompactionTotalSize: DefaultCompactionTotalSize,
	}
	return StorageParams{
		OffsetLen:       DefaultOffsetLen,
		HeaderOffsetLen: DefaultHeaderOffsetLen,
		DbParams:        dbParams,
	}
}

func DefaultTestingStorageParams() StorageParams {
	d := DefaultStorageParams()
	d.DbParams.N = 10
	return d
}

// ValidationParams are validation parameters.
// VerificationGoroutinesNum specifies how many goroutines will be run for verification of transactions and blocks signatures.
type ValidationParams struct {
	VerificationGoroutinesNum int
	Time                      types.Time
}

type StateParams struct {
	StorageParams
	ValidationParams
	// When StoreExtendedApiData is true, state builds additional data required for API.
	StoreExtendedApiData bool
	// ProvideExtendedApi specifies whether state must provide data for extended API.
	ProvideExtendedApi bool
	// BuildStateHashes enables building and storing state hashes by height.
	BuildStateHashes bool
}

func DefaultStateParams() StateParams {
	return StateParams{
		StorageParams: DefaultStorageParams(),
		ValidationParams: ValidationParams{
			VerificationGoroutinesNum: runtime.NumCPU() * 2,
			Time:                      ntptime.Stub{},
		},
	}
}

func DefaultTestingStateParams() StateParams {
	return StateParams{
		StorageParams: DefaultTestingStorageParams(),
		ValidationParams: ValidationParams{
			VerificationGoroutinesNum: runtime.NumCPU() * 2,
			Time:                      ntptime.Stub{},
		},
	}
}
