package fsm

import (
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type currentScorer interface {
	// CurrentScore gets current blockchain score (at top height).
	CurrentScore() (*proto.Score, error)
}

type Actions interface {
	SendScore(currentScorer)
	SendBlock(block *proto.Block)
}

type ActionsImpl struct {
	logger   *slog.Logger
	services services.Services
}

func (a *ActionsImpl) SendScore(s currentScorer) {
	curScore, err := s.CurrentScore()
	if err != nil {
		a.logger.Error("Failed to get current score", logging.Error(err))
		return
	}
	var (
		msg = &proto.ScoreMessage{Score: curScore.Bytes()}
		cnt int
	)
	a.services.Peers.EachConnected(func(peer peer.Peer, score *proto.Score) {
		peer.SendMessage(msg)
		cnt++
	})
	a.logger.Debug("Network message sent to peers", logging.Type(msg), slog.Int("count", cnt),
		slog.Any("currentScore", curScore))
}

func (a *ActionsImpl) SendBlock(block *proto.Block) {
	bts, err := block.Marshaller().Marshal(a.services.Scheme)
	if err != nil {
		a.logger.Error("Failed to marshal block", slog.String("blockID", block.BlockID().String()),
			logging.Error(err))
		return
	}

	activated, err := a.services.State.IsActivated(int16(settings.BlockV5))
	if err != nil {
		a.logger.Error("Failed to get feature activation status", slog.Any("feature", settings.BlockV5),
			logging.Error(err))
		return
	}

	var (
		msg proto.Message
		cnt int
	)
	if activated {
		msg = &proto.PBBlockMessage{PBBlockBytes: bts}
	} else {
		msg = &proto.BlockMessage{BlockBytes: bts}
	}
	a.services.Peers.EachConnected(func(p peer.Peer, score *proto.Score) {
		p.SendMessage(msg)
		cnt++
	})
	a.logger.Debug("Network message sent to peers", logging.Type(msg), slog.Int("count", cnt),
		slog.Any("blockID", block.BlockID()))
}
