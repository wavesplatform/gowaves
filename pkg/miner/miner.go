package miner

import (
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"bytes"
	"context"
	"time"
)

type Miner interface {
	Mine(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest)
}

type DefaultMiner struct {
	utx       *utxpool.Utx
	state     state.State
	peer      peer_manager.PeerManager
	scheduler types.Scheduler
	interrupt *atomic.Bool
}

func NewDefaultMiner(utx *utxpool.Utx, state state.State, peer peer_manager.PeerManager, scheduler types.Scheduler) *DefaultMiner {
	return &DefaultMiner{
		scheduler: scheduler,
		utx:       utx,
		state:     state,
		peer:      peer,
		interrupt: atomic.NewBool(false),
	}
}

func (a *DefaultMiner) Mine(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest) {
	a.interrupt.Store(false)
	defer a.scheduler.Reschedule()
	lastKnownBlock, err := a.state.Block(parent)
	if err != nil {
		zap.S().Error(err)
		return
	}

	transactions := proto.Transactions{}
	var invalidTransactions []*utxpool.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(time.Now())
	mu := a.state.Mutex()
	locked := mu.Lock()
	for i := 0; i < 100; i++ {
		t := a.utx.Pop()
		if t == nil {
			break
		}

		if a.interrupt.Load() {
			a.state.ResetValidationList()
			locked.Unlock()
			return
		}

		if err = a.state.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp); err != nil {
			invalidTransactions = append(invalidTransactions, t)
		} else {
			transactions = append(transactions, t.T)
		}
	}
	a.state.ResetValidationList()
	locked.Unlock()

	buf := new(bytes.Buffer)
	_, err = transactions.WriteTo(buf)
	if err != nil {
		return
	}

	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: GenSignature,
	}

	b, err := proto.CreateBlock(proto.NewReprFromTransactions(transactions), t, parent, k.Public(), nxt)
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
	}
}

func (a *DefaultMiner) Interrupt() {
	a.interrupt.Store(true)
}

func Run(ctx context.Context, a Miner, s *scheduler.SchedulerImpl) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			a.Mine(ctx, v.Timestamp, v.KeyPair, v.ParentBlockSignature, v.BaseTarget, v.GenSignature)
		}
	}
}

type noOpMiner struct {
}

func (noOpMiner) Interrupt() {
}

func NoOpMiner() noOpMiner {
	return noOpMiner{}
}
