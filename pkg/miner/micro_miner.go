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
func (a *MicroMiner) Micro(
	minedBlock *proto.Block,
	rest proto.MiningLimits,
	keyPair proto.KeyPair,
	endorsements []proto.EndorseBlock,
) (*proto.Block, *proto.MicroBlock, proto.MiningLimits, error) {
	if minedBlock == nil {
		return nil, nil, rest, errors.New("no block provided")
	}

	topBlock := a.state.TopBlock()
	if topBlock.BlockSignature != minedBlock.BlockSignature {
		return nil, nil, rest, ErrStateChanged
	}

	height, err := a.state.Height()
	if err != nil {
		return nil, nil, rest, err
	}
	a.logger.Debug("Generating micro block", "TopBlockID", topBlock.BlockID(), "height", height)

	parentTimestamp, err := a.getParentTimestamp(height)
	if err != nil {
		return nil, nil, rest, err
	}

	appliedTransactions, txSnapshots, binSize,
		txCount, droppedTxCount, inapplicable := a.collectTransactions(minedBlock, rest, parentTimestamp)

	a.logger.Debug("Transaction validation for micro block finished",
		slog.Int("transactions", len(appliedTransactions)),
		slog.Int("inapplicable", len(inapplicable)),
		slog.Int("dropped", droppedTxCount),
	)

	if txCount == 0 {
		if len(inapplicable) > 0 || rest.MaxTxsSizeInBytes-binSize < 40 {
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

	newBlock, sh, err := a.createNewBlock(minedBlock, keyPair, transactions, txSnapshots, height)
	if err != nil {
		return nil, nil, rest, err
	}

	micro, err := a.createMicroBlock(newBlock, keyPair, transactions, endorsements, sh, txCount)
	if err != nil {
		return nil, nil, rest, err
	}

	newRest := proto.MiningLimits{
		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - binSize,
	}
	return newBlock, &micro, newRest, nil
}

// --- helpers ---

func (a *MicroMiner) getParentTimestamp(height uint64) (uint64, error) {
	parentTimestamp := a.state.TopBlock().Timestamp
	if height > 1 {
		parent, err := a.state.BlockByHeight(height - 1)
		if err != nil {
			return 0, err
		}
		parentTimestamp = parent.Timestamp
	}
	return parentTimestamp, nil
}

func (a *MicroMiner) collectTransactions(
	minedBlock *proto.Block,
	rest proto.MiningLimits,
	parentTimestamp uint64,
) ([]*types.TransactionWithBytes, [][]proto.AtomicSnapshot, int, int, int, []*types.TransactionWithBytes) {
	const minTransactionSize = 40
	txCount, binSize, droppedTxCount := 0, 0, 0
	var appliedTransactions, inapplicable []*types.TransactionWithBytes
	var txSnapshots [][]proto.AtomicSnapshot

	_ = a.state.MapUnsafe(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()
		const uint32SizeBytes = 4

		for txCount <= maxMicroblockTransactions {
			if rest.MaxTxsSizeInBytes-binSize < minTransactionSize {
				break
			}
			t := a.utx.Pop()
			if t == nil {
				a.logNoMoreTransactions(appliedTransactions, inapplicable, droppedTxCount)
				break
			}

			if shouldSkip, _ := a.checkTxSize(t, binSize, rest.MaxTxsSizeInBytes, uint32SizeBytes); shouldSkip {
				inapplicable = append(inapplicable, t)
				continue
			}

			snapshot, errVal := s.ValidateNextTx(t.T, minedBlock.Timestamp, parentTimestamp, minedBlock.Version, true)
			if stateerr.IsTxCommitmentError(errVal) {
				droppedTxCount += a.resetOnCommitmentError(s, t, appliedTransactions, droppedTxCount)
				appliedTransactions = nil
				txSnapshots = nil
				txCount = 0
				continue
			}
			if errVal != nil {
				a.logInapplicableTx(t, errVal)
				inapplicable = append(inapplicable, t)
				continue
			}

			txCount++
			binSize += len(t.B) + uint32SizeBytes
			appliedTransactions = append(appliedTransactions, t)
			txSnapshots = append(txSnapshots, snapshot)
		}

		a.returnInapplicableTxs(s, inapplicable, &droppedTxCount)
		return nil
	})

	return appliedTransactions, txSnapshots, binSize, txCount, droppedTxCount, inapplicable
}

func (a *MicroMiner) logNoMoreTransactions(applied, inapplicable []*types.TransactionWithBytes, dropped int) {
	a.logger.Debug("No more transactions in UTX",
		slog.Int("transactions", len(applied)),
		slog.Int("inapplicable", len(inapplicable)),
		slog.Int("dropped", dropped),
	)
}

func (a *MicroMiner) logInapplicableTx(t *types.TransactionWithBytes, err error) {
	a.logger.Debug("Transaction from UTX is not applicable",
		logging.Error(err),
		logging.TxID(t.T, a.scheme),
	)
}

func (a *MicroMiner) checkTxSize(
	t *types.TransactionWithBytes,
	binSize, maxSize int,
	uint32SizeBytes int,
) (bool, int) {
	txSizeWithLen := len(t.B) + uint32SizeBytes
	if binSize+txSizeWithLen > maxSize {
		return true, txSizeWithLen
	}
	return false, txSizeWithLen
}

func (a *MicroMiner) returnInapplicableTxs(
	s state.NonThreadSafeState,
	inapplicable []*types.TransactionWithBytes,
	dropped *int,
) {
	for _, tx := range inapplicable {
		if uErr := a.utx.AddWithBytes(s, tx.T, tx.B); uErr != nil {
			(*dropped)++
			a.logger.Debug("Failed to return inapplicable tx",
				logging.Error(uErr),
				logging.TxID(tx.T, a.scheme),
			)
		}
	}
}

func (a *MicroMiner) resetOnCommitmentError(
	s state.NonThreadSafeState,
	t *types.TransactionWithBytes,
	applied []*types.TransactionWithBytes,
	dropped int,
) int {
	a.logger.Error("Tx commitment error, returning applied txs", logging.TxID(t.T, a.scheme))
	s.ResetValidationList()
	for _, appliedTx := range applied {
		if uErr := a.utx.AddWithBytesRaw(appliedTx.T, appliedTx.B); uErr != nil {
			dropped++
			a.logger.Warn("Failed to return applied tx", logging.Error(uErr), logging.TxID(appliedTx.T, a.scheme))
		}
	}
	return dropped
}

func (a *MicroMiner) createNewBlock(
	minedBlock *proto.Block,
	keyPair proto.KeyPair,
	transactions []proto.Transaction,
	txSnapshots [][]proto.AtomicSnapshot,
	height uint64,
) (*proto.Block, *crypto.Digest, error) {
	lightNodeNewBlockActivated, err := a.state.IsActiveLightNodeNewBlocksFields(height)
	if err != nil {
		return nil, nil, err
	}

	var sh *crypto.Digest
	if lightNodeNewBlockActivated {
		prevSh, ok := minedBlock.GetStateHash()
		if !ok {
			return nil, nil, errors.New("mined block should have a state hash field")
		}
		newSh, errSh := state.CalculateSnapshotStateHash(a.scheme, height, prevSh, transactions, txSnapshots)
		if errSh != nil {
			return nil, nil, errSh
		}
		sh = &newSh
	}

	newTransactions := minedBlock.Transactions.Join(transactions)
	newBlock, err := proto.CreateBlock(
		newTransactions, minedBlock.Timestamp, minedBlock.Parent,
		minedBlock.GeneratorPublicKey, minedBlock.NxtConsensus,
		minedBlock.Version, minedBlock.Features, minedBlock.RewardVote,
		a.scheme, sh,
	)
	if err != nil {
		return nil, nil, err
	}

	if err = newBlock.SetTransactionsRootIfPossible(a.scheme); err != nil {
		return nil, nil, err
	}
	if err = newBlock.Sign(a.scheme, keyPair.Secret); err != nil {
		return nil, nil, err
	}
	if err = newBlock.GenerateBlockID(a.scheme); err != nil {
		return nil, nil, err
	}

	return newBlock, sh, nil
}

func (a *MicroMiner) createMicroBlock(
	newBlock *proto.Block,
	keyPair proto.KeyPair,
	transactions []proto.Transaction,
	endorsements []proto.EndorseBlock,
	sh *crypto.Digest,
	txCount int,
) (proto.MicroBlock, error) {
	micro := proto.MicroBlock{
		VersionField:          byte(newBlock.Version),
		SenderPK:              keyPair.Public,
		Transactions:          transactions,
		TransactionCount:      uint32(txCount),
		Reference:             a.state.TopBlock().BlockID(),
		TotalResBlockSigField: newBlock.BlockSignature,
		TotalBlockID:          newBlock.BlockID(),
		StateHash:             sh,
		Endorsements:          endorsements,
	}
	if err := micro.Sign(a.scheme, keyPair.Secret); err != nil {
		return proto.MicroBlock{}, err
	}
	a.logger.Debug("Micro block mined", "micro", micro)
	return micro, nil
}
