package api

import "github.com/wavesplatform/gowaves/pkg/proto"

type Score struct {
	Score string `json:"score"`
}

func (a *App) BlocksScoreAt(at proto.Height) (*Score, error) {
	score, err := a.state.ScoreAtHeight(at)
	if err != nil {
		return nil, err
	}
	return &Score{Score: score.String()}, nil
}

func (a *App) BlocksLast() (*proto.Block, error) {
	h, err := a.state.Height()
	if err != nil {
		return nil, &InternalError{err}
	}

	block, err := a.state.BlockByHeight(h)
	if err != nil {
		return nil, &InternalError{err}
	}
	block.Height = h
	return block, nil
}

func (a *App) BlocksFirst() (*proto.Block, error) {
	block, err := a.state.BlockByHeight(1)
	if err != nil {
		return nil, &InternalError{err}
	}
	block.Height = 1
	return block, nil
}
