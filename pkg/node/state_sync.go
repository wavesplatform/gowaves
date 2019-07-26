package node

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"go.uber.org/zap"
)

type StateSync struct {
	peerManager  peer_manager.PeerManager
	stateManager state.State
	subscribe    *Subscribe
	interrupt    chan struct{}
	scheduler    types.Scheduler
	blockApplier *BlockApplier
	interrupter  types.MinerInterrupter
	syncCh       chan struct{}
}

func NewStateSync(stateManager state.State, peerManager peer_manager.PeerManager, subscribe *Subscribe, scheduler types.Scheduler, interrupter types.MinerInterrupter) *StateSync {
	return &StateSync{
		peerManager:  peerManager,
		stateManager: stateManager,
		subscribe:    subscribe,
		interrupt:    make(chan struct{}),
		scheduler:    scheduler,
		blockApplier: NewBlockApplier(stateManager, peerManager, scheduler),
		interrupter:  interrupter,
		syncCh:       make(chan struct{}, 20),
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
		_ = a.sync(p)
	}
}

func (a *StateSync) sync(p Peer) error {
	messCh, unsubscribe := a.subscribe.Subscribe(p, &proto.SignaturesMessage{})
	defer unsubscribe()

	sigs, err := a.askSignatures(p)
	if err != nil {
		return err
	}

	select {
	case <-a.interrupt:
		return errors.Errorf("interrupt error")
	case <-time.After(15 * time.Second):
		// TODO handle timeout
		zap.S().Info("timeout waiting &proto.SignaturesMessage{}")
		return TimeoutErr
	case received := <-messCh:
		mess := received.(*proto.SignaturesMessage)
		downloadSignatures(mess, sigs, p, a.subscribe, a.blockApplier, a.interrupter, a.scheduler)
	}

	return nil
}

// Send GetSignaturesMessage to peer with 100 last signatures
func (a *StateSync) askSignatures(p Peer) (*Signatures, error) {
	sigs, err := a.lastSignatures()
	if err != nil {
		zap.S().Error(err)
		return nil, err
	}

	send := &proto.GetSignaturesMessage{
		Blocks: sigs.Signatures(),
	}

	p.SendMessage(send)
	return sigs, nil
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

// get last 100 signatures from db, from highest to lowest
func (a *StateSync) lastSignatures() (*Signatures, error) {
	var signatures []crypto.Signature

	// getting signatures
	height, err := a.stateManager.Height()
	if err != nil {
		zap.S().Error(err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		select {
		case <-a.interrupt:
			return nil, errors.Errorf("interrupt")
		default:
		}

		sig, err := a.stateManager.HeightToBlockID(height)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures), nil
}

func (a *StateSync) Close() {
	close(a.interrupt)
}

func downloadSignatures(
	receivedSignatures *proto.SignaturesMessage,
	blockSignatures *Signatures,
	p Peer,
	subscribe *Subscribe,
	applier *BlockApplier,
	interrupt types.MinerInterrupter,
	scheduler types.Scheduler) {

	defer scheduler.Reschedule()
	var sigs []crypto.Signature
	for _, sig := range receivedSignatures.Signatures {
		if !blockSignatures.Exists(sig) {
			sigs = append(sigs, sig)
		}
	}

	ch := make(chan blockBytes, len(sigs))

	ctx, cancel := context.WithCancel(context.Background())
	subscribeCh, unsubscribe := subscribe.Subscribe(p, &proto.BlockMessage{})
	defer unsubscribe()
	defer cancel()

	go func() {
		e := newExpectedBlocks(sigs, ch)
		sendBulk(sigs, p)

		// expect that all messages will income in 2 minutes
		timeout := time.After(120 * time.Second)

		// wait until we receive all signatures
		for e.hasNext() {
			select {
			case <-timeout:
				return
			case blockMessage := <-subscribeCh:
				bts := blockMessage.(*proto.BlockMessage).BlockBytes
				err := e.add(bts)
				if err != nil {
					zap.S().Warn(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for i := 0; i < len(sigs); i++ {

		timeout := time.After(30 * time.Second)

		// ask block again after 5 second
		cancel := cancellable.After(5*time.Second, func() {
			p.SendMessage(&proto.GetBlockMessage{BlockID: sigs[i]})
		})

		select {
		case <-timeout:
			// TODO HANDLE timeout, maybe block peer ot other
			zap.S().Error("timeout getting block", sigs[i])
			cancel()
			return

		case bts := <-ch:
			cancel()

			interrupt.Interrupt()
			err := applier.ApplyBytes(bts)
			if err != nil {
				zap.S().Error(err)
				continue
			}
			scheduler.Reschedule()
		}
	}
}

func sendBulk(sigs []crypto.Signature, p Peer) {
	bulkMessages := proto.BulkMessage{}
	for _, sig := range sigs {
		bulkMessages = append(bulkMessages, &proto.GetBlockMessage{BlockID: sig})
	}
	p.SendMessage(bulkMessages)
}
