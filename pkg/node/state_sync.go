package node

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type interrupt = chan struct{}

type StateSync struct {
	peerManager  peer_manager.PeerManager
	stateManager state.State
	subscribe    *Subscribe
	interrupt    interrupt
	scheduler    types.Scheduler
	blockApplier types.BlockApplier
	interrupter  types.MinerInterrupter
	syncCh       chan struct{}
	services     services.Services
}

func NewStateSync(services services.Services, subscribe *Subscribe, interrupter types.MinerInterrupter) *StateSync {
	return &StateSync{
		peerManager:  services.Peers,
		stateManager: services.State,
		subscribe:    subscribe,
		interrupt:    make(chan struct{}),
		scheduler:    services.Scheduler,
		blockApplier: services.BlockApplier,
		interrupter:  interrupter,
		syncCh:       make(chan struct{}, 20),
		services:     services,
	}
}

var TimeoutErr = errors.Errorf("Timeout")

func (a *StateSync) Sync() {
	select {
	case a.syncCh <- struct{}{}:
		return
	default:
		zap.S().Error("failed add sync job, chan is full")
	}
}

func (a *StateSync) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.syncCh:
			zap.S().Info("StateSync.Run: <-a.syncCh")
			a.run(ctx)
		}
	}
}

func (a *StateSync) run(ctx context.Context) {
	for {
		p, err := a.getPeerWithHighestScore()
		if err != nil {
			return
		}
		_ = a.sync(ctx, p)
	}
}

func (a *StateSync) sync(ctx context.Context, p Peer) error {
	ctx, cancel := context.WithCancel(ctx)
	sigs, err := LastSignatures(a.stateManager)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return err
	}

	errCh := make(chan error, 2)
	incoming := make(chan crypto.Signature, 256)

	go func() {
		errCh <- PreloadSignatures(ctx, incoming, p, sigs, a.subscribe)
	}()
	go func() {
		errCh <- downloadBlocks(ctx, incoming, p, a.subscribe, a.services, a.interrupter)
	}()

	err = <-errCh
	if err != nil {
		// TODO switch on error type, maybe suspend node
		zap.S().Error(err)
	}
	cancel()

	return nil
}

func (a *StateSync) getPeerWithHighestScore() (Peer, error) {
	p, score, ok := a.peerManager.PeerWithHighestScore()
	if !ok || score.String() == "0" {
		// no peers, skip
		zap.S().Info("no peers, skip ", score.String())
		return nil, errors.Errorf("no score found")
	}

	// compare my score with highest known
	myScore, err := a.stateManager.CurrentScore()
	if err != nil {
		return nil, err
	}

	if myScore.Cmp(score) >= 0 {
		return nil, errors.Errorf("we have highest score, nothing to do")
	}
	return p, nil
}

func (a *StateSync) Close() {
	close(a.interrupt)
}

func downloadBlocks(
	ctx context.Context,
	signaturesCh chan crypto.Signature,
	p Peer,
	subscribe *Subscribe,
	services services.Services,
	interrupt types.MinerInterrupter) error {

	defer services.Scheduler.Reschedule()

	errCh := make(chan error, 1)

	receivedBlocksCh := make(chan blockBytes, 256)

	downloader := newBlockDownloader(64, p, subscribe, receivedBlocksCh)
	go downloader.run(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(15 * time.Second):
				// TODO handle timeout
				zap.S().Info("timeout waiting &proto.SignaturesMessage{}")
				errCh <- TimeoutErr
				return
			case sig := <-signaturesCh:
				downloader.download(sig)
			}
		}
	}()

	go func() {
		for {
			score, err := services.Peers.Score(p)
			if err != nil {
				errCh <- err
				return
			}

			curScore, err := services.State.CurrentScore()
			if err != nil {
				errCh <- err
				return
			}

			if score.Cmp(curScore) == 0 {
				errCh <- nil
				return
			}

			select {
			case <-ctx.Done():
				return

			case bts := <-receivedBlocksCh:
				interrupt.Interrupt()
				err := services.BlockApplier.ApplyBytes(bts)
				if err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	select {
	case mess := <-errCh:
		return mess
	case <-ctx.Done():
		return ctx.Err()
	}
}
