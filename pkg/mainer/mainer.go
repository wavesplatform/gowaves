package mainer

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mainer/pool"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type timestamp = uint64

type Mainer struct {
	utx *pool.Utx
}

func (a *Mainer) Mine(t timestamp, k KeyPair, parent crypto.Signature, baseTarget consensus.BaseTarget, GenSignature crypto.Digest) {

	b := proto.Block{}

	blockHeader := proto.BlockHeader{
		Version:                3,
		Timestamp:              t,
		Parent:                 parent,
		FeaturesCount:          0,   // ??
		Features:               nil, // ??
		ConsensusBlockLength:   0,   //  ??
		TransactionBlockLength: 0,   // ??
		TransactionCount:       0,
		GenPublicKey:           k.Public(),
		BlockSignature:         crypto.Signature{}, //

		NxtConsensus: proto.NxtConsensus{
			BaseTarget:   baseTarget,
			GenSignature: GenSignature,
		},
	}

	var transactions []proto.Transaction
	var invalidTransactions []proto.Transaction
	for i := 0; i < 100; i++ {
		t := a.utx.Pop()
		if t == nil {
			break
		}

		transactions = append(transactions, t)
	}

}

func Run(ctx context.Context, a *Mainer, s *Scheduler) {
	for {
		select {
		case <-ctx.Done():
			return
		case v := <-s.Mine():
			a.Mine(v.Timestamp, v.KeyPair, v.ParentBlockSignature, v.BaseTarget, v.GenSignature)
		}
	}
}
