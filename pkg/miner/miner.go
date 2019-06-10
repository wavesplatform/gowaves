package miner

import (
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"bytes"
	"context"
	"time"
)

type Miner struct {
	utx       *utxpool.Utx
	state     state.State
	peer      node.PeerManager
	scheduler types.Scheduler
	interrupt *atomic.Bool
}

func New(utx *utxpool.Utx, state state.State, peer node.PeerManager, scheduler types.Scheduler) *Miner {
	return &Miner{
		scheduler: scheduler,
		utx:       utx,
		state:     state,
		peer:      peer,
		interrupt: atomic.NewBool(false),
	}
}

func (a *Miner) Mine(t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest) {
	a.interrupt.Store(false)
	defer a.scheduler.Reschedule()
	lastKnownBlock, err := a.state.Block(parent)
	if err != nil {
		zap.S().Error(err)
		return
	}

	transactions := proto.Transactions{}
	var invalidTransactions []proto.Transaction
	currentTimestamp := proto.NewTimestampFromTime(time.Now())
	mu := a.state.Mutex()
	mu.Lock()
	for i := 0; i < 100; i++ {
		t := a.utx.Pop()
		if t == nil {
			break
		}

		if a.interrupt.Load() {
			a.state.ResetValidationList()
			mu.Unlock()
			return
		}

		if err = a.state.ValidateNextTx(t, currentTimestamp, lastKnownBlock.Timestamp); err != nil {
			invalidTransactions = append(invalidTransactions, t)
		} else {
			transactions = append(transactions, t)
		}
	}
	a.state.ResetValidationList()
	mu.Unlock()

	buf := new(bytes.Buffer)
	_, err = transactions.WriteTo(buf)
	if err != nil {
		return
	}

	nxt := proto.NxtConsensus{
		BaseTarget:   baseTarget,
		GenSignature: GenSignature,
	}

	b, err := proto.BlockBuilder(transactions, t, parent, k.Public(), nxt)
	if err != nil {
		zap.S().Error(err)
		return
	}

	err = b.Sign(k.Private())
	if err != nil {
		zap.S().Error(err)
		return
	}

	ba := node.NewBlockApplier(a.state, a.peer, a.scheduler, a)
	err = ba.Apply(b)
	if err != nil {
		zap.S().Error(err)
	}
}

func (a *Miner) Interrupt() {
	a.interrupt.Store(true)
}

func Run(ctx context.Context, a *Miner, s *scheduler.SchedulerImpl) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			a.Mine(v.Timestamp, v.KeyPair, v.ParentBlockSignature, v.BaseTarget, v.GenSignature)
		}
	}
}
