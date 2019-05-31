package mainer

import (
	"bytes"
	"context"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mainer/scheduler"
	"github.com/wavesplatform/gowaves/pkg/mainer/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"time"
)

type Mainer struct {
	utx       *utxpool.Utx
	state     state.State
	peer      node.PeerManager
	scheduler types.Scheduler
	interrupt *atomic.Bool
}

func New(utx *utxpool.Utx, state state.State, peer node.PeerManager, scheduler types.Scheduler) *Mainer {
	return &Mainer{
		scheduler: scheduler,
		utx:       utx,
		state:     state,
		peer:      peer,
		interrupt: atomic.NewBool(false),
	}
}

func (a *Mainer) Mine(t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest) {
	a.interrupt.Store(false)
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

	b := proto.Block{
		BlockHeader: proto.BlockHeader{
			Version:                3,
			Timestamp:              t,
			Parent:                 parent,
			FeaturesCount:          0,   // ??
			Features:               nil, // ??
			ConsensusBlockLength:   40,  //  ??
			TransactionBlockLength: uint32(len(buf.Bytes()) + 4),
			TransactionCount:       len(transactions),
			GenPublicKey:           k.Public(),
			BlockSignature:         crypto.Signature{}, //

			NxtConsensus: proto.NxtConsensus{
				BaseTarget:   baseTarget,   // 8
				GenSignature: GenSignature, //
			},
		},
		Transactions: buf.Bytes(),
	}

	zap.S().Infof("%+v", b)

	buf = new(bytes.Buffer)
	_, err = b.WriteTo(buf)
	if err != nil {
		zap.S().Error(err)
		return
	}

	sign := crypto.Sign(k.Private(), buf.Bytes())
	buf.Write(sign[:])

	ba := node.NewBlockApplier(a.state, a.peer, a.scheduler, a)
	err = ba.Apply(buf.Bytes())
	if err != nil {
		zap.S().Error(err)
	}
}

func (a *Mainer) Interrupt() {
	a.interrupt.Store(true)
}

func Run(ctx context.Context, a *Mainer, s *scheduler.SchedulerImpl) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			a.Mine(v.Timestamp, v.KeyPair, v.ParentBlockSignature, v.BaseTarget, v.GenSignature)
		}
	}
}
