package node

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/libs/nullable"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type interrupt = chan struct{}

type StateSync struct {
	peerManager  peer_manager.PeerManager
	stateManager state.State
	subscribe    types.Subscribe
	interrupt    interrupt
	scheduler    types.Scheduler
	blockApplier types.BlockApplier
	interrupter  types.MinerInterrupter
	syncCh       chan struct{}
	services     services.Services
	scoreSender  types.ScoreSender

	// need to enable or disable sync
	mu      sync.Mutex
	enabled bool
}

func NewStateSync(services services.Services, interrupter types.MinerInterrupter, scoreSender types.ScoreSender) *StateSync {
	return &StateSync{
		peerManager:  services.Peers,
		stateManager: services.State,
		subscribe:    services.Subscribe,
		interrupt:    make(chan struct{}, 1),
		scheduler:    services.Scheduler,
		blockApplier: services.BlockApplier,
		interrupter:  interrupter,
		syncCh:       make(chan struct{}, 20),
		services:     services,
		enabled:      true,
		scoreSender:  scoreSender,
	}
}

var TimeoutErr = errors.New("timeout")
var NothingToRequestErr = errors.New("nothing ot request")

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
		zap.S().Info("[%s] StateSync: Ended with code %q", err)
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

	go func() {
		errCh <- PreloadSignatures(ctx, incoming, p, sigs, a.subscribe, &wg)
	}()
	go func() {
		errCh <- a.downloadBlocks(ctx, incoming, p, &wg)
	}()

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
	defer a.services.Scheduler.Reschedule()
	wg.Add(1)
	defer wg.Done()

	errCh := make(chan error, 3)
	receivedBlocksCh := make(chan blockBytes, 128)

	downloader := newBlockDownloader(128, p, a.subscribe, receivedBlocksCh)
	go downloader.run(ctx, wg)

	const blockCnt = 50

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(120 * time.Second):
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
	}()

	blocksBulk := make(chan [][]byte, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		blocks := make([][]byte, 0, blockCnt)
		for {
			score, err := a.services.Peers.Score(p)
			if err != nil {
				errCh <- err
				return
			}
			curScore, err := a.services.State.CurrentScore()
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
				// it means that we at the end. halt
				if bts == nil {
					zap.S().Infof("[%s] StateSync: CreateBulk: exit with null bytes")
					out := make([][]byte, len(blocks))
					copy(out, blocks)
					blocksBulk <- out
					return
				}

				blocks = append(blocks, bts)
				if l := len(blocks); uint64(l) == blockCnt {
					out := make([][]byte, l)
					copy(out, blocks)
					blocksBulk <- out
					blocks = blocks[:0]
				}
			}
		}
	}()

	// block applier
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case blocks := <-blocksBulk:
				zap.S().Debugf("[%s] BlockDownloader: Applying: received %d blocks", p.ID(), len(blocks))
				if len(blocks) == 0 {
					errCh <- nil
					return
				}
				a.interrupter.Interrupt()
				err := applyBlocks(a.services, blocks, p)
				if err != nil {
					errCh <- err
					return
				}
				go a.services.BlockAddedNotifier.Handle()
				a.scoreSender.NonPriority()

				// received less than expected, it means successful exit
				if len(blocks) < blockCnt {
					errCh <- nil
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

func applyBlocks(services services.Services, blocks [][]byte, p Peer) error {
	locked := services.State.Mutex().Lock()
	defer locked.Unlock()
	h, err := services.State.Height()
	if err != nil {
		return err
	}
	id, err := services.State.HeightToBlockID(h)
	if err != nil {
		return err
	}
	parent, err := proto.BlockGetParent(blocks[0])
	if err != nil {
		return err
	}
	sig, err := proto.BlockGetSignature(blocks[0])
	if err != nil {
		return err
	}
	rollback := false
	if !bytes.Equal(id[:], parent[:]) {
		err := services.State.RollbackTo(parent)
		if err != nil {
			return err
		}
		rollback = true
	}
	size := 0
	groupIndex := 0
	for i, block := range blocks {
		blocksNumer := i + 1
		size += len(block)
		if (size < importer.MaxTotalBatchSizeForNetworkSync) && (blocksNumer != len(blocks)) {
			continue
		}
		blocksToApply := blocks[groupIndex:blocksNumer]
		groupIndex = blocksNumer
		if err := services.State.AddNewBlocks(blocksToApply); err != nil {
			zap.S().Debugf("[%s] BlockDownloader: error on adding new blocks: %q, sig: %s, parent sig %s, rollback: %v", p.ID(), err, sig, parent, rollback)
			return err
		}
		if err := MaybeEnableExtendedApi(services.State); err != nil {
			panic(fmt.Sprintf("[%s] BlockDownloader: MaybeEnableExtendedApi(): %v. Failed to persist address transactions for API after successfully applying valid blocks.", p.ID(), err))
		}
		size = 0
	}
	return nil
}
