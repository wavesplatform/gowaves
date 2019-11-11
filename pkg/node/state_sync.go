package node

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
		interrupt:    make(chan struct{}, 1),
		scheduler:    services.Scheduler,
		blockApplier: services.BlockApplier,
		interrupter:  interrupter,
		syncCh:       make(chan struct{}, 20),
		services:     services,
	}
}

var TimeoutErr = errors.New("timeout")
var NothingToRequestErr = errors.New("nothing ot request")

func (a *StateSync) Sync() {
	select {
	case a.syncCh <- struct{}{}:
		return
	default:
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

	n := 0
	for err := range errCh {
		switch err {
		case TimeoutErr:
			a.peerManager.Suspend(p)
			cancel()
			go func() {
				<-time.After(2 * time.Second)
				a.Sync()
			}()
			return nil
		case NothingToRequestErr:
			cancel()
		default:
			if err != nil {
				zap.S().Errorf("[%s] StateSync: Error: %v", p.ID(), err)
			}
		}
		n++
		if n == 2 {
			break
		}
	}
	cancel()
	return nil
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

func downloadBlocks(ctx context.Context, signaturesCh chan crypto.Signature, p Peer, subscribe *Subscribe, services services.Services, interrupt types.MinerInterrupter) error {
	defer services.Scheduler.Reschedule()

	errCh := make(chan error, 3)

	receivedBlocksCh := make(chan blockBytes, 128)

	downloader := newBlockDownloader(128, p, subscribe, receivedBlocksCh)
	go downloader.run(ctx)
	requested := 0
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				// TODO handle timeout
				zap.S().Infof("[%s] StateSync: timeout waiting for SignaturesMessage", p.ID())
				errCh <- TimeoutErr
				return
			case sig := <-signaturesCh:
				if downloader.download(sig) {
					requested++
				}
			}
		}
	}()

	blocksBulk := make(chan [][]byte, 1)

	go func() {
		const blockCnt = 50
		blocks := make([][]byte, 0, blockCnt)
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
				blocks = append(blocks, bts)
				size := blockCnt
				if requested < size {
					size = requested
				}
				if l := len(blocks); l == size {
					out := make([][]byte, l)
					copy(out, blocks)
					blocksBulk <- out
					blocks = blocks[:0]
					requested = requested - size
				}
			}
		}
	}()

	scoreUpdated := make(chan struct{}, 1)

	// block applier
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case blocks := <-blocksBulk:
				if len(blocks) == 0 {
					return
				}
				interrupt.Interrupt()
				locked := services.State.Mutex().Lock()
				h, err := services.State.Height()
				if err != nil {
					errCh <- err
					return
				}
				id, err := services.State.HeightToBlockID(h)
				if err != nil {
					errCh <- err
					return
				}
				parent, err := proto.BlockGetParent(blocks[0])
				if err != nil {
					errCh <- err
					return
				}
				if !bytes.Equal(id[:], parent[:]) {
					err := services.State.RollbackTo(parent)
					if err != nil {
						errCh <- err
						return
					}
				}
				err = services.State.AddNewBlocks(blocks)
				locked.Unlock()
				if err != nil {
					errCh <- err
					return
				}
				go services.BlockAddedNotifier.Handle()
				select {
				case scoreUpdated <- struct{}{}:
				default:
				}
			}
		}
	}()

	// send score to nodes
	go func() {
		tick := time.NewTicker(10 * time.Second)
		update := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-scoreUpdated:
				update = true
			case <-tick.C:
				if update {
					curScore, err := services.State.CurrentScore()
					if err != nil {
						zap.S().Info(err)
						continue
					}
					services.Peers.EachConnected(func(peer Peer, score *proto.Score) {
						peer.SendMessage(&proto.ScoreMessage{Score: curScore.Bytes()})
					})
				}
				update = false
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
