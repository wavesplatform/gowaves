package node

import (
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"time"
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
	//for {
	p, score, ok := a.peerManager.PeerWithHighestScore()
	if !ok || score.String() == "0" {
		// no peers, skip
		zap.S().Info("no peers, skip ", score.String())
		//time.Sleep(5 * time.Second)
		return errors.Errorf("no score found")
	}

	// TODO check if we have highest score

	//
	sigs, err := a.lastSignatures()
	if err != nil {
		zap.S().Error(err)
		return err
	}

	send := &proto.GetSignaturesMessage{
		Blocks: sigs.Signatures(),
	}
	//zap.S().Info("Sended signatures", send)

	p.SendMessage(send)

	messCh, unsubscribe := a.subscribe.Subscribe(p, &proto.SignaturesMessage{})

	//var mess *proto.SignaturesMessage

	select {
	case <-a.interrupt:
		return errors.Errorf("interrupt error")
	case <-time.After(15 * time.Second):
		// TODO handle timeout
		zap.S().Info("timeout waiting &proto.SignaturesMessage{}")
		return TimeoutErr
	case received := <-messCh:
		//a.subscribe.Unsubscribe(p, &proto.SignaturesMessage{})
		zap.S().Info("received signatures", received)
		unsubscribe()
		mess := received.(*proto.SignaturesMessage)
		applyBlock2(mess, sigs, p, a.subscribe, a.stateManager)
	}

	//?, ? := a.findMaxCommonBlock(mess.Signatures)

	//for _, i := range mess.Signatures {
	//}

	//if err != nil {
	//	if err == TimeoutErr {
	//		// TODO handle timeout
	//	}
	//}

	//ask.Subscribe(15*time.Second)
	//
	//a.subscribe.Clear(ask)
	//
	//if ask.Timeout() {
	//	// TODO handle timeout
	//}
	//
	//m := ask.Get().(*proto.SignaturesMessage{})

	//}
	return nil
}

func (a *StateSync) lastSignatures() (*BlockSignatures, error) {
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
	return NewBlockSignatures(signatures), nil
}

func (a *StateSync) Close() {
	close(a.interrupt)
}

func applyBlock(mess *proto.SignaturesMessage, blockSignatures *BlockSignatures, p peer.Peer, subscribe *Subscribe, stateManager state.State) {
	subscribeCh, unsubscribe := subscribe.Subscribe(p, &proto.BlockMessage{})
	defer unsubscribe()
	for _, sig := range mess.Signatures {
		if !blockSignatures.Exists(sig) {
			p.SendMessage(&proto.GetBlockMessage{BlockID: sig})

			// wait for block with expected signature
			timeout := time.After(30 * time.Second)
			for {
				select {
				case <-timeout:
					// TODO HANDLE timeout
					zap.S().Error("timeout getting block", sig)
					return

				case blockMessage := <-subscribeCh:
					bts := blockMessage.(*proto.BlockMessage).BlockBytes
					blockSignature, err := proto.BlockGetSignature(bts)
					if err != nil {
						zap.S().Error(err)
						continue
					}

					if blockSignature != sig {
						continue
					}

					err = stateManager.AddBlock(bts)
					if err != nil {
						zap.S().Error(err)
						// TODO handle error
					}
					break
				}
			}
		}
	}
}

func applyBlock2(receivedSignatures *proto.SignaturesMessage, blockSignatures *BlockSignatures, p peer.Peer, subscribe *Subscribe, stateManager state.State) {

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

		select {
		case <-timeout:
			// TODO HANDLE timeout
			zap.S().Error("timeout getting block", sigs[i])
			return

		case bts := <-ch:
			//bts := blockMessage.(*proto.BlockMessage).BlockBytes
			//blockSignature, err := proto.BlockGetSignature(bts)
			//if err != nil {
			//	zap.S().Error(err)
			//	continue
			//}

			//if blockSignature != sig {
			//	continue
			//}

			err := stateManager.AddBlock(bts)
			if err != nil {
				zap.S().Error(err)
				// TODO handle error
			}
			break
		}
	}

	////subscribeCh, unsubscribe := subscribe.Subscribe(p, &proto.BlockMessage{})
	////defer unsubscribe()
	//for _, sig := range receivedSignatures.Signatures {
	//	if !blockSignatures.Exists(sig) {
	//		p.SendMessage(&proto.GetBlockMessage{BlockID: sig})
	//
	//		// wait for block with expected signature
	//		timeout := time.After(30 * time.Second)
	//		for {
	//			select {
	//			case <-timeout:
	//				// TODO HANDLE timeout
	//				zap.S().Error("timeout getting block", sig)
	//				return
	//
	//			case blockMessage := <-subscribeCh:
	//				bts := blockMessage.(*proto.BlockMessage).BlockBytes
	//				blockSignature, err := proto.BlockGetSignature(bts)
	//				if err != nil {
	//					zap.S().Error(err)
	//					continue
	//				}
	//
	//				if blockSignature != sig {
	//					continue
	//				}
	//
	//				err = stateManager.AddBlock(bts)
	//				if err != nil {
	//					zap.S().Error(err)
	//					// TODO handle error
	//				}
	//				break
	//			}
	//		}
	//	}
	//}
}

func sendBulk(sigs []crypto.Signature, p peer.Peer) {
	bulkMessages := proto.BulkMessage{}
	for _, sig := range sigs {
		bulkMessages = append(bulkMessages, &proto.GetBlockMessage{BlockID: sig})
	}
	p.SendMessage(bulkMessages)
}
