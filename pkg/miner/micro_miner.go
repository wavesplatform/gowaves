package miner

import (
	"context"
	"errors"
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	maxMicroblockTransactions = 255
)

var ErrBlockIsFull = errors.New("block is full")
var ErrNoTransactions = errors.New("no transactions")
var ErrStateChanged = errors.New("state changed")

type MicroMiner struct {
	state  state.State
	utx    types.UtxPool
	scheme proto.Scheme
	logger *slog.Logger
}

func NewMicroMiner(services services.Services) *MicroMiner {
	return &MicroMiner{
		state:  services.State,
		utx:    services.UtxPool,
		scheme: services.Scheme,
		logger: slog.Default().With(logging.NamespaceKey, "MICRO MINER"),
	}
}

func (a *MicroMiner) Micro(minedBlock *proto.Block, rest proto.MiningLimits, keyPair proto.KeyPair) (*proto.Block, *proto.MicroBlock, proto.MiningLimits, error) {
	// way to stop mine microblocks
	if minedBlock == nil {
		return nil, nil, rest, errors.New("no block provided")
	}

	topBlock := a.state.TopBlock()
	if topBlock.BlockSignature != minedBlock.BlockSignature {
		// block changed, exit
		return nil, nil, rest, ErrStateChanged
	}

	height, err := a.state.Height()
	if err != nil {
		return nil, nil, rest, err
	}
	a.logger.Debug("Generating micro block", "TopBlockID", topBlock.BlockID(), "height", height)

	parentTimestamp := topBlock.Timestamp
	if height > 1 {
		parent, err := a.state.BlockByHeight(height - 1)
		if err != nil {
			return nil, nil, rest, err
		}
		parentTimestamp = parent.Timestamp
	}

	txCount := 0 // counter for successfully applied transactions
	binSize := 0
	droppedTxCount := 0

	var appliedTransactions []*types.TransactionWithBytes
	var inapplicable []*types.TransactionWithBytes
	var txSnapshots [][]proto.AtomicSnapshot
	const minTransactionSize = 40 // Roughly estimated minimal transaction size.

	_ = a.state.MapUnsafe(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()
		const uint32SizeBytes = 4

		for txCount <= maxMicroblockTransactions {
			if rest.MaxTxsSizeInBytes-binSize < minTransactionSize {
				break
			}
			t := a.utx.Pop()
			if t == nil {
				a.logger.Debug("No more transactions in UTX",
					slog.Int("transactions", len(appliedTransactions)),
					slog.Int("inapplicable", len(inapplicable)),
					slog.Int("dropped", droppedTxCount),
				)
				break
			}
			txSizeWithLen := len(t.B) + uint32SizeBytes
			if newTxsSize := binSize + txSizeWithLen; newTxsSize > rest.MaxTxsSizeInBytes {
				inapplicable = append(inapplicable, t)
				continue
			}

			// In the miner we pack transactions from UTX into new block.
			// We should accept failed transactions here.
			// Validate and apply tx to state.
			snapshot, errVal := s.ValidateNextTx(t.T, minedBlock.Timestamp, parentTimestamp, minedBlock.Version, true)
			if stateerr.IsTxCommitmentError(errVal) {
				a.logger.Error("Failed to validate a transaction from UTX, returning applied transactions to UTX",
					logging.Error(errVal), logging.TxID(t.T, a.scheme),
					slog.Int("transactions", len(appliedTransactions)),
					slog.Int("inapplicable", len(inapplicable)),
					slog.Int("dropped", droppedTxCount),
				)
				droppedTxCount++ // drop this tx
				// This should not happen in practice.
				// Reset state, tx count, return applied transactions to UTX.
				s.ResetValidationList()
				txCount = 0
				for _, appliedTx := range appliedTransactions {
					// transactions were validated before, so no need to validate them with state again
					uErr := a.utx.AddWithBytesRaw(appliedTx.T, appliedTx.B)
					if uErr != nil {
						droppedTxCount++ // drop this tx
						a.logger.Warn("Failed to return a successfully applied transaction to UTX, throwing tx away",
							logging.Error(uErr), logging.TxID(t.T, a.scheme),
						)
					}
				}
				a.logger.Debug("Applied transactions returned to UTX, resetting applied list, continuing",
					slog.Int("returned", len(appliedTransactions)),
					slog.Int("inapplicable", len(inapplicable)),
					slog.Int("dropped", droppedTxCount),
				)
				appliedTransactions = nil
				txSnapshots = nil
				continue
			}
			if errVal != nil {
				a.logger.Debug("Transaction from UTX is not applicable to state, skipping",
					logging.Error(errVal), logging.TxID(t.T, a.scheme),
				)
				inapplicable = append(inapplicable, t)
				continue
			}

			txCount += 1
			binSize += txSizeWithLen
			appliedTransactions = append(appliedTransactions, t)
			txSnapshots = append(txSnapshots, snapshot)
		}

		// return inapplicable transactions to utx
		for _, tx := range inapplicable {
			uErr := a.utx.AddWithBytes(s, tx.T, tx.B) // validate with state while adding back
			if uErr != nil {
				droppedTxCount++ // drop this tx
				a.logger.Debug("Failed to return an inapplicable transaction to UTX, throwing tx away",
					logging.Error(uErr), logging.TxID(tx.T, a.scheme),
				)
			}
		}
		return nil
	})

	a.logger.Debug("Transaction validation for micro block finished",
		slog.Int("transactions", len(appliedTransactions)),
		slog.Int("inapplicable", len(inapplicable)),
		slog.Int("dropped", droppedTxCount),
	)

	// no transactions applied, skip
	if txCount == 0 {
		// TODO: we should distinguish between block is full because of size and because or because of complexity
		//  limit reached. For now we return the same error.
		if len(inapplicable) > 0 || rest.MaxTxsSizeInBytes-binSize < minTransactionSize {
			return nil, nil, rest, ErrBlockIsFull
		}
		return nil, nil, rest, ErrNoTransactions
	}

	transactions := make([]proto.Transaction, len(appliedTransactions))
	for i, appliedTx := range appliedTransactions {
		if a.logger.Enabled(context.Background(), slog.LevelDebug) {
			a.logger.Debug("Appending transaction", logging.TxID(appliedTx.T, a.scheme))
		}
		transactions[i] = appliedTx.T
	}
	lightNodeNewBlockActivated, err := a.state.IsActiveLightNodeNewBlocksFields(height)
	if err != nil {
		return nil, nil, rest, err
	}

	var sh *crypto.Digest
	if lightNodeNewBlockActivated {
		prevSh, ok := minedBlock.GetStateHash()
		if !ok {
			return nil, nil, rest, errors.New("mined block should have a state hash field")
		}
		newSh, errSh := state.CalculateSnapshotStateHash(a.scheme, height, prevSh, transactions, txSnapshots)
		if errSh != nil {
			return nil, nil, rest, errSh
		}
		sh = &newSh
	}

	newTransactions := minedBlock.Transactions.Join(transactions)

	newBlock, err := proto.CreateBlock(
		newTransactions,
		minedBlock.Timestamp,
		minedBlock.Parent,
		minedBlock.GeneratorPublicKey,
		minedBlock.NxtConsensus,
		minedBlock.Version,
		minedBlock.Features,
		minedBlock.RewardVote,
		a.scheme,
		sh,
	)
	if err != nil {
		return nil, nil, rest, err
	}
	sk := keyPair.Secret

	err = newBlock.SetTransactionsRootIfPossible(a.scheme)
	if err != nil {
		return nil, nil, rest, err
	}
	err = newBlock.Sign(a.scheme, keyPair.Secret)
	if err != nil {
		return nil, nil, rest, err
	}
	err = newBlock.GenerateBlockID(a.scheme)
	if err != nil {
		return nil, nil, rest, err
	}
	micro := proto.MicroBlock{
		VersionField:          byte(newBlock.Version),
		SenderPK:              keyPair.Public,
		Transactions:          transactions,
		TransactionCount:      uint32(txCount),
		Reference:             a.state.TopBlock().BlockID(),
		TotalResBlockSigField: newBlock.BlockSignature,
		TotalBlockID:          newBlock.BlockID(),
		StateHash:             sh,
	}

	err = micro.Sign(a.scheme, sk)
	if err != nil {
		return nil, nil, rest, err
	}

	a.logger.Debug("Micro block mined", "micro", micro)

	newRest := proto.MiningLimits{
		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - binSize,
	}
	return newBlock, &micro, newRest, nil
}
