package state

import (
	"math/big"
	"runtime"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type StateErrorType byte

const (
	// Unmarshal error of block or transaction.
	DeserializationError StateErrorType = iota + 1
	// Marshal error of block or transaction.
	SerializationError
	TxValidationError
	BlockValidationError
	// Either block or tx.
	ValidationError
	RollbackError
	// Errors occurring while getting data from database.
	RetrievalError
	// Errors occurring while updating/modifying state data.
	ModificationError
	InvalidInputError
	// DB or block storage Close() error.
	ClosureError
	// Minor technical errors which shouldn't ever happen.
	Other
)

type StateError struct {
	errorType     StateErrorType
	originalError error
}

func (err StateError) Error() string {
	return err.originalError.Error()
}

func ErrorType(err error) StateErrorType {
	switch e := err.(type) {
	case StateError:
		return e.errorType
	default:
		return 0
	}
}

// State represents overall Node's state.
// Data retrievals (e.g. account balances), as well as modifiers (like adding or rolling back blocks)
// should all be made using this interface.
type State interface {
	// Block getters.
	Block(blockID crypto.Signature) (*proto.Block, error)
	BlockByHeight(height uint64) (*proto.Block, error)
	BlockBytes(blockID crypto.Signature) ([]byte, error)
	BlockBytesByHeight(height uint64) ([]byte, error)
	// Header getters.
	Header(blockID crypto.Signature) (*proto.BlockHeader, error)
	HeaderByHeight(height uint64) (*proto.BlockHeader, error)
	HeaderBytes(blockID crypto.Signature) ([]byte, error)
	HeaderBytesByHeight(height uint64) ([]byte, error)
	// Height returns current blockchain height.
	Height() (uint64, error)
	// Height <---> blockID converters.
	BlockIDToHeight(blockID crypto.Signature) (uint64, error)
	HeightToBlockID(height uint64) (crypto.Signature, error)
	// AccountBalance retrieves balance of address in specific currency, asset is asset's ID.
	// nil asset = Waves.
	AccountBalance(addr proto.Address, asset []byte) (uint64, error)
	// WavesAddressesNumber returns total number of Waves addresses in state.
	// It is extremely slow, so it is recommended to only use for testing purposes.
	WavesAddressesNumber() (uint64, error)
	// AddBlock adds single block to state.
	// It's not recommended to use this function when you are able to accumulate big blocks batch,
	// since it's much more efficient to add many blocks at once.
	AddBlock(block []byte) error
	AddDeserializedBlock(block *proto.Block) error
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
	RollbackToHeight(height uint64) error
	RollbackTo(removalEdge crypto.Signature) error
	// Get cumulative blocks score at given height.
	ScoreAtHeight(height uint64) (*big.Int, error)
	// Get current blockchain score (at top height).
	CurrentScore() (*big.Int, error)
	// Retrieve current blockchain settings.
	BlockchainSettings() (*settings.BlockchainSettings, error)
	// Effective balance by address in given height range.
	// WARNING: this function takes into account newest blocks (which are currently being added)
	// and works correctly for height ranges exceeding current Height() if there are such blocks.
	// It does not work for heights older than rollbackMax blocks before the current block.
	EffectiveBalance(addr proto.Address, startHeight, endHeight uint64) (uint64, error)

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
	Peers() ([]proto.TCPAddr, error)

	// Features.
	IsActivated(featureID int16) (bool, error)
	ActivationHeight(featureID int16) (uint64, error)
	IsApproved(featureID int16) (bool, error)
	ApprovalHeight(featureID int16) (uint64, error)

	Close() error
}

// NewState() creates State.
// dataDir is path to directory to store all data, it's also possible to provide folder with existing data,
// and state will try to sync and use it in this case.
// params are state parameters (see below).
// settings are blockchain settings (settings.MainNetSettings, settings.TestNetSettings or custom settings).
func NewState(dataDir string, params StateParams, settings *settings.BlockchainSettings) (State, error) {
	return newStateManager(dataDir, params, settings)
}

// StorageParams are storage parameters, they specify lengths of byte offsets for headers and transactions
// and Bloom Filter's parameters.
// Use state.DefaultStorageParams() to create default parameters.
type StorageParams struct {
	OffsetLen, HeaderOffsetLen int
	BloomParams                keyvalue.BloomFilterParams
}

func DefaultStorageParams() StorageParams {
	return StorageParams{OffsetLen: 8, HeaderOffsetLen: 8, BloomParams: keyvalue.BloomFilterParams{N: 2e8, FalsePositiveProbability: 0.01}}
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
