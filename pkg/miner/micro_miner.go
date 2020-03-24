package miner

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var NoTransactionsErr = errors.New("no transactions")

type MicroMiner struct {
	state  state.State
	utx    types.UtxPool
	scheme proto.Scheme
}

func (a *MicroMiner) Micro(
	minedBlock *proto.Block,
	rest proto.MiningLimits,
	blocks ng.Blocks,
	keyPair proto.KeyPair) (*proto.Block, *proto.MicroBlock, proto.MiningLimits, error) {

	// way to stop mine microblocks
	if minedBlock == nil {
		return nil, nil, rest, errors.New("no block provided")
	}

	//height, err := a.state.Height()
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}
	//
	//topBlock, err := a.state.BlockByHeight(height)
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}
	topBlock := a.state.TopBlock()
	rlocked := a.state.Mutex().RLock()
	height, err := a.state.Height()
	rlocked.Unlock()
	if err != nil {
		return nil, nil, rest, err
	}

	if topBlock.BlockSignature != minedBlock.BlockSignature {
		// block changed, exit
		return nil, nil, rest, errors.New("block changed")
	}
	parentTimestamp := topBlock.Timestamp
	if height > 1 {
		parent, err := a.state.BlockByHeight(height - 1)
		if err != nil {
			return nil, nil, rest, err
		}
		parentTimestamp = parent.Timestamp
	}

	//
	transactions := make([]proto.Transaction, 0)
	cnt := 0
	binSize := 0

	var unAppliedTransactions []*types.TransactionWithBytes

	mu := a.state.Mutex()
	locked := mu.Lock()

	// 255 is max transactions count in microblock
	for i := 0; i < 255; i++ {
		t := a.utx.Pop()
		if t == nil {
			break
		}
		binTr := t.B
		transactionLenBytes := 4
		if binSize+len(binTr)+transactionLenBytes > rest.MaxTxsSizeInBytes {
			unAppliedTransactions = append(unAppliedTransactions, t)
			continue
		}

		err = a.state.ValidateNextTx(t.T, minedBlock.Timestamp, parentTimestamp, minedBlock.Version)
		if err != nil {
			unAppliedTransactions = append(unAppliedTransactions, t)
			continue
		}

		cnt += 1
		binSize += len(binTr) + transactionLenBytes
		transactions = append(transactions, t.T)
	}

	a.state.ResetValidationList()
	locked.Unlock()

	// return unapplied transactions
	for _, unapplied := range unAppliedTransactions {
		_ = a.utx.AddWithBytes(unapplied.T, unapplied.B)
	}

	// no transactions applied, skip
	if cnt == 0 {
		return nil, nil, rest, NoTransactionsErr
	}
	row := blocks.Row()
	lastsig := row.LastSignature()
	newTransactions := minedBlock.Transactions.Join(transactions)

	newBlock, err := proto.CreateBlock(
		newTransactions,
		minedBlock.Timestamp,
		minedBlock.Parent,
		minedBlock.GenPublicKey,
		minedBlock.NxtConsensus,
		minedBlock.Version,
		minedBlock.Features,
		minedBlock.RewardVote,
		a.scheme,
	)
	if err != nil {
		return nil, nil, rest, err
	}

	sk := keyPair.Secret
	err = newBlock.Sign(a.scheme, keyPair.Secret)
	if err != nil {
		return nil, nil, rest, err
	}

	//locked = mu.Lock()
	//_ = a.state.RollbackTo(minedBlock.Parent)
	//locked.Unlock()

	//err = a.services.BlocksApplier.Apply(a.state, []*proto.Block{newBlock})
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}

	micro := proto.MicroBlock{
		VersionField:          3,
		SenderPK:              keyPair.Public,
		Transactions:          transactions,
		TransactionCount:      uint32(cnt),
		PrevResBlockSigField:  lastsig,
		TotalResBlockSigField: newBlock.BlockSignature,
	}

	err = micro.Sign(sk)
	if err != nil {
		return nil, nil, rest, err
	}

	inv := proto.NewUnsignedMicroblockInv(micro.SenderPK, micro.TotalResBlockSigField, micro.PrevResBlockSigField)
	err = inv.Sign(sk, a.scheme)
	if err != nil {
		return nil, nil, rest, err
	}

	// TODO implement
	//a.ngRuntime.MinedMicroblock(&micro, inv)

	newRest := proto.MiningLimits{
		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - binSize,
	}

	_, err = blocks.AddMicro(&micro)
	if err != nil {
		return nil, nil, rest, err
	}

	return newBlock, &micro, newRest, nil

	//go a.mineMicro(ctx, newRest, newBlock, newBlocks, keyPair)
}
