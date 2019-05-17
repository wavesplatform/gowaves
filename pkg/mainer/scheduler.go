package mainer

import (
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"go.uber.org/zap"
	"time"
)

type Emit struct {
	Timestamp            uint64
	KeyPair              KeyPair
	GenSignature         crypto.Digest
	BaseTarget           consensus.BaseTarget
	ParentBlockSignature crypto.Signature
}

type Scheduler struct {
	keyPairs []KeyPair
	mine     chan Emit
	cancel   []func()
	settings settings.BlockchainSettings
}

func (a *Scheduler) Mine() chan Emit {
	return a.mine
}

func (a *Scheduler) Reschedule(state state.State, curBlock *proto.Block, height uint64) {
	if len(a.keyPairs) == 0 {
		return
	}

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil

	//
	parentBlock, err := state.BlockByHeight(height - 1)
	if err != nil {
		zap.S().Error(err)
		return
	}

	greatGrandParent, err := state.BlockByHeight(height - 3)
	if err != nil {
		zap.S().Error(err)
		return
	}

	for _, keyPair := range a.keyPairs {
		genSig, err := consensus.GeneratorSignature(curBlock.BlockHeader.GenSignature, keyPair.Public())
		if err != nil {
			zap.S().Error(err)
			continue
		}

		hit, err := consensus.GenHit(genSig[:])
		if err != nil {
			zap.S().Error(err)
			continue
		}

		c := &consensus.FairPosCalculator{}
		baseTarget, err := c.CalculateBaseTarget(a.settings.AverageBlockDelaySeconds, height, curBlock.BlockHeader.BaseTarget, parentBlock.Timestamp, greatGrandParent.Timestamp, proto.NewTimestampFromTime(time.Now()))
		if err != nil {
			zap.S().Error(err)
			continue
		}

		effectiveBalance, err := state.EffectiveBalance(keyPair.Addr(), height-1000, height)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		delay, err := c.CalculateDelay(hit, baseTarget, effectiveBalance)
		if err != nil {
			zap.S().Error(err)
			continue
		}

		now := proto.NewTimestampFromTime(time.Now())
		if curBlock.Timestamp+delay > now { // timestamp in future
			// delta from now to future
			timeout := curBlock.Timestamp + delay - now
			// start timeout before emit mine
			cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
				a.mine <- Emit{
					Timestamp:            curBlock.Timestamp + delay,
					KeyPair:              keyPair,
					GenSignature:         genSig,
					BaseTarget:           baseTarget,
					ParentBlockSignature: curBlock.BlockSignature,
				}
			})
			a.cancel = append(a.cancel, cancel)
		} else {
			a.mine <- Emit{
				Timestamp:            curBlock.Timestamp + delay,
				KeyPair:              keyPair,
				GenSignature:         genSig,
				BaseTarget:           baseTarget,
				ParentBlockSignature: curBlock.BlockSignature,
			}
		}
	}
}
