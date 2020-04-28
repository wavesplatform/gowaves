package state_fsm

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"go.uber.org/zap"
)

type currentScorer interface {
	// Get current blockchain score (at top height).
	CurrentScore() (*proto.Score, error)
}

type Actions interface {
	SendScore(currentScorer)
	SendBlock(block *proto.Block)
}

type ActionsImpl struct {
	services services.Services
}

func (a *ActionsImpl) SendScore(s currentScorer) {
	curScore, err := s.CurrentScore()
	if err != nil {
		zap.S().Error(err)
		return
	}
	bts := curScore.Bytes()
	a.services.Peers.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(&proto.ScoreMessage{Score: bts})
	})
}

func (a *ActionsImpl) SendBlock(block *proto.Block) {
	bts, err := block.Marshaller().Marshal(a.services.Scheme)
	if err != nil {
		zap.S().Error(err)
		return
	}

	activated, err := a.services.State.IsActivated(int16(settings.BlockV5))
	if err != nil {
		zap.S().Error(err)
		return
	}

	if activated {
		a.services.Peers.EachConnected(func(p peer.Peer, score *proto.Score) {
			p.SendMessage(&proto.PBBlockMessage{PBBlockBytes: bts})
		})
	} else {
		a.services.Peers.EachConnected(func(p peer.Peer, score *proto.Score) {
			p.SendMessage(&proto.BlockMessage{BlockBytes: bts})
		})
	}
}
