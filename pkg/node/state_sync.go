package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/nullable"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"

	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type interrupt = chan struct{}

type StateSync struct {
	peerManager         peer_manager.PeerManager
	stateManager        state.State
	subscribe           types.Subscribe
	interrupt           interrupt
	scheduler           types.Scheduler
	interrupter         types.MinerInterrupter
	syncCh              chan struct{}
	services            services.Services
	scoreSender         types.ScoreSender
	historyBlockApplier HistoryBlockApplier

	// need to enable or disable sync
	mu      sync.Mutex
	enabled bool
}

func NewStateSync(services services.Services, interrupter types.MinerInterrupter, scoreSender types.ScoreSender) *StateSync {
	return &StateSync{
		peerManager:         services.Peers,
		stateManager:        services.State,
		subscribe:           services.Subscribe,
		interrupt:           make(chan struct{}, 1),
		scheduler:           services.Scheduler,
		interrupter:         interrupter,
		syncCh:              make(chan struct{}, 20),
		services:            services,
		enabled:             true,
		scoreSender:         scoreSender,
		historyBlockApplier: NewHistoryBlockApplier(services, interrupter, scoreSender),
	}
}

var TimeoutErr = errors.New("timeout")

func (a *StateSync) Sync() {
	a.mu.Lock()
	enabled := a.enabled
	a.mu.Unlock()
	if !enabled {
		return
	}
	select {
	case a.syncCh <- struct{}{}:
		return
	default:
	}
}

func (a *StateSync) SetEnabled(enabled bool) {
	a.mu.Lock()
	a.enabled = enabled
	a.mu.Unlock()
	if enabled {
		zap.S().Debug("StateSync: enabled")
	} else {
		zap.S().Debug("StateSync: disabled")
	}
}

func (a *StateSync) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.syncCh:
			a.run(ctx)
		}
	}
}

func (a *StateSync) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		p, err := a.getPeerWithHighestScore()
		if err != nil {
			return
		}
		zap.S().Infof("[%s] StateSync: Starting synchronization with %s", p.ID(), p.ID())
		err = a.sync(ctx, p)
		zap.S().Infof("[%s] StateSync: Ended with code %q", p.ID(), err)
		if err != nil {
			<-time.After(10 * time.Second)
			continue
		}
		return
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
	incoming := make(chan nullable.Signature, 256)

	var wg sync.WaitGroup

	a.services.LoggableRunner.Named("StateSync.PreloadSignatures", func() {
		errCh <- PreloadSignatures(ctx, incoming, p, sigs, a.subscribe, &wg)
	})
	a.services.LoggableRunner.Named("StateSync.downloadBlocks", func() {
		errCh <- a.downloadBlocks(ctx, incoming, p, &wg)
	})

	err = <-errCh
	switch err {
	case TimeoutErr:
		a.peerManager.Suspend(p, err.Error())
		cancel()
		go func() {
			<-time.After(2 * time.Second)
			a.Sync()
		}()
	default:
		if err != nil {
			cancel()

			a.peerManager.Suspend(p, fmt.Sprintf("switch default, %s", err.Error()))
			zap.S().Errorf("[%s] StateSync: Error: %v", p.ID(), err)
		}
	}

	wg.Wait()
	cancel()
	zap.S().Debugf("StateSync: done waiting")

	return err
}

func (a *StateSync) getPeerWithHighestScore() (Peer, error) {
	p, score, ok := a.peerManager.PeerWithHighestScore()
	if !ok || score.String() == "0" {
		// no peers, skip
		zap.S().Infof("StateSync: no peers, skip %s", score.String())
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

func (a *StateSync) downloadBlocks(ctx context.Context, signaturesCh chan nullable.Signature, p Peer, wg *sync.WaitGroup) error {
	runner := a.services.LoggableRunner
	defer a.services.Scheduler.Reschedule()
	wg.Add(1)
	defer wg.Done()

	errCh := make(chan error, 3)
	receivedBlocksCh := make(chan blockBytes, 128)

	downloader := newBlockDownloader(128, p, a.subscribe, receivedBlocksCh)
	runner.Named("StateSync.downloadBlocks.downloader.run", func() {
		downloader.run(ctx, wg)
	})

	const blockCnt = 50

	wg.Add(1)
	runner.Named("StateSync.downloadBlocks.receiveSignatures",
		func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					zap.S().Infof("[%s] StateSync: DownloadBlocks: timeout waiting for SignaturesMessage", p.ID())
					errCh <- TimeoutErr
					return
				case sig := <-signaturesCh:
					downloader.download(sig)
					if sig.Null() {
						return
					}
				}
			}
		})

	blocksBulk := make(chan [][]byte, 1)

	wg.Add(1)
	runner.Named("StateSync.downloadBlocks.CreateBulk", func() {
		defer wg.Done()
		select {
		case errCh <- createBulkWorker(ctx, blockCnt, receivedBlocksCh, blocksBulk):
		default:
		}
	})

	// block applier
	wg.Add(1)
	runner.Named("StateSync.downloadBlocks.ApplyBlocks", func() {
		defer wg.Done()
		select {
		case errCh <- applyWorker(ctx, blockCnt, blocksBulk, a.historyBlockApplier):
		default:
		}
	})

	select {
	case mess := <-errCh:
		return mess
	case <-ctx.Done():
		return ctx.Err()
	}
}

func applyWorker(ctx context.Context, blockCnt int, blocksBulk chan [][]byte, applier HistoryBlockApplier) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case blocks := <-blocksBulk:
			zap.S().Debugf("[*] BlockDownloader: Applying: received %d blocks", len(blocks))
			if len(blocks) == 0 {
				return nil
			}
			err := applier.ApplyBlocksBytes(blocks)
			if err != nil {
				return err
			}
			// received less than expected, it means successful exit
			if len(blocks) < blockCnt {
				return nil
			}
		}
	}
}

func createBulkWorker(ctx context.Context, blockCnt int, receivedBlocksCh chan blockBytes, blocksBulk chan []blockBytes) error {
	blocks := make([][]byte, 0, blockCnt)
	for {
		select {
		case <-ctx.Done():
			return nil
		case bts := <-receivedBlocksCh:
			// it means that we at the end. halt
			if bts == nil {
				zap.S().Infof("[%s] StateSync: CreateBulk: exit with null bytes")
				out := make([][]byte, len(blocks))
				copy(out, blocks)
				blocksBulk <- out
				return nil
			}
			blocks = append(blocks, bts)
			if l := len(blocks); l == blockCnt {
				out := make([][]byte, l)
				copy(out, blocks)
				blocksBulk <- out
				blocks = blocks[:0]
			}
		}
	}
}
