package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	genesisScoreStr = "120000000219"
)

func TestCalculateScore(t *testing.T) {
	genesisScore, ok := new(big.Int).SetString(genesisScoreStr, 10)
	require.True(t, ok)
	genesisTarget := uint64(153722867)
	result, err := CalculateScore(genesisTarget)
	require.NoError(t, err)
	assert.Equal(t, genesisScore, result)
}

func TestCalculateScoreFromZeroValue(t *testing.T) {
	var bt uint64
	_, err := CalculateScore(bt)
	assert.Error(t, err)
}
