package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

func calculateScore(baseTarget uint64) (*big.Int, error) {
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
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch
}

func newScores(db keyvalue.KeyValue, dbBatch keyvalue.Batch) (*scores, error) {
	return &scores{db: db, dbBatch: dbBatch}, nil
}

func (s *scores) saveScoreToDb(score *big.Int, height uint64) error {
	key := scoreKey{height: height}
	s.dbBatch.Put(key.bytes(), score.Bytes())
	return nil
}

func (s *scores) addScore(prevScore, score *big.Int, height uint64) error {
	score.Add(score, prevScore)
	if err := s.saveScoreToDb(score, height); err != nil {
		return err
	}
	return nil
}

func (s *scores) score(height uint64) (*big.Int, error) {
	key := scoreKey{height: height}
	scoreBytes, err := s.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	var score big.Int
	score.SetBytes(scoreBytes)
	return &score, nil
}

func (s *scores) rollback(newHeight, oldHeight uint64) error {
	for h := oldHeight; h > newHeight; h-- {
		key := scoreKey{height: h}
		if err := s.db.Delete(key.bytes()); err != nil {
			return err
		}
	}
	return nil
}
