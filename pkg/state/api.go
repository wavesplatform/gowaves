package state

import (
	"math/big"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type StateErrorType byte

const (
	// Unmarshal error of block or transaction.
	DeserializationError StateErrorType = iota + 1
	TxValidationError
	BlockValidationError
	RollbackError
	// Errors occurring while getting data from database.
	RetrievalError
	// Errors occurring while updating/modifying state data.
	ModificationError
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
	// Height returns current blockchain height.
	Height() (uint64, error)
	// Height <---> blockID converters.
	BlockIDToHeight(blockID crypto.Signature) (uint64, error)
	HeightToBlockID(height uint64) (crypto.Signature, error)
	// AccountBalance retrieves balance of address in specific currency, asset is asset's ID.
	// nil asset = Waves.
	AccountBalance(addr proto.Address, asset []byte) (uint64, error)
	// AddressesNumber returns total number of addresses in state.
	AddressesNumber() (uint64, error)
	// AddBlock adds single block to state.
	// It's not recommended to use this function when you are able to accumulate big blocks batch,
	// since it's much more efficient to add many blocks at once.
	AddBlock(block []byte) error
	// AddNewBlocks adds batch of new blocks to state.
	// Use it when blocks are logically new.
	AddNewBlocks(blocks [][]byte) error
	// AddOldBlocks adds batch of old blocks to state.
	// Use it when importing historical blockchain.
	AddOldBlocks(blocks [][]byte) error
	// Rollback functionality.
	RollbackToHeight(height uint64) error
	RollbackTo(removalEdge crypto.Signature) error
	// Get cumulative blocks score at given height.
	ScoreAtHeight(height uint64) (*big.Int, error)
	// Get current blockchain score (at top height).
	CurrentScore() (*big.Int, error)
	// Miner's effective balance in given height range.
	EffectiveBalance(addr proto.Address, startHeight, endHeight uint64) (uint64, error)
	// Retrieve current blockchain settings.
	BlockchainSettings() (*settings.BlockchainSettings, error)

	Close() error
}

// NewState() creates State.
// dataDir is path to directory to store all data, it's also possible to provide folder with existing data,
// and state will try to sync and use it in this case.
// params are block storage parameters, they specify lengths of byte offsets for headers and transactions.
// Use state.DefaultBlockStorageParams() to create default parameters.
// Settings are blockchain settings, you can use settings.MainNetSettings, ...
// (TODO: settings.TestNetSettings and custom settings aren't yet supported).
func NewState(dataDir string, params BlockStorageParams, settings *settings.BlockchainSettings) (State, error) {
	return newStateManager(dataDir, params, settings)
}

type BlockStorageParams struct {
	OffsetLen, HeaderOffsetLen int
}

func DefaultBlockStorageParams() BlockStorageParams {
	return BlockStorageParams{OffsetLen: 8, HeaderOffsetLen: 8}
}
