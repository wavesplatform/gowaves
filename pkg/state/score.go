package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func CalculateScore(baseTarget uint64) (*big.Int, error) {
	if baseTarget == 0 {
		return nil, errors.New("zero base target")
	}
	var max big.Int
	max.SetString("18446744073709551616", 10)
	var target big.Int
	target.SetUint64(baseTarget)
	var score big.Int
	score.Div(&max, &target)
	return &score, nil
}

type scores struct {
	hs *historyStorage
}

func newScores(hs *historyStorage) *scores {
	return &scores{hs}
}

func (s *scores) appendBlockScore(block *proto.Block, height uint64) error {
	blockScore, err := CalculateScore(block.BaseTarget)
	if err != nil {
		return err
	}
	if height > 1 {
		prevScore, err := s.newestScore(height - 1)
		if err != nil {
			return err
		}
		blockScore.Add(blockScore, prevScore)
	}
	scoreKey := scoreKey{height: height}
	return s.hs.addNewEntry(score, scoreKey.bytes(), blockScore.Bytes(), block.BlockID())
}

func scoreFromBytes(scoreBytes []byte) *big.Int {
	var score big.Int
	score.SetBytes(scoreBytes)
	return &score
}

func (s *scores) score(height uint64) (*big.Int, error) {
	key := scoreKey{height: height}
	scoreBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	return scoreFromBytes(scoreBytes), nil
}

func (s *scores) newestScore(height uint64) (*big.Int, error) {
	key := scoreKey{height: height}
	scoreBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	return scoreFromBytes(scoreBytes), nil
}
