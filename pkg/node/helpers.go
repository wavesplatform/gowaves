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
	protoMsg  peer.ProtoMessage
	messageID uint64
}

// messageIDFilterFunc is a function type that takes a proto.Message and returns a uint64
// representing the deterministic message ID. This is used to filter messages based on
// their message ID: it can be calculated from the message payload, or from the whole message itself.
//
// If the returned message ID is 0, the message bypasses filtering.
type messageIDFilterFunc func(message proto.Message) uint64

type txMessagePayloadIDFilter struct {
	seed maphash.Seed
}

func (f txMessagePayloadIDFilter) Filter(message proto.Message) uint64 {
	switch msg := message.(type) {
	case *proto.TransactionMessage:
		return maphash.Bytes(f.seed, msg.Transaction)
	case *proto.PBTransactionMessage:
		return maphash.Bytes(f.seed, msg.Transaction)
	default:
		return 0 // non-transaction messages bypass filtering
	}
}

func messagesSplitter(
	ctx context.Context,
	origMessageCh <-chan peer.ProtoMessage,
	noFilteredChan, filteredChan chan<- protoMessageWrapper,
	sm *safeMap[uint64, struct{}],
	messageIDFilter messageIDFilterFunc,
) {
	for {
		var (
			msg       peer.ProtoMessage
			messageID uint64
		)
		select {
		case <-ctx.Done():
			return
		case msg = <-origMessageCh:
			messageID = messageIDFilter(msg.Message)
		}
		bypassesFiltering := messageID == 0 // if messageID is 0, it means the message no need to be filtered.
		needToWrite := bypassesFiltering || sm.SetIfNew(messageID, struct{}{})
		if !needToWrite {
			continue // skip a message if message ID already exists in the map
		}
		outCh := noFilteredChan
		if !bypassesFiltering {
			outCh = filteredChan
		}
		select {
		case <-ctx.Done():
			return
		case outCh <- protoMessageWrapper{msg, messageID}:
		}
	}
}

func runMessagesMerger(
	ctx context.Context, wg *sync.WaitGroup,
	noFilteredChan, filteredChan <-chan protoMessageWrapper,
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
			case wMsg = <-noFilteredChan:
			case wMsg = <-filteredChan:
			}
			select {
			case <-ctx.Done():
				return
			case outMessageCh <- wMsg.protoMsg:
				if wMsg.messageID != 0 {
					sm.Delete(wMsg.messageID) // remove message ID from the map after sending the message
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

type waiter interface {
	Wait()
}

func deduplicateProtoMessages(
	ctx context.Context, origMessageCh <-chan peer.ProtoMessage,
	messageIDFilter messageIDFilterFunc,
) (<-chan peer.ProtoMessage, lenProvider, waiter) {
	const noFilteredChanSize = 1000

	wg := &sync.WaitGroup{}
	sm := newSafeMap[uint64, struct{}]()

	noFilteredChan := make(chan protoMessageWrapper, noFilteredChanSize)
	filteredChan := make(chan protoMessageWrapper, max(cap(origMessageCh)-noFilteredChanSize, noFilteredChanSize))

	wg.Add(1)
	go func() { // run messages splitter with filtering
		defer wg.Done()
		messagesSplitter(ctx, origMessageCh, noFilteredChan, filteredChan, sm, messageIDFilter)
	}()

	out := runMessagesMerger(ctx, wg, noFilteredChan, filteredChan, sm)

	lp := aggregatedLenProvider{
		chanLenProvider[peer.ProtoMessage](origMessageCh),
		chanLenProvider[protoMessageWrapper](noFilteredChan),
		chanLenProvider[protoMessageWrapper](filteredChan),
		chanLenProvider[peer.ProtoMessage](out),
	}
	return out, lp, wg
}

func deduplicateProtoTxMessages(
	ctx context.Context, origMessageCh <-chan peer.ProtoMessage,
) (<-chan peer.ProtoMessage, lenProvider, waiter) {
	payloadIDFilter := txMessagePayloadIDFilter{seed: maphash.MakeSeed()}
	return deduplicateProtoMessages(ctx, origMessageCh, payloadIDFilter.Filter)
}
