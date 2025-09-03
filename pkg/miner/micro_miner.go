package miner

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mr-tron/base58"

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

	txCount := 0
	binSize := 0
	droppedTxCount := 0

	var appliedTransactions []*types.TransactionWithBytes
	var inapplicable []*types.TransactionWithBytes
	var txSnapshots [][]proto.AtomicSnapshot

	_ = a.state.MapUnsafe(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()
		const uint32SizeBytes = 4

		for txCount <= maxMicroblockTransactions {
			t := a.utx.Pop()
			if t == nil {
				a.logger.Debug("No more transactions in UTX",
					slog.Int("txCount", txCount),
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
					logging.Error(errVal), txIDSlogAttr(t.T, a.scheme),
					slog.Int("txCount", txCount),
					slog.Int("transactions", len(appliedTransactions)),
					slog.Int("inapplicable", len(inapplicable)),
					slog.Int("dropped", droppedTxCount),
				)
				droppedTxCount += 1 // drop this tx
				// This should not happen in practice.
				// Reset state, tx count, return applied transactions to UTX.
				s.ResetValidationList()
				txCount = 0
				for _, appliedTx := range appliedTransactions {
					// transactions were validated before, so no need to validate them with state again
					uErr := a.utx.AddWithBytesRaw(appliedTx.T, appliedTx.B)
					if uErr != nil {
						droppedTxCount += 1
						a.logger.Warn("Failed to return an successfully applied transaction to UTX, throwing tx away",
							logging.Error(uErr), txIDSlogAttr(t.T, a.scheme),
						)
					}
				}
				a.logger.Debug("Applied transactions returned to UTX, resetting applied list, continuing",
					slog.Int("txCount", txCount),
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
					logging.Error(errVal), txIDSlogAttr(t.T, a.scheme),
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
				droppedTxCount += 1
				a.logger.Debug("Failed to return an inapplicable transaction to UTX, throwing tx away",
					logging.Error(uErr), txIDSlogAttr(tx.T, a.scheme),
				)
			}
		}
		return nil
	})

	a.logger.Debug("Transaction validation for micro block finished",
		slog.Int("txCount", txCount),
		slog.Int("transactions", len(appliedTransactions)),
		slog.Int("inapplicable", len(inapplicable)),
		slog.Int("dropped", droppedTxCount),
	)

	// no transactions applied, skip
	if txCount == 0 {
		return nil, nil, rest, ErrNoTransactions
	}

	transactions := make([]proto.Transaction, len(appliedTransactions))
	for i, appliedTx := range appliedTransactions {
		if a.logger.Enabled(context.Background(), slog.LevelDebug) {
			if id, idErr := appliedTx.T.GetID(a.scheme); idErr != nil {
				slog.Error("Failed to get transaction ID", logging.Error(idErr))
			} else {
				a.logger.Debug("Appending transaction", "TxID", proto.B58Bytes(id))
			}
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

type txIDSlogValuer struct {
	t      proto.Transaction
	scheme proto.Scheme
}

func (v txIDSlogValuer) LogValue() slog.Value {
	id, err := v.t.GetID(v.scheme)
	if err != nil {
		return slog.GroupValue(slog.Group("tx-get-id", logging.Error(err)))
	}
	return slog.StringValue(base58.Encode(id))
}

func txIDSlogAttr(t proto.Transaction, scheme proto.Scheme) slog.Attr {
	var val slog.LogValuer = txIDSlogValuer{t: t, scheme: scheme}
	return slog.Any("tx-id", val)
}
