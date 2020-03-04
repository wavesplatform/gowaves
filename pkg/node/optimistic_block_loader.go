package node

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/channel"
	"github.com/wavesplatform/gowaves/pkg/libs/nullable"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type blockBytes = []byte

type expectedBlocks struct {
	curPosition     int
	blockToPosition map[crypto.Signature]int
	lst             []blockBytes
	notify          chan blockBytes
	mu              sync.Mutex
}

func newExpectedBlocks(signatures []crypto.Signature, notify chan blockBytes) *expectedBlocks {
	blockToPosition := make(map[crypto.Signature]int, len(signatures))

	for idx, value := range signatures {
		blockToPosition[value] = idx
	}

	return &expectedBlocks{
		blockToPosition: blockToPosition,
		curPosition:     0,
		lst:             make([]blockBytes, len(signatures)),
		notify:          notify,
	}
}

func (a *expectedBlocks) add(block blockBytes) error {
	s, err := proto.BlockGetSignature(block)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	n, ok := a.blockToPosition[s]
	if !ok {
		return errors.Errorf("unexpected block sig %s", s)
	}

	a.lst[n] = block

	for a.curPosition < len(a.lst) {
		if a.lst[a.curPosition] == nil {
			break
		}
		a.notify <- a.lst[a.curPosition]
		a.curPosition += 1
	}

	return nil
}

func (a *expectedBlocks) hasNext() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.curPosition < len(a.lst)
}

type sigs struct {
	sigSequence    []nullable.Signature
	uniqSignatures map[crypto.Signature]blockBytes
	mu             sync.Mutex
}

func newSigs() *sigs {
	return &sigs{
		sigSequence:    nil,
		uniqSignatures: make(map[crypto.Signature]blockBytes),
		mu:             sync.Mutex{},
	}
}

func (a *sigs) contains(sig crypto.Signature) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.uniqSignatures[sig]
	return ok
}

func (a *sigs) setBytes(sig crypto.Signature, b blockBytes) {
	a.mu.Lock()
	a.uniqSignatures[sig] = b
	a.mu.Unlock()
}

func (a *sigs) pop() (nullable.Signature, blockBytes, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.sigSequence) == 0 {
		return nullable.Signature{}, nil, false
	}
	firstSig := a.sigSequence[0]
	if firstSig.Null() {
		a.sigSequence = a.sigSequence[1:]
		return firstSig, nil, true
	}
	bts := a.uniqSignatures[firstSig.Sig()]
	if bts != nil {
		delete(a.uniqSignatures, firstSig.Sig())
		a.sigSequence = a.sigSequence[1:]
		return firstSig, bts, true
	}
	return nullable.Signature{}, nil, false
}

// true - added, false - not added
func (a *sigs) add(sig nullable.Signature) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	// already contains
	if _, ok := a.uniqSignatures[sig.Sig()]; ok {
		return false
	}
	a.sigSequence = append(a.sigSequence, sig)
	a.uniqSignatures[sig.Sig()] = nil
	return true
}

type blockDownload struct {
	threads   chan int
	sigs      *sigs
	p         peer.Peer
	subscribe types.Subscribe
	out       channel.Channel
	closeCh   chan struct{}
}

func newBlockDownloader(workersCount int, p peer.Peer, subscribe types.Subscribe, channel channel.Channel) *blockDownload {
	return &blockDownload{
		threads:   make(chan int, workersCount),
		sigs:      newSigs(),
		p:         p,
		subscribe: subscribe,
		out:       channel,
		closeCh:   make(chan struct{}),
	}
}

func (a *blockDownload) download(sig nullable.Signature) bool {
	r := a.sigs.add(sig)
	if r && !sig.Null() {
		a.threads <- 1
		a.p.SendMessage(&proto.GetBlockMessage{BlockID: sig.Sig()})
	}
	return r
}

func (a *blockDownload) close() {
	close(a.closeCh)
}

func (a *blockDownload) subscr(ctx context.Context, times int) (chan proto.Message, func(), error) {
	subscribeCh, unsubscribe, err := a.subscribe.Subscribe(a.p, &proto.BlockMessage{})
	if err != nil {
		if times == 0 {
			return subscribeCh, unsubscribe, err
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return a.subscr(ctx, times-1)
		}
	}
	return subscribeCh, unsubscribe, nil
}

func (a *blockDownload) run(ctx context.Context, wg *sync.WaitGroup) {
	defer zap.S().Debug("Exit blockDownload")
	defer a.out.Close()
	wg.Add(1)
	defer wg.Done()
	subscribeCh, unsubscribe, err := a.subscr(ctx, 10)
	if err != nil {
		zap.S().Error(err)
		zap.S().Debug("Exit blockDownload, subscribe problem")
		return
	}
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			a.out.Close()
			return
		case mess := <-subscribeCh:
			bb := mess.(*proto.BlockMessage).BlockBytes
			sig, err := proto.BlockGetSignature(bb)
			if err != nil {
				continue
			}
			// we are not waiting for this sig
			if !a.sigs.contains(sig) {
				continue
			}
			a.sigs.setBytes(sig, bb)
			select {
			case <-a.threads:
			case <-ctx.Done():
				return
			}

			for {
				_, bts, ok := a.sigs.pop()
				if ok {
					if !a.out.Send(bts) {
						zap.S().Debug("Exit blockDownload, !a.out.Send(bts)")
						return
					}
					if bts == nil {
						return
					}
					continue
				}
				break
			}
		}
	}
}

type sendMessage interface {
	types.ID
	SendMessage(proto.Message)
}

func PreloadSignatures(ctx context.Context, out chan nullable.Signature, p sendMessage, lastSignatures *Signatures, subscribe types.Subscribe, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()
	messCh, unsubscribe, err := subscribe.Subscribe(p, &proto.SignaturesMessage{})
	if err != nil {
		return err
	}
	defer unsubscribe()
	for {
		es := lastSignatures.Signatures()
		if len(es) == 0 {
			return nil
		}
		send := &proto.GetSignaturesMessage{
			Blocks: es,
		}
		p.SendMessage(send)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(120 * time.Second):
			zap.S().Debugf("[%s] Optimistic Loader: timeout while waiting for new signature", p.ID())
			return TimeoutErr
		case received := <-messCh:
			mess := received.(*proto.SignaturesMessage)
			var newSigs []crypto.Signature
			for _, sig := range mess.Signatures {
				if lastSignatures.Exists(sig) {
					continue
				}
				newSigs = append(newSigs, sig)
				select {
				case out <- nullable.NewSignature(sig):
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// we are near end. Send
			if len(mess.Signatures) < 100 {
				select {
				case out <- nullable.NewNullSignature():
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			lastSignatures = NewSignatures(newSigs...).Revert()
			zap.S().Debugf("[%s] Optimistic loader: %d new signatures received", p.ID(), len(lastSignatures.Signatures()))
		}
	}
}
