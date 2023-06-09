package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func makeTestNetRewards(t *testing.T, gen proto.WavesAddress, amounts ...uint64) proto.Rewards {
	require.True(t, len(amounts) > 0 && len(amounts) <= 3)
	addresses := make([]proto.WavesAddress, 3)
	addresses[0] = gen
	copy(addresses[1:], settings.TestNetSettings.RewardAddresses)
	r := make(proto.Rewards, 0, 3)
	for i, a := range amounts {
		r = append(r, proto.NewReward(addresses[i], a))
	}
	return r
}

func TestFeature19RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)
	mf := &mockFeaturesState{
		newestIsActivatedAtHeightFunc: func(featureID int16, height uint64) bool {
			switch featureID {
			case int16(settings.BlockRewardDistribution):
				return height >= 1000
			default:
				return false
			}
		},
	}
	c := newRewardsCalculator(settings.TestNetSettings, mf)
	for _, test := range []struct {
		height  uint64
		reward  uint64
		rewards proto.Rewards
	}{
		{900, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{900, 6_5000_0000, makeTestNetRewards(t, gen, 6_5000_0000)},
		{1000, 6_5000_0000, makeTestNetRewards(t, gen, 2_1666_6668, 2_1666_6666, 2_1666_6666)},
		{900, 3_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000)},
		{1000, 3_0000_0000, makeTestNetRewards(t, gen, 1_0000_0000, 1_0000_0000, 1_0000_0000)},
		{900, 0, makeTestNetRewards(t, gen, 0)},
		{1000, 0, makeTestNetRewards(t, gen, 0, 0, 0)},
	} {
		actual, err := c.calculateRewards(gen, test.height, test.reward)
		require.NoError(t, err)
		assert.ElementsMatch(t, test.rewards, actual)
	}
}

func TestFeatures19And21RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)
	mf := &mockFeaturesState{
		newestIsActivatedAtHeightFunc: func(featureID int16, height uint64) bool {
			switch featureID {
			case int16(settings.BlockRewardDistribution):
				return height >= 1000
			case int16(settings.XTNBuyBackCessation):
				return height >= 2000
			default:
				return false
			}
		},
	}
	c := newRewardsCalculator(settings.TestNetSettings, mf)
	for _, test := range []struct {
		height  uint64
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 6_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000, 3_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000, 3_0000_0000)},
	} {
		actual, err := c.calculateRewards(gen, test.height, test.reward)
		require.NoError(t, err)
		assert.ElementsMatch(t, test.rewards, actual)
	}
}
