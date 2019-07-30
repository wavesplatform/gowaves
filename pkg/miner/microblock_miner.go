package miner

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type restLimits struct {
	MaxScriptRunsInBlock        int
	MaxScriptsComplexityInBlock int
	ClassicAmountOfTxsInBlock   int
	MaxTxsSizeInBytes           int
}

type MicroblockMiner struct {
	utx         *utxpool.Utx
	state       state.State
	peer        peer_manager.PeerManager
	scheduler   types.Scheduler
	interrupt   *atomic.Bool
	constraints Constraints
	ngRuntime   ng.Runtime
	scheme      proto.Scheme
}

func NewMicroblockMiner(utx *utxpool.Utx, state state.State, peer peer_manager.PeerManager, scheduler types.Scheduler, ngRuntime ng.Runtime, scheme proto.Scheme) *MicroblockMiner {
	return &MicroblockMiner{
		scheduler:   scheduler,
		utx:         utx,
		state:       state,
		peer:        peer,
		interrupt:   atomic.NewBool(false),
		constraints: DefaultConstraints(),
		ngRuntime:   ngRuntime,
		scheme:      scheme,
	}
}

func (a *MicroblockMiner) Mine(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest) {
	a.interrupt.Store(false)
	defer a.scheduler.Reschedule()

	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: GenSignature,
	}

	b, err := proto.CreateBlock(proto.NewReprFromTransactions(nil), t, parent, k.Public(), nxt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	err = b.Sign(k.Private())
	if err != nil {
		zap.S().Error(err)
		return
	}

	ba := node.NewBlockApplier(a.state, a.peer, a.scheduler)
	err = ba.Apply(b)
	if err != nil {
		zap.S().Error(err)
		return
	}

	curScore, err := a.state.CurrentScore()
	if err != nil {
		zap.S().Error(err)
		return
	}

	a.peer.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(&proto.ScoreMessage{
			Score: curScore.Bytes(),
		})
	})
	blockBytes, err := b.MarshalBinary()
	if err != nil {
		zap.S().Error(err)
		return
	}
	a.peer.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(&proto.BlockMessage{
			BlockBytes: blockBytes,
		})
	})

	rest := restLimits{
		MaxScriptRunsInBlock:        a.constraints.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: a.constraints.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   a.constraints.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           a.constraints.MaxTxsSizeInBytes - 4,
	}
	go a.mineMicro(ctx, rest, b, ng.NewBlocksFromBlock(b), k, a.scheme)
}

func (a *MicroblockMiner) Interrupt() {
	a.interrupt.Store(true)
}

func (a *MicroblockMiner) mineMicro(ctx context.Context, rest restLimits, blockApplyOn *proto.Block, blocks ng.Blocks, keyPair proto.KeyPair, scheme proto.Scheme) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	// way to stop mine microblocks
	if blockApplyOn == nil {
		return
	}

	height, err := a.state.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	lastBlock, err := a.state.BlockByHeight(height)
	if err != nil {
		zap.S().Error(err)
		return
	}

	if lastBlock.BlockSignature != blockApplyOn.BlockSignature {
		// block changed, exit
		return
	}

	bts_, err := blockApplyOn.Transactions.Bytes()
	if err != nil {
		zap.S().Error(err)
		return
	}
	bts := make([]byte, len(bts_))
	copy(bts, bts_)

	//
	bytesBuf := make([]byte, 0)
	cnt := 0

	var unAppliedTransactions []*utxpool.TransactionWithBytes

	mu := a.state.Mutex()
	locked := mu.Lock()
	currentTimestamp := proto.NewTimestampFromTime(time.Now())

	// 255 is max transactions count in microblock
	for i := 0; i < 255; i++ {
		t := a.utx.Pop()
		if t == nil {
			break
		}
		binTr := t.B
		transactionLenBytes := 4
		if len(bytesBuf)+len(binTr)+transactionLenBytes > rest.MaxTxsSizeInBytes {
			unAppliedTransactions = append(unAppliedTransactions, t)
			continue
		}

		err = a.state.ValidateNextTx(t.T, currentTimestamp, blockApplyOn.Timestamp)
		if err != nil {
			unAppliedTransactions = append(unAppliedTransactions, t)
			continue
		}

		cnt += 1
		bytesBuf = append(bytesBuf, trWithLen(binTr)...)
	}

	a.state.ResetValidationList()
	locked.Unlock()

	// return unapplied transactions
	for _, unapplied := range unAppliedTransactions {
		a.utx.AddWithBytes(unapplied.T, unapplied.B)
	}

	// no transactions applied, skip
	if cnt == 0 {
		go a.mineMicro(ctx, rest, blockApplyOn, blocks, keyPair, scheme)
		return
	}

	row, err := blocks.Row()
	if err != nil {
		zap.S().Error(err)
		return
	}

	var lastsig crypto.Signature
	if len(row.MicroBlocks) > 0 {
		lastsig = row.MicroBlocks[len(row.MicroBlocks)-1].TotalResBlockSigField
	} else {
		lastsig = row.KeyBlock.BlockSignature
	}

	transactions, err := blockApplyOn.Transactions.Join(proto.NewReprFromBytes(bytesBuf, cnt))
	if err != nil {
		zap.S().Error(err)
		return
	}

	newBlock, err := proto.CreateBlock(
		transactions,
		blockApplyOn.Timestamp,
		blockApplyOn.Parent,
		blockApplyOn.GenPublicKey,
		blockApplyOn.NxtConsensus)
	if err != nil {
		zap.S().Error(err)
		return
	}

	err = newBlock.Sign(keyPair.Private())
	if err != nil {
		zap.S().Error(err)
		return
	}

	locked = mu.Lock()
	_ = a.state.RollbackTo(blockApplyOn.Parent)
	locked.Unlock()

	ba := node.NewBlockApplier(a.state, a.peer, a.scheduler)
	err = ba.Apply(newBlock)
	if err != nil {
		zap.S().Error(err)
		return
	}

	micro := proto.MicroBlock{
		VersionField:          3,
		SenderPK:              keyPair.Public(),
		Transactions:          proto.NewReprFromBytes(bytesBuf, cnt),
		TransactionCount:      uint32(cnt),
		PrevResBlockSigField:  lastsig,
		TotalResBlockSigField: newBlock.BlockSignature,
	}

	err = micro.Sign(keyPair.Private())
	if err != nil {
		zap.S().Error(err)
		return
	}

	inv := proto.NewUnsignedMicroblockInv(micro.SenderPK, micro.TotalResBlockSigField, micro.PrevResBlockSigField)
	err = inv.Sign(keyPair.Private(), scheme)
	if err != nil {
		zap.S().Error(err)
		return
	}

	a.ngRuntime.MinedMicroblock(&micro, inv)

	newRest := restLimits{
		MaxScriptRunsInBlock:        rest.MaxScriptRunsInBlock,
		MaxScriptsComplexityInBlock: rest.MaxScriptsComplexityInBlock,
		ClassicAmountOfTxsInBlock:   rest.ClassicAmountOfTxsInBlock,
		MaxTxsSizeInBytes:           rest.MaxTxsSizeInBytes - len(bytesBuf),
	}

	newBlocks, err := blocks.AddMicro(&micro)
	if err != nil {
		zap.S().Error(err)
		return
	}

	go a.mineMicro(ctx, newRest, newBlock, newBlocks, keyPair, scheme)
}

func trWithLen(bts []byte) []byte {
	out := make([]byte, len(bts)+4)
	binary.BigEndian.PutUint32(out[:4], uint32(len(bts)))
	copy(out[4:], bts)
	return out
}
