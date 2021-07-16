package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

//const (
//	maxBlocksSequenceLength = 100
//)

type Score struct {
	Score string `json:"score"`
}

func (a *App) BlocksScoreAt(at proto.Height) (Score, error) {
	score, err := a.state.ScoreAtHeight(at)
	if err != nil {
		return Score{}, err
	}
	return Score{Score: score.String()}, nil
}

func (a *App) BlockByHeight(height proto.Height) (*proto.Block, error) {
	block, err := a.state.BlockByHeight(height)
	if err != nil {
		if origErr := errors.Cause(err); state.IsInvalidInput(origErr) || state.IsNotFound(origErr) {
			// nickeskov: in this cases scala node sends empty response
			return nil, appErrorNoData
		}
		return nil, errors.Wrapf(err, "failed to get block by height=%d", height)
	}
	return block, nil
}

func (a *App) HeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	header, err := a.state.HeaderByHeight(height)
	if err != nil {
		if origErr := errors.Cause(err); state.IsInvalidInput(origErr) || state.IsNotFound(origErr) {
			return nil, appErrorNoData
		}
		return nil, errors.Wrapf(err, "failed to get block header by height=%d", height)
	}
	return header, nil
}

func (a *App) Block(id proto.BlockID) (*proto.Block, error) {
	block, err := a.state.Block(id)
	if err != nil {
		if origErr := errors.Cause(err); state.IsNotFound(origErr) {
			return nil, appErrorNoData
		}
		return nil, errors.Wrapf(err, "failed to get block by id=%s", id.String())
	}
	return block, nil
}

func (a *App) Header(id proto.BlockID) (*proto.BlockHeader, error) {
	header, err := a.state.Header(id)
	if err != nil {
		if origErr := errors.Cause(err); state.IsNotFound(origErr) {
			return nil, appErrorNoData
		}
		return nil, errors.Wrapf(err, "failed to get block header by id=%s", id.String())
	}
	return header, nil
}

func (a *App) BlockIDToHeight(id proto.BlockID) (proto.Height, error) {
	height, err := a.state.BlockIDToHeight(id)
	if err != nil {
		if origErr := errors.Cause(err); state.IsNotFound(origErr) {
			return 0, appErrorNoData
		}
		return 0, errors.Wrapf(err, "failed to get block height for id=%s", id.String())
	}
	return height, nil
}

func (a *App) BlocksLast() (*proto.Block, error) {
	h, err := a.state.Height()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state height")
	}

	block, err := a.state.BlockByHeight(h)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %d block from state", h)
	}
	block.Height = h
	return block, nil
}

func (a *App) BlocksLastHeader() (*proto.BlockHeader, error) {
	h, err := a.state.Height()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state height")
	}

	blockHeader, err := a.state.HeaderByHeight(h)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %d block header from state", h)
	}
	blockHeader.Height = h
	return blockHeader, nil
}

func (a *App) BlocksFirst() (*proto.Block, error) {
	block, err := a.state.BlockByHeight(1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get first block from state")
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
		return nil, errors.Wrap(err, "failed to get state height")
	}

	// show only last 150 rows
	initialHeight := proto.Height(1)
	if curHeight > 150 {
		initialHeight = curHeight - 150
	}

	out := Generators{}
	for i := initialHeight; i < curHeight; i++ {
		block, err := a.state.BlockByHeight(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get from state block by height %d", i)
		}

		out = append(out, Generator{
			Height: i,
			PubKey: block.GenPublicKey,
		})
	}

	return out, nil
}
