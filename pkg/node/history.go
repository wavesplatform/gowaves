package node

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type History struct {
	ctx  context.Context
	wait func() error

	networkCh <-chan peer.ProtoMessage

	scheme proto.Scheme
	st     state.State
}

func NewHistory(networkCh <-chan peer.ProtoMessage, scheme proto.Scheme, st state.State) *History {
	return &History{
		networkCh: networkCh,
		scheme:    scheme,
		st:        st,
	}
}

func (h *History) Run(ctx context.Context) {
	g, gc := errgroup.WithContext(ctx)
	h.ctx = gc
	h.wait = g.Wait
	g.Go(h.handleEvents)
}

func (h *History) Shutdown() {
	if err := h.wait(); err != nil {
		zap.S().Named(logging.HistoryNamespace).
			Warnf("Failed to properly shutdown history: %v", err)
	}
}

func (h *History) handleEvents() error {
	for {
		select {
		case <-h.ctx.Done():
			zap.S().Named(logging.HistoryNamespace).Info("History termination started")
			return nil
		case m, ok := <-h.networkCh:
			if err := h.handleNetworkMessages(m, ok); err != nil {
				return err
			}
		}
	}
}

func (h *History) handleNetworkMessages(m peer.ProtoMessage, ok bool) error {
	if !ok {
		zap.S().Named(logging.HistoryNamespace).Warn("History messages channel was closed by producer")
		return errors.New("history messages channel was closed")
	}
	switch msg := m.Message.(type) {
	case *proto.GetBlockMessage:
		h.handleGetBlockMessage(m.ID, msg.BlockID)
	case *proto.GetSignaturesMessage:
		h.handleGetSignaturesMessage(m.ID, msg)
	case *proto.GetBlockIdsMessage:
		h.getBlockIDs(m.ID, msg.Blocks, false)
	default:
		zap.S().Named(logging.HistoryNamespace).Errorf("Unexpected history message '%T'", m)
		return errors.Errorf("unexpected history message type '%T'", m)
	}
	return nil
}

func (h *History) handleGetBlockMessage(p peer.Peer, id proto.BlockID) {
	metricGetBlockMessage.Inc()

	block, err := h.st.Block(id)
	if err != nil {
		zap.S().Named(logging.HistoryNamespace).Warnf("Failed to retriev a block by ID '%s': %v", id.String(), err)
		return
	}
	bm, err := proto.MessageByBlock(block, h.scheme)
	if err != nil {
		zap.S().Named(logging.HistoryNamespace).Errorf("Failed to build Block message: %v", err)
		return
	}
	p.SendMessage(bm)
}

func (h *History) handleGetSignaturesMessage(p peer.Peer, msg *proto.GetSignaturesMessage) {
	ids := make([]proto.BlockID, len(msg.Signatures))
	for i, sig := range msg.Signatures {
		ids[i] = proto.NewBlockIDFromSignature(sig)
	}
	h.getBlockIDs(p, ids, true)
}

func (h *History) getBlockIDs(p peer.Peer, ids []proto.BlockID, asSignatures bool) {
	for _, id := range ids {
		if height, err := h.st.BlockIDToHeight(id); err == nil {
			h.sendNextBlockIDs(p, height, id, asSignatures)
			return
		}
	}
}

func (h *History) sendNextBlockIDs(p peer.Peer, height proto.Height, id proto.BlockID, asSignatures bool) {
	ids := make([]proto.BlockID, 1, blockIDsSequenceLength)
	ids[0] = id                                   // Put the common block ID as first in result
	for i := 1; i < blockIDsSequenceLength; i++ { // Add up to 100 more IDs
		b, err := h.st.HeaderByHeight(height + uint64(i))
		if err != nil {
			break
		}
		ids = append(ids, b.BlockID())
	}

	// There are block signatures to send in addition to requested one
	if len(ids) > 1 {
		if asSignatures {
			sigs := convertToSignatures(ids) // It could happen that only part of IDs can be converted to signatures
			if len(sigs) > 1 {
				p.SendMessage(&proto.SignaturesMessage{Signatures: sigs})
			}
			return
		}
		p.SendMessage(&proto.BlockIdsMessage{Blocks: ids})
	}
}
