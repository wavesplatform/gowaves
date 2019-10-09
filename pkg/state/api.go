package state

import (
	"math/big"
	"runtime"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/lock"
)

// StateNewest returns information that takes into account any intermediate changes
// occurring during applying block. This state corresponds to the latest validated transaction,
// and for now is only needed for Ride and Consensus modules, which are both called during the validation.
type StateNewest interface {
	AddingBlockHeight() (proto.Height, error)
	NewestHeight() (proto.Height, error)

	// Effective balance by account in given height range.
	// WARNING: this function takes into account newest blocks (which are currently being added)
	// and works correctly for height ranges exceeding current Height() if there are such blocks.
	// It does not work for heights older than rollbackMax blocks before the current block.
	EffectiveBalance(account proto.Recipient, startHeight, endHeight proto.Height) (uint64, error)

	NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error)

	// Aliases.
	NewestAddrByAlias(alias proto.Alias) (proto.Address, error)

	// Accounts data storage.
	RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error)
	RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)

	// Transactions.
	NewestTransactionByID(id []byte) (proto.Transaction, error)
	NewestTransactionHeightByID(id []byte) (uint64, error)

	// Asset fee sponsorship.
	NewestAssetIsSponsored(assetID crypto.Digest) (bool, error)
	NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)

	NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error)
}

// StateStable returns information that corresponds to latest fully applied block.
// This should be used for APIs and other modules where stable, fully verified state is needed.
type StateStable interface {
	// Block getters.
	Block(blockID crypto.Signature) (*proto.Block, error)
	BlockByHeight(height proto.Height) (*proto.Block, error)
	BlockBytes(blockID crypto.Signature) ([]byte, error)
	BlockBytesByHeight(height proto.Height) ([]byte, error)
	// Header getters.
	Header(blockID crypto.Signature) (*proto.BlockHeader, error)
	HeaderByHeight(height proto.Height) (*proto.BlockHeader, error)
	HeaderBytes(blockID crypto.Signature) ([]byte, error)
	HeaderBytesByHeight(height proto.Height) ([]byte, error)
	// Height returns current blockchain height.
	Height() (proto.Height, error)
	// Height <---> blockID converters.
	BlockIDToHeight(blockID crypto.Signature) (proto.Height, error)
	HeightToBlockID(height proto.Height) (crypto.Signature, error)
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
	IsActivated(featureID int16) (bool, error)
	ActivationHeight(featureID int16) (proto.Height, error)
	IsApproved(featureID int16) (bool, error)
	ApprovalHeight(featureID int16) (proto.Height, error)

	// Aliases.
	AddrByAlias(alias proto.Alias) (proto.Address, error)

	// Accounts data storage.
	RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error)
	RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)

	// Transactions.
	TransactionByID(id []byte) (proto.Transaction, error)
	TransactionHeightByID(id []byte) (uint64, error)

	// Asset fee sponsorship.
	AssetIsSponsored(assetID crypto.Digest) (bool, error)
	AssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
}

// StateModifier contains all the methods needed to modify node's state.
type StateModifier interface {
	// Global mutex of state.
	Mutex() *lock.RwMutex
	// AddBlock adds single block to state.
	// It's not recommended to use this function when you are able to accumulate big blocks batch,
	// since it's much more efficient to add many blocks at once.
	AddBlock(block []byte) (*proto.Block, error)
	AddDeserializedBlock(block *proto.Block) (*proto.Block, error)
	// AddNewBlocks adds batch of new blocks to state.
	// Use it when blocks are logically new.
	AddNewBlocks(blocks [][]byte) error
	// AddNewDeserializedBlocks marshals blocks to binary and calls AddNewBlocks().
	AddNewDeserializedBlocks(blocks []*proto.Block) error
	// AddOldBlocks adds batch of old blocks to state.
	// Use it when importing historical blockchain.
	// It is faster than AddNewBlocks but it is only safe when importing from scratch when no rollbacks are possible at all.
	AddOldBlocks(blocks [][]byte) error
	// AddOldDeserializedBlocks marshals blocks to binary and calls AddOldBlocks().
	AddOldDeserializedBlocks(blocks []*proto.Block) error
	// Rollback functionality.
	RollbackToHeight(height proto.Height) error
	RollbackTo(removalEdge crypto.Signature) error

	// -------------------------
	// Validation functionality.
	// -------------------------
	// ValidateSingleTx() validates single transaction against current state.
	// It does not change state. When validating, it does not take into account previous transactions that were validated.
	// Returns TxValidationError or nil.
	ValidateSingleTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error
	// ValidateNextTx() validates transaction against state, taking into account all the previous changes from transactions
	// that were added using ValidateNextTx() until you call ResetValidationList().
	// Does not change state.
	// Returns TxValidationError or nil.
	ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64) error
	// ResetValidationList() resets the validation list, so you can ValidateNextTx() from scratch after calling it.
	ResetValidationList()

	// Create or replace Peers.
	SavePeers([]proto.TCPAddr) error

	Close() error
}

// State represents overall Node's state.
type State interface {
	StateModifier
	StateStable
	StateNewest

	IsNotFound(err error) bool
}

// NewState() creates State.
// dataDir is path to directory to store all data, it's also possible to provide folder with existing data,
// and state will try to sync and use it in this case.
// params are state parameters (see below).
// settings are blockchain settings (settings.MainNetSettings, settings.TestNetSettings or custom settings).
func NewState(dataDir string, params StateParams, settings *settings.BlockchainSettings) (State, error) {
	return newStateManager(dataDir, params, settings)
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

// ValidationParams are validation parameters.
// VerificationGoroutinesNum specifies how many goroutines will be run for verification of transactions and blocks signatures.
type ValidationParams struct {
	VerificationGoroutinesNum int
}

type StateParams struct {
	StorageParams
	ValidationParams
}

func DefaultStateParams() StateParams {
	return StateParams{DefaultStorageParams(), ValidationParams{runtime.NumCPU() * 2}}
}
