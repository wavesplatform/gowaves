package node

import (
	"context"
	"hash/maphash"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	maxShiftFromNow = 600000 // 10 minutes.
)

func MaybeEnableExtendedApi(state state.State, time types.Time) error {
	lastBlock := state.TopBlock()
	return maybeEnableExtendedApi(state, lastBlock, proto.NewTimestampFromTime(time.Now()))
}

type startProvidingExtendedApi interface {
	StartProvidingExtendedApi() error
}

func maybeEnableExtendedApi(state startProvidingExtendedApi, lastBlock *proto.Block, now proto.Timestamp) error {
	provideExtended := false
	if lastBlock.Timestamp > now {
		provideExtended = true
	} else if now-lastBlock.Timestamp < maxShiftFromNow {
		provideExtended = true
	}
	if provideExtended {
		if err := state.StartProvidingExtendedApi(); err != nil {
			return err
		}
	}
	return nil
}

type safeMap[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]V
}

func newSafeMap[K comparable, V any]() *safeMap[K, V] {
	return &safeMap[K, V]{
		m: make(map[K]V),
	}
}

func (s *safeMap[K, V]) SetIfNew(key K, value V) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[key]; exists {
		return false
	}
	s.m[key] = value
	return true
}

func (s *safeMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

type protoMessageWrapper struct {
	protoMsg      peer.ProtoMessage
	payloadDigest uint64
}

func messagesSplitter(
	ctx context.Context,
	origMessageCh <-chan peer.ProtoMessage,
	noTxChan, txChan chan<- protoMessageWrapper,
	sm *safeMap[uint64, struct{}],
) {
	seed := maphash.MakeSeed()
	for {
		var (
			msg           peer.ProtoMessage
			payloadDigest uint64
		)
		outCh := noTxChan
		select {
		case <-ctx.Done():
			return
		case msg = <-origMessageCh:
			switch protoMsg := msg.Message.(type) {
			case *proto.TransactionMessage:
				payloadDigest = maphash.Bytes(seed, protoMsg.Transaction)
				outCh = txChan
			case *proto.PBTransactionMessage:
				payloadDigest = maphash.Bytes(seed, protoMsg.Transaction)
				outCh = txChan
			}
		}
		needToWrite := payloadDigest == 0 || sm.SetIfNew(payloadDigest, struct{}{})
		if !needToWrite {
			continue // skip a message if payload digest already exists in the map
		}
		select {
		case <-ctx.Done():
			return
		case outCh <- protoMessageWrapper{msg, payloadDigest}:
		}
	}
}

func runMessagesMerger(
	ctx context.Context, wg *sync.WaitGroup,
	noTxChan, txChan <-chan protoMessageWrapper,
	sm *safeMap[uint64, struct{}],
) <-chan peer.ProtoMessage {
	wg.Add(1)
	outMessageCh := make(chan peer.ProtoMessage) // intentionally unbuffered channel because of map usage
	go func() {
		defer wg.Done()
		for {
			var wMsg protoMessageWrapper
			select {
			case <-ctx.Done():
				return
			case wMsg = <-noTxChan:
			case wMsg = <-txChan:
			}
			select {
			case <-ctx.Done():
				return
			case outMessageCh <- wMsg.protoMsg:
				if wMsg.payloadDigest != 0 {
					sm.Delete(wMsg.payloadDigest) // remove payload digest from the map after sending the message
				}
			}
		}
	}()
	return outMessageCh
}

type chanLenProvider[T any] <-chan T

func (c chanLenProvider[T]) Len() int { return len(c) }

type aggregatedLenProvider []lenProvider

func (a aggregatedLenProvider) Len() int {
	l := 0
	for _, p := range a {
		l += p.Len()
	}
	return l
}

func wrapParentProtoMessagesChan(
	ctx context.Context, origMessageCh <-chan peer.ProtoMessage,
) (<-chan peer.ProtoMessage, lenProvider, interface{ Wait() }) {
	const noTxChanSize = 1000
	wg := &sync.WaitGroup{}
	sm := newSafeMap[uint64, struct{}]()

	noTxChan := make(chan protoMessageWrapper, noTxChanSize)
	txChan := make(chan protoMessageWrapper, max(cap(origMessageCh)-noTxChanSize, noTxChanSize))

	wg.Add(1)
	go func() { // run messages splitter with filtering
		defer wg.Done()
		messagesSplitter(ctx, origMessageCh, noTxChan, txChan, sm)
	}()

	out := runMessagesMerger(ctx, wg, noTxChan, txChan, sm)

	lp := aggregatedLenProvider{
		chanLenProvider[peer.ProtoMessage](origMessageCh),
		chanLenProvider[protoMessageWrapper](noTxChan),
		chanLenProvider[protoMessageWrapper](txChan),
		chanLenProvider[peer.ProtoMessage](out),
	}
	return out, lp, wg
}
