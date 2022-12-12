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
		zap.S().Errorf("Failed to get current score: %v", err)
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
	zap.S().Debugf("Network message '%T' sent to %d peers: currentScore=%s", msg, cnt, curScore)
}

func (a *ActionsImpl) SendBlock(block *proto.Block) {
	bts, err := block.Marshaller().Marshal(a.services.Scheme)
	if err != nil {
		zap.S().Errorf("Failed to marshal block with ID %q: %v", block.BlockID().String(), err)
		return
	}

	activated, err := a.services.State.IsActivated(int16(settings.BlockV5))
	if err != nil {
		zap.S().Errorf("Failed to get feature (%d) activation status: %v", settings.BlockV5, err)
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
	zap.S().Debugf("Network message '%T' sent to %d peers: blockID='%s'", msg, cnt, block.BlockID())
}
