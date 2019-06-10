package state

import (
	"math/big"
	"testing"
)

const (
	genesisScoreStr = "120000000219"
)

func TestCalculateScore(t *testing.T) {
	var genesisScore big.Int
	genesisScore.SetString(genesisScoreStr, 10)
	genesisTarget := uint64(153722867)
	result, err := CalculateScore(genesisTarget)
	if err != nil {
		t.Fatalf("Failed to calculate score: %v\n", err)
	}
	if result.Cmp(&genesisScore) != 0 {
		t.Errorf("Scores are not equal.")
	}
}
