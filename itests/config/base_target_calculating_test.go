package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/types"
	"testing"
)

func TestCalculateBaseTarget(t *testing.T) {
	settings := GenesisSettings{
		AverageBlockDelay:    10,
		MinBlockTime:         5000,
		DelayDelta:           0,
		Distributions:        nil,
		PreactivatedFeatures: []FeatureInfo{{Feature: 8}, {Feature: 15}},
	}
	pos := getPosCalculator(&settings)

	tests := []struct {
		balance    uint64
		baseTarget types.BaseTarget
	}{
		{balance: 10000000000000, baseTarget: 468754},
		{balance: 15000000000000, baseTarget: 312505},
		{balance: 25000000000000, baseTarget: 187506},
		{balance: 25000000000000, baseTarget: 187506},
		{balance: 40000000000000, baseTarget: 117194},
		{balance: 45000000000000, baseTarget: 105475},
		{balance: 60000000000000, baseTarget: 78132},
		{balance: 80000000000000, baseTarget: 58601},
		{balance: 100000000000000, baseTarget: 46883},
		{balance: 6000000000000000, baseTarget: 771},
	}
	for _, tc := range tests {
		bt, err := calculateBaseTarget(pos, consensus.MinBaseTarget, maxBaseTarget, tc.balance, settings.AverageBlockDelay)
		assert.NoError(t, err)
		assert.Equal(t, bt, tc.baseTarget)
	}
}
