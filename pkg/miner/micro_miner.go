package miner

import (
	"errors"

	"github.com/mr-tron/base58"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/crypto"
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
}

func NewMicroMiner(services services.Services) *MicroMiner {
	return &MicroMiner{
		state:  services.State,
		utx:    services.UtxPool,
		scheme: services.Scheme,
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
	zap.S().Debugf("[MICRO MINER] Top block ID '%s'", topBlock.BlockID())

	height, err := a.state.Height()
	if err != nil {
		return nil, nil, rest, err
	}
	zap.S().Debugf("[MICRO MINER] Height %d", height)

	parentTimestamp := topBlock.Timestamp
	if height > 1 {
		parent, err := a.state.BlockByHeight(height - 1)
		if err != nil {
			return nil, nil, rest, err
		}
		parentTimestamp = parent.Timestamp
	}

	//
	txCount := 0
	binSize := 0

	var appliedTransactions []*types.TransactionWithBytes
	var inapplicable []*types.TransactionWithBytes
	var txSnapshots [][]proto.AtomicSnapshot

	_ = a.state.Map(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()

		for txCount <= maxMicroblockTransactions {
			t := a.utx.Pop()
			if t == nil {
				break
			}
			binTr := t.B
			transactionLenBytes := 4
			if binSize+len(binTr)+transactionLenBytes > rest.MaxTxsSizeInBytes {
				inapplicable = append(inapplicable, t)
				continue
			}

			// In the miner we pack transactions from UTX into new block.
			// We should accept failed transactions here.
			snapshot, errVal := s.ValidateNextTx(t.T, minedBlock.Timestamp, parentTimestamp, minedBlock.Version, true)
			if stateerr.IsTxCommitmentError(errVal) {
				// This should not happen in practice.
				// Reset state, tx count, return applied transactions to UTX.
				s.ResetValidationList()
				txCount = 0
				for _, appliedTx := range appliedTransactions {
					_ = a.utx.AddWithBytes(appliedTx.T, appliedTx.B)
				}
				appliedTransactions = nil
				txSnapshots = nil
				continue
			}
			if errVal != nil {
				inapplicable = append(inapplicable, t)
				continue
			}

			txCount += 1
			binSize += len(binTr) + transactionLenBytes
			appliedTransactions = append(appliedTransactions, t)
			txSnapshots = append(txSnapshots, snapshot)
		}
		return nil
	})

	// return inapplicable transactions
	for _, tx := range inapplicable {
		_ = a.utx.AddWithBytes(tx.T, tx.B)
	}

	// no transactions applied, skip
	if txCount == 0 {
		return nil, nil, rest, ErrNoTransactions
	}

	transactions := make([]proto.Transaction, len(appliedTransactions))
	for i, appliedTx := range appliedTransactions {
		if zap.S().Level() <= zap.DebugLevel {
			if id, idErr := appliedTx.T.GetID(a.scheme); idErr != nil {
				zap.S().Errorf("Failed to get transaction ID: %v", idErr)
			} else {
				zap.S().Debugf("[MICRO MINER] Appending transaction '%s'", base58.Encode(id))
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

	zap.S().Debugf("micro_miner mined %+v", micro)

	newRest := proto.MiningLimits{
		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - binSize,
	}
	return newBlock, &micro, newRest, nil
}
