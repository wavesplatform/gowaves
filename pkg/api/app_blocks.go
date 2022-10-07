package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Score struct {
	Score string `json:"score"`
}

type Block struct {
	*proto.Block
	Generator proto.WavesAddress `json:"generator"`
	Height    proto.Height       `json:"height"`
}

func newAPIBlock(block *proto.Block, scheme proto.Scheme, height proto.Height) (*Block, error) {
	generator, err := proto.NewAddressFromPublicKey(scheme, block.GeneratorPublicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate address from public key %q", block.GeneratorPublicKey)
	}
	return &Block{
		Block:     block,
		Generator: generator,
		Height:    height,
	}, nil
}

func newAPIBlockFromHeader(header proto.BlockHeader, scheme proto.Scheme, height proto.Height) (*Block, error) {
	block := &proto.Block{
		BlockHeader:  header,
		Transactions: nil,
	}
	return newAPIBlock(block, scheme, height)
}

func (a *App) BlocksScoreAt(at proto.Height) (Score, error) {
	score, err := a.state.ScoreAtHeight(at)
	if err != nil {
		return Score{}, err
	}
	return Score{Score: score.String()}, nil
}

func (a *App) BlocksLast() (*Block, error) {
	h, err := a.state.Height()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state height")
	}
	block, err := a.state.BlockByHeight(h)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %d block from state", h)
	}
	return newAPIBlock(block, a.services.Scheme, h)
}

func (a *App) BlocksHeadersLast() (*Block, error) {
	h, err := a.state.Height()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state height")
	}
	blockHeader, err := a.state.HeaderByHeight(h)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %d block header from state", h)
	}
	return newAPIBlockFromHeader(*blockHeader, a.services.Scheme, h)
}

func (a *App) BlocksFirst() (*Block, error) {
	const genesisHeight = 1
	block, err := a.state.BlockByHeight(genesisHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get first block from state")
	}
	return newAPIBlock(block, a.services.Scheme, genesisHeight)
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
			PubKey: block.GeneratorPublicKey,
		})
	}

	return out, nil
}
