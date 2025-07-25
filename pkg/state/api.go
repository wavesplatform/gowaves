package state

import (
	"math/big"
	"runtime"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
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
	Transaction() (proto.Transaction, proto.TransactionStatus, error)
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
	WavesBalance(account proto.Recipient) (uint64, error)
	// FullWavesBalance returns complete Waves balance record.
	FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error)
	GeneratingBalance(account proto.Recipient, height proto.Height) (uint64, error)
	// AssetBalance retrieves balance of account in specific currency, asset is asset's ID.
	AssetBalance(account proto.Recipient, assetID proto.AssetID) (uint64, error)
	// WavesAddressesNumber returns total number of Waves addresses in state.
	// It is extremely slow, so it is recommended to only use for testing purposes.
	WavesAddressesNumber() (uint64, error)

	// Get cumulative blocks score at given height.
	ScoreAtHeight(height proto.Height) (*big.Int, error)
	// Get current blockchain score (at top height).
	CurrentScore() (*big.Int, error)

	// Retrieve current blockchain settings.
	BlockchainSettings() (*settings.BlockchainSettings, error)

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
	EstimatorVersion() (int, error)
	IsActiveLightNodeNewBlocksFields(blockHeight proto.Height) (bool, error)

	// Aliases.
	AddrByAlias(alias proto.Alias) (proto.WavesAddress, error)
	AliasesByAddr(addr proto.WavesAddress) ([]string, error)

	// Accounts data storage.
	RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error)
	RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error)
	RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)

	// Transactions.
	TransactionByID(id []byte) (proto.Transaction, error)
	TransactionByIDWithStatus(id []byte) (proto.Transaction, proto.TransactionStatus, error)
	TransactionHeightByID(id []byte) (uint64, error)
	// NewAddrTransactionsIterator() returns iterator to iterate all transactions that affected
	// given address.
	// Iterator will move in range from most recent to oldest transactions.
	NewAddrTransactionsIterator(addr proto.Address) (TransactionIterator, error)

	// Asset fee sponsorship.
	AssetIsSponsored(assetID proto.AssetID) (bool, error)
	IsAssetExist(assetID proto.AssetID) (bool, error)
	AssetInfo(assetID proto.AssetID) (*proto.AssetInfo, error)
	FullAssetInfo(assetID proto.AssetID) (*proto.FullAssetInfo, error)
	EnrichedFullAssetInfo(assetID proto.AssetID) (*proto.EnrichedFullAssetInfo, error)
	NFTList(account proto.Recipient, limit uint64, afterAssetID *proto.AssetID) ([]*proto.FullAssetInfo, error)
	// Script information.
	ScriptBasicInfoByAccount(account proto.Recipient) (*proto.ScriptBasicInfo, error)
	ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error)
	ScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error)
	NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error)
	NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error)

	// Leases.
	IsActiveLeasing(leaseID crypto.Digest) (bool, error)

	// Invoke results.
	InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error)
	// True if state stores additional information in order to provide extended API.
	ProvidesExtendedApi() (bool, error)
	// True if state stores and calculates state hashes for each block height.
	ProvidesStateHashes() (bool, error)

	// State hashes.
	LegacyStateHashAtHeight(height proto.Height) (*proto.StateHash, error)
	SnapshotStateHashAtHeight(height proto.Height) (crypto.Digest, error)
	// CreateNextSnapshotHash creates snapshot hash for next block in the context of current state.
	CreateNextSnapshotHash(block *proto.Block) (crypto.Digest, error)

	// Map on readable state. Way to apply multiple operations under same lock.
	MapR(func(StateInfo) (any, error)) (any, error)

	// HitSourceAtHeight reads hit source stored in state.
	HitSourceAtHeight(height proto.Height) ([]byte, error)
	// BlockVRF calculates VRF value for the block at given height.
	BlockVRF(blockHeader *proto.BlockHeader, blockHeight proto.Height) ([]byte, error)

	// ShouldPersistAddressTransactions checks the size of temporary transaction storage file and returns true if we
	// should move transactions into the main storage.
	ShouldPersistAddressTransactions() (bool, error)

	// RewardAtHeight returns reward for the block at the given height.
	// Return zero without error if the feature #14 "BlockReward" is not activated.
	// It takes into account the reward multiplier introduced with the feature #23 "BoostBlockReward".
	RewardAtHeight(height proto.Height) (uint64, error)

	RewardVotes(height proto.Height) (proto.RewardVotes, error)

	// TotalWavesAmount returns total amount of Waves in the system at the given height.
	// It returns the initial Waves amount of 100 000 000 before activation of feature #14 "BlockReward".
	// It takes into account the reward multiplier introduced with the feature #23 "BoostBlockReward".
	TotalWavesAmount(height proto.Height) (uint64, error)
	// BlockRewards calculates block rewards for the block at given height with given generator address.
	BlockRewards(generator proto.WavesAddress, height proto.Height) (proto.Rewards, error)

	// SnapshotsAtHeight returns block snapshots at the given height.
	SnapshotsAtHeight(height proto.Height) (proto.BlockSnapshot, error)
}

// StateModifier contains all the methods needed to modify node's state.
// Methods of this interface are not thread-safe.
type StateModifier interface {
	// AddBlock adds single block to state.
	// It's not recommended using this function when you are able to accumulate big blocks batch,
	// since it's much more efficient to add many blocks at once.
	AddBlock(block []byte) (*proto.Block, error)
	AddDeserializedBlock(block *proto.Block) (*proto.Block, error)
	// AddBlocks adds batch of new blocks to state.
	AddBlocks(blocks [][]byte) error
	AddBlocksWithSnapshots(blocks [][]byte, snapshots []*proto.BlockSnapshot) error
	// AddDeserializedBlocks marshals blocks to binary and calls AddBlocks.
	AddDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error)
	AddDeserializedBlocksWithSnapshots(blocks []*proto.Block, snapshots []*proto.BlockSnapshot) (*proto.Block, error)
	// Rollback functionality.
	RollbackToHeight(height proto.Height) error
	RollbackTo(removalEdge proto.BlockID) error

	// -------------------------
	// Validation functionality (for UTX).
	// -------------------------
	// ValidateNextTx() validates transaction against state, taking into account all the previous changes from transactions
	// that were added using ValidateNextTx() until you call ResetValidationList().
	// Returns TxCommitmentError or other state error or nil.
	// When TxCommitmentError is returned, state MUST BE cleared using ResetValidationList().
	ValidateNextTx(
		tx proto.Transaction,
		currentTimestamp, parentTimestamp uint64,
		blockVersion proto.BlockVersion,
		acceptFailed bool,
	) ([]proto.AtomicSnapshot, error)
	// ResetValidationList() resets the validation list, so you can ValidateNextTx() from scratch after calling it.
	ResetValidationList()

	// Func internally calls ResetValidationList.
	TxValidation(func(validation TxValidation) error) error

	// Way to call multiple operations under same lock.
	Map(func(state NonThreadSafeState) error) error

	// State will provide extended API data after returning.
	StartProvidingExtendedApi() error

	// PersistAddressTransactions sorts and saves transactions to storage.
	PersistAddressTransactions() error

	Close() error
}

type NonThreadSafeState = State

type TxValidation interface {
	ValidateNextTx(
		tx proto.Transaction,
		currentTimestamp, parentTimestamp uint64,
		blockVersion proto.BlockVersion,
		acceptFailed bool,
	) ([]proto.AtomicSnapshot, error)
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
func NewState(
	dataDir string,
	amend bool,
	params StateParams,
	settings *settings.BlockchainSettings,
	enableLightNode bool,
) (State, error) {
	s, err := newStateManager(dataDir, amend, params, settings, enableLightNode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new state instance")
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
		CacheParams: keyvalue.CacheParams{CacheSize: DefaultCacheSize},
		BloomFilterParams: keyvalue.BloomFilterParams{
			BloomFilterCapacity:      DefaultBloomFilterSize,
			FalsePositiveProbability: DefaultBloomFilterFalsePositiveProbability,
			BloomFilterStore:         keyvalue.NewStore(""),
		},
		WriteBuffer:            DefaultWriteBuffer,
		CompactionTableSize:    DefaultCompactionTableSize,
		CompactionTotalSize:    DefaultCompactionTotalSize,
		OpenFilesCacheCapacity: DefaultOpenFilesCacheCapacity,
	}
	return StorageParams{
		OffsetLen:       DefaultOffsetLen,
		HeaderOffsetLen: DefaultHeaderOffsetLen,
		DbParams:        dbParams,
	}
}

func DefaultTestingStorageParams() StorageParams {
	d := DefaultStorageParams()
	d.DbParams.BloomFilterCapacity = 10
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
