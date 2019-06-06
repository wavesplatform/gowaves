package api

import "github.com/wavesplatform/gowaves/pkg/proto"

type Score struct {
	Score string `json:"score"`
}

func (a *App) BlocksScoreAt(at proto.Height) (*Score, error) {
	score, err := a.node.State().ScoreAtHeight(at)
	if err != nil {
		return nil, err
	}
	return &Score{Score: score.String()}, nil
}
