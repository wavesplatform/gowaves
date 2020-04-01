package miner

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	//"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type MicroblockMiner struct {
	utx         types.UtxPool
	state       state.State
	peer        peer_manager.PeerManager
	scheduler   types.Scheduler
	constraints Constraints
	//ngRuntime   ng.Runtime
	scheme   proto.Scheme
	services services.Services
	features Features
	// reward vote 600000000
	reward int64
}

func NewMicroblockMiner(services services.Services, features Features, reward int64) *MicroblockMiner {
	return &MicroblockMiner{
		scheduler:   services.Scheduler,
		utx:         services.UtxPool,
		state:       services.State,
		peer:        services.Peers,
		constraints: DefaultConstraints(),
		//ngRuntime:   ngRuntime,
		scheme:   services.Scheme,
		services: services,
		features: features,
		reward:   reward,

		//
		//messageCH: messageCH,
	}
}

func (a *MicroblockMiner) MineKeyBlock(
	ctx context.Context,
	t proto.Timestamp,
	k proto.KeyPair,
	parent proto.BlockID,
	baseTarget types.BaseTarget,
	GenSignature []byte) (*proto.Block, proto.MiningLimits, error) {

	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: GenSignature,
	}

	b, err := func() (*proto.Block, error) {
		defer a.state.Mutex().RLock().Unlock()
		v, err := blockVersion(a.state)
		if err != nil {
			return nil, err
		}
		validatedFeatured, err := ValidateFeaturesWithoutLock(a.state, a.features)
		if err != nil {
			return nil, err
		}
		b, err := MineBlock(v, nxt, k, validatedFeatured, t, parent, a.reward, a.scheme)
		if err != nil {
			return nil, err
		}
		return b, nil
	}()
	if err != nil {
		return nil, proto.MiningLimits{}, err
	}

	//blockBytes, err := b.MarshalBinary()
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}

	//err = a.services.BlocksApplier.Apply(a.state, []*proto.Block{b})
	//if err != nil {
	//	zap.S().Errorf("Miner: applying created block: %q, timestamp %d", err, t)
	//	return
	//}
	zap.S().Debugf("Miner: generated new block id: %s, time: %d", b.BlockID().String(), t)

	//locked := a.state.Mutex().RLock()
	//curScore, err := a.state.CurrentScore()
	//locked.Unlock()
	//if err != nil {
	//	zap.S().Error(err)
	//	return
	//}

	//zap.S().Debugf("Miner: generated new block sig: %s, time: %d", b.BlockSignature, t)
	//
	//a.peer.EachConnected(func(peer peer.Peer, score *proto.Score) {
	//	peer.SendMessage(&proto.ScoreMessage{
	//		Score: curScore.Bytes(),
	//	})
	//})
	//msg, err := proto.MessageByBlock(b, a.scheme)
	//if err != nil {
	//zap.S().Error(err)
	//return
	//}

	//a.messageCH <- peer.ProtoMessage{
	//	ID:      nil,
	//	Message: msg,
	//}

	//a.peer.EachConnected(func(peer peer.Peer, score *proto.Score) {
	//	peer.SendMessage(msg)
	//})

	rest := proto.MiningLimits{
		MaxScriptRunsInBlock:        a.constraints.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: a.constraints.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   a.constraints.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           a.constraints.MaxTxsSizeInBytes - 4,
	}
	//_ = rest
	//go a.mineMicro(ctx, rest, b, ng.NewBlocksFromBlock(b), k)
	return b, rest, nil
}

//func (a *MicroblockMiner) mineMicro(ctx context.Context, rest proto.MiningLimits, minedBlock *proto.Block, keyPair proto.KeyPair) {
//	//select {
//	//case <-ctx.Done():
//	//	return
//	//case <-time.After(5 * time.Second):
//	//}
//
//	// way to stop mine microblocks
//	if minedBlock == nil {
//		return
//	}
//
//	//height, err := a.state.Height()
//	//if err != nil {
//	//	zap.S().Error(err)
//	//	return
//	//}
//	//
//	//topBlock, err := a.state.BlockByHeight(height)
//	//if err != nil {
//	//	zap.S().Error(err)
//	//	return
//	//}
//	topBlock := a.state.TopBlock()
//	rlocked := a.state.Mutex().RLock()
//	height, err := a.state.Height()
//	rlocked.Unlock()
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	if topBlock.BlockID() != minedBlock.BlockID() {
//		// block changed, exit
//		return
//	}
//	parentTimestamp := topBlock.Timestamp
//	if height > 1 {
//		parent, err := a.state.BlockByHeight(height - 1)
//		if err != nil {
//			zap.S().Error(err)
//			return
//		}
//		parentTimestamp = parent.Timestamp
//	}
//
//	//
//	transactions := make([]proto.Transaction, 0)
//	cnt := 0
//	binSize := 0
//
//	var unAppliedTransactions []*types.TransactionWithBytes
//
//	mu := a.state.Mutex()
//	locked := mu.Lock()
//
//	// 255 is max transactions count in microblock
//	for i := 0; i < 255; i++ {
//		t := a.utx.Pop()
//		if t == nil {
//			break
//		}
//		binTr := t.B
//		transactionLenBytes := 4
//		if binSize+len(binTr)+transactionLenBytes > rest.MaxTxsSizeInBytes {
//			unAppliedTransactions = append(unAppliedTransactions, t)
//			continue
//		}
//
//		err = a.state.ValidateNextTx(t.T, minedBlock.Timestamp, parentTimestamp, minedBlock.Version)
//		if err != nil {
//			unAppliedTransactions = append(unAppliedTransactions, t)
//			continue
//		}
//
//		cnt += 1
//		binSize += len(binTr) + transactionLenBytes
//		transactions = append(transactions, t.T)
//	}
//
//	a.state.ResetValidationList()
//	locked.Unlock()
//
//	// return unapplied transactions
//	for _, unapplied := range unAppliedTransactions {
//		_ = a.utx.AddWithBytes(unapplied.T, unapplied.B)
//	}
//
//	// no transactions applied, skip
//	if cnt == 0 {
//		go a.mineMicro(ctx, rest, minedBlock, keyPair)
//		return
//	}
//
//	//row := blocks.Row()
//	//if err != nil {
//	//	zap.S().Error(err)
//	//	return
//	//}
//	//
//	//var ref proto.BlockID
//	//if len(row.MicroBlocks) > 0 {
//	//	ref = row.MicroBlocks[len(row.MicroBlocks)-1].TotalBlockID
//	//} else {
//	//	ref = row.KeyBlock.BlockID()
//	//}
//
//	newTransactions := minedBlock.Transactions.Join(transactions)
//
//	newBlock, err := proto.CreateBlock(
//		newTransactions,
//		minedBlock.Timestamp,
//		minedBlock.Parent,
//		minedBlock.GenPublicKey,
//		minedBlock.NxtConsensus,
//		minedBlock.Version,
//		minedBlock.Features,
//		minedBlock.RewardVote,
//		a.scheme,
//	)
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	sk := keyPair.Secret
//	err = newBlock.Sign(a.scheme, keyPair.Secret)
//	if err != nil {
//		zap.S().Errorf("Failed to sing a block: %v", err)
//		return
//	}
//
//	locked = mu.Lock()
//	_ = a.state.RollbackTo(minedBlock.Parent)
//	locked.Unlock()
//
//	err = a.services.BlocksApplier.Apply(a.state, []*proto.Block{newBlock})
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	micro := proto.MicroBlock{
//		VersionField:          byte(newBlock.Version),
//		SenderPK:              keyPair.Public,
//		Transactions:          transactions,
//		TransactionCount:      uint32(cnt),
//		Reference:             topBlock.BlockID(),
//		TotalResBlockSigField: newBlock.BlockSignature,
//		TotalBlockID:          newBlock.BlockID(),
//	}
//
//	err = micro.Sign(sk)
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	inv := proto.NewUnsignedMicroblockInv(micro.SenderPK, micro.TotalBlockID, micro.Reference)
//	err = inv.Sign(sk, a.scheme)
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	// TODO implement
//	//a.ngRuntime.MinedMicroblock(&micro, inv)
//
//	newRest := proto.MiningLimits{
//		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
//		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
//		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
//		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - binSize,
//	}
//
//	newBlocks, err := blocks.AddMicro(&micro)
//	if err != nil {
//		zap.S().Error(err)
//		return
//	}
//
//	go a.mineMicro(ctx, newRest, newBlock, newBlocks, keyPair)
//}

func blockVersion(state state.State) (proto.BlockVersion, error) {
	blockV5Activated, err := state.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return 0, err
	}
	if blockV5Activated {
		return proto.ProtoBlockVersion, nil
	}
	height, err := state.Height()
	if err != nil {
		return 0, err
	}
	blockRewardActivated, err := state.IsActiveAtHeight(int16(settings.BlockReward), height)
	if err != nil {
		return 0, err
	}
	if blockRewardActivated {
		return proto.RewardBlockVersion, nil
	}
	return proto.NgBlockVersion, nil
}

func Run(ctx context.Context, a types.Miner, s *scheduler.SchedulerImpl, internalCh chan messages.InternalMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			block, limits, err := a.MineKeyBlock(ctx, v.Timestamp, v.KeyPair, v.Parent, v.BaseTarget, v.GenSignature)
			if err != nil {
				zap.S().Error(err)
				continue
			}
			internalCh <- messages.NewMinedBlockInternalMessage(block, limits, v.KeyPair)
		}
	}
}
