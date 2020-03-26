package node

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/libs/channel"
	"github.com/wavesplatform/gowaves/pkg/libs/nullable"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type blockBytes struct {
	bytes      []byte
	isProtobuf bool
}

type expectedBlocks struct {
	curPosition     int
	blockToPosition map[proto.BlockID]int
	lst             []blockBytes
	notify          chan blockBytes
	mu              sync.Mutex
}

func newExpectedBlocks(ids []proto.BlockID, notify chan blockBytes) *expectedBlocks {
	blockToPosition := make(map[proto.BlockID]int, len(ids))

	for idx, value := range ids {
		blockToPosition[value] = idx
	}

	return &expectedBlocks{
		blockToPosition: blockToPosition,
		curPosition:     0,
		lst:             make([]blockBytes, len(ids)),
		notify:          notify,
	}
}

func (a *expectedBlocks) add(id proto.BlockID, block blockBytes) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	n, ok := a.blockToPosition[id]
	if !ok {
		return errors.Errorf("unexpected block id %s", id.String())
	}

	a.lst[n] = block

	for a.curPosition < len(a.lst) {
		if a.lst[a.curPosition].bytes == nil {
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

type ids struct {
	idSequence   []nullable.BlockID
	uniqBlockIDs map[proto.BlockID]blockBytes
	mu           sync.Mutex
}

func newIds() *ids {
	return &ids{
		idSequence:   nil,
		uniqBlockIDs: make(map[proto.BlockID]blockBytes),
		mu:           sync.Mutex{},
	}
}

func (a *ids) contains(id proto.BlockID) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.uniqBlockIDs[id]
	return ok
}

func (a *ids) setBytes(id proto.BlockID, b blockBytes) {
	a.mu.Lock()
	a.uniqBlockIDs[id] = b
	a.mu.Unlock()
}

func (a *ids) pop() (nullable.BlockID, blockBytes, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.idSequence) == 0 {
		return nullable.BlockID{}, blockBytes{}, false
	}
	firstId := a.idSequence[0]
	if firstId.Null() {
		a.idSequence = a.idSequence[1:]
		return firstId, blockBytes{}, true
	}
	bts := a.uniqBlockIDs[firstId.ID()]
	if bts.bytes != nil {
		delete(a.uniqBlockIDs, firstId.ID())
		a.idSequence = a.idSequence[1:]
		return firstId, bts, true
	}
	return nullable.BlockID{}, blockBytes{}, false
}

// true - added, false - not added
func (a *ids) add(id nullable.BlockID) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	// already contains
	if _, ok := a.uniqBlockIDs[id.ID()]; ok {
		return false
	}
	a.idSequence = append(a.idSequence, id)
	a.uniqBlockIDs[id.ID()] = blockBytes{}
	return true
}

type blockDownload struct {
	threads   chan int
	ids       *ids
	p         peer.Peer
	subscribe types.Subscribe
	out       channel.Channel
	closeCh   chan struct{}
	scheme    proto.Scheme
}

func newBlockDownloader(workersCount int, p peer.Peer, subscribe types.Subscribe, channel channel.Channel, scheme proto.Scheme) *blockDownload {
	return &blockDownload{
		threads:   make(chan int, workersCount),
		ids:       newIds(),
		p:         p,
		subscribe: subscribe,
		out:       channel,
		closeCh:   make(chan struct{}),
		scheme:    scheme,
	}
}

func (a *blockDownload) download(id nullable.BlockID) bool {
	r := a.ids.add(id)
	if r && !id.Null() {
		a.threads <- 1
		a.p.SendMessage(&proto.GetBlockMessage{BlockID: id.ID()})
	}
	return r
}

func (a *blockDownload) close() {
	close(a.closeCh)
}

func (a *blockDownload) subscrBlock(ctx context.Context, times int) (chan proto.Message, func(), error) {
	subscribeCh, unsubscribe, err := a.subscribe.Subscribe(a.p, &proto.BlockMessage{})
	if err != nil {
		if times == 0 {
			return subscribeCh, unsubscribe, err
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return a.subscrBlock(ctx, times-1)
		}
	}
	return subscribeCh, unsubscribe, nil
}

func (a *blockDownload) subscrPBBlock(ctx context.Context, times int) (chan proto.Message, func(), error) {
	subscribeCh, unsubscribe, err := a.subscribe.Subscribe(a.p, &proto.PBBlockMessage{})
	if err != nil {
		if times == 0 {
			return subscribeCh, unsubscribe, err
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return a.subscrPBBlock(ctx, times-1)
		}
	}
	return subscribeCh, unsubscribe, nil
}

func (a *blockDownload) run(ctx context.Context, wg *sync.WaitGroup) {
	defer zap.S().Debug("Exit blockDownload")
	defer a.out.Close()
	wg.Add(1)
	defer wg.Done()
	subscribeCh, unsubscribe, err := a.subscrBlock(ctx, 10)
	if err != nil {
		zap.S().Error(err)
		zap.S().Debug("Exit blockDownload, subscribe problem")
		return
	}
	defer unsubscribe()

	subscribeCh2, unsubscribe2, err := a.subscrPBBlock(ctx, 10)
	if err != nil {
		zap.S().Error(err)
		zap.S().Debug("Exit blockDownload, subscribe problem")
		return
	}
	defer unsubscribe2()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			a.out.Close()
			return
		case mess := <-subscribeCh:
			bb := mess.(*proto.BlockMessage).BlockBytes
			block := &proto.Block{}
			if err := block.UnmarshalBinary(bb, a.scheme); err != nil {
				continue
			}
			id := block.BlockID()
			// we are not waiting for this id
			if !a.ids.contains(id) {
				continue
			}
			a.ids.setBytes(id, blockBytes{bb, false})
			select {
			case <-a.threads:
			case <-ctx.Done():
				return
			}

			for {
				_, bts, ok := a.ids.pop()
				if ok {
					if !a.out.Send(bts) {
						zap.S().Debug("Exit blockDownload, !a.out.Send(bts)")
						return
					}
					if bts.bytes == nil {
						return
					}
					continue
				}
				break
			}
		case mess := <-subscribeCh2:
			bb := mess.(*proto.PBBlockMessage).PBBlockBytes
			block := &proto.Block{}
			if err := block.UnmarshalFromProtobuf(bb); err != nil {
				continue
			}
			id := block.BlockID()
			// we are not waiting for this id
			if !a.ids.contains(id) {
				continue
			}
			a.ids.setBytes(id, blockBytes{bb, true})
			select {
			case <-a.threads:
			case <-ctx.Done():
				return
			}

			for {
				_, bts, ok := a.ids.pop()
				if ok {
					if !a.out.Send(bts) {
						zap.S().Debug("Exit blockDownload, !a.out.Send(bts)")
						return
					}
					if bts.bytes == nil {
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
	Handshake() proto.Handshake
}

func PreloadBlockIds(ctx context.Context, out chan nullable.BlockID, p sendMessage, lastBlockIds *BlockIds, subscribe types.Subscribe, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()
	messCh, unsubscribe, err := subscribe.Subscribe(p, &proto.SignaturesMessage{})
	if err != nil {
		return err
	}
	defer unsubscribe()
	messCh2, unsubscribe2, err := subscribe.Subscribe(p, &proto.BlockIdsMessage{})
	if err != nil {
		return err
	}
	defer unsubscribe2()
	for {
		es := lastBlockIds.Ids()
		if len(es) == 0 {
			return nil
		}
		sendGetBlockIds(es, p)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(120 * time.Second):
			zap.S().Debugf("[%s] Optimistic Loader: timeout while waiting for new idnature", p.ID())
			return TimeoutErr
		case received := <-messCh:
			mess := received.(*proto.SignaturesMessage)
			var newIds []proto.BlockID
			for _, sig := range mess.Signatures {
				id := proto.NewBlockIDFromSignature(sig)
				if lastBlockIds.Exists(id) {
					continue
				}
				newIds = append(newIds, id)
				select {
				case out <- nullable.NewBlockID(id):
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// we are near end. Send
			if len(mess.Signatures) < 100 {
				select {
				case out <- nullable.NewNullBlockID():
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			lastBlockIds = NewBlockIds(newIds...).Revert()
			zap.S().Debugf("[%s] Optimistic loader: %d new ids received", p.ID(), len(lastBlockIds.Ids()))
		case received := <-messCh2:
			mess := received.(*proto.BlockIdsMessage)
			var newIds []proto.BlockID
			for _, id := range mess.Blocks {
				if lastBlockIds.Exists(id) {
					continue
				}
				newIds = append(newIds, id)
				select {
				case out <- nullable.NewBlockID(id):
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// we are near end. Send
			if len(mess.Blocks) < 100 {
				select {
				case out <- nullable.NewNullBlockID():
				case <-time.After(2 * time.Minute):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			lastBlockIds = NewBlockIds(newIds...).Revert()
			zap.S().Debugf("[%s] Optimistic loader: %d new ids received", p.ID(), len(lastBlockIds.Ids()))
		}
	}
}
