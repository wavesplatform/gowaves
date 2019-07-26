package api

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

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

type Generators []Generator
type Generator struct {
	Height proto.Height     `json:"height"`
	PubKey crypto.PublicKey `json:"pub_key"`
}

func (a *App) BlocksGenerators() (Generators, error) {
	curHeight, err := a.state.Height()
	if err != nil {
		return nil, &InternalError{err}
	}

	out := Generators{}
	for i := proto.Height(1); i < curHeight; i++ {
		block, err := a.state.BlockByHeight(i)
		if err != nil {
			return nil, err
		}

		out = append(out, Generator{
			Height: i,
			PubKey: block.GenPublicKey,
		})
	}

	return out, nil
}
