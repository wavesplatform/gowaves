package node

import (
	"fmt"
	"math/big"
	"time"

	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"go.uber.org/zap"
)

type StateSync struct {
	peerManager  PeerManager
	stateManager state.State
	subscribe    *Subscribe
	interrupt    chan struct{}
}

func NewStateSync(stateManager state.State, peerManager PeerManager, subscribe *Subscribe) *StateSync {
	return &StateSync{
		peerManager:  peerManager,
		stateManager: stateManager,
		subscribe:    subscribe,
		interrupt:    make(chan struct{}),
	}
}

var TimeoutErr = errors.Errorf("Timeout")

func (a *StateSync) Sync() error {

	p, err := a.getPeerWithHighestScore()
	if err != nil {
		return err
	}

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

		zap.S().Info("received signatures", received)
		mess := received.(*proto.SignaturesMessage)
		applyBlock2(mess, sigs, p, a.subscribe, a.stateManager, a.peerManager)
	}

	return nil
}

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

func applyBlock2(receivedSignatures *proto.SignaturesMessage, blockSignatures *Signatures, p Peer, subscribe *Subscribe, stateManager state.State, peerManager PeerManager) {

	var sigs []crypto.Signature
	for _, sig := range receivedSignatures.Signatures {
		if !blockSignatures.Exists(sig) {
			sigs = append(sigs, sig)
		}
	}

	ch := make(chan blockBytes, len(sigs))

	go func() {
		subscribeCh, unsubscribe := subscribe.Subscribe(p, &proto.BlockMessage{})
		defer unsubscribe()

		e := newExpectedBlocks(sigs, ch)
		sendBulk(sigs, p)

		// expect that all messages will income in 2 minutes
		timeout := time.After(120 * time.Second)

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
			}
		}

	}()

	for i := 0; i < len(sigs); i++ {

		timeout := time.After(30 * time.Second)

		// ask block again after 5 second
		cancel := cancellable.After(5*time.Second, func() {
			p.SendMessage(&proto.GetBlockMessage{BlockID: sigs[i]})
		})

		zap.S().Info("waiting for sig  ", sigs[i])

		select {
		case <-timeout:
			// TODO HANDLE timeout
			zap.S().Error("timeout getting block", sigs[i])
			cancel()
			return

		case bts := <-ch:
			cancel()
			err := stateManager.AddBlock(bts)
			if err != nil {

				fmt.Println(bts)

				zap.S().Error(err)
				continue
			}

			cur, err := stateManager.CurrentScore()
			if err == nil {
				peerManager.EachConnected(func(peer Peer, i *big.Int) {
					peer.SendMessage(&proto.ScoreMessage{
						Score: cur.Bytes(),
					})
				})
			}
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
