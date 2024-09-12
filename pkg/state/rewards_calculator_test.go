package state

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func makeTestNetRewards(t *testing.T, gen proto.WavesAddress, amounts ...uint64) proto.Rewards {
	s := settings.MustTestNetSettings()
	require.True(t, len(amounts) > 0 && len(amounts) <= 3)
	addresses := make([]proto.WavesAddress, 3)
	addresses[0] = gen
	copy(addresses[1:], s.RewardAddresses)
	r := make(proto.Rewards, 0, 3)
	for i, a := range amounts {
		r = append(r, proto.NewReward(addresses[i], a))
	}
	return r
}

func makeMockFeaturesStateForRewardsCalc(features ...settings.Feature) featuresStateForRewardsCalculator {
	enabledFeatures := make(map[int16]struct{}, len(features))
	for _, f := range features {
		enabledFeatures[int16(f)] = struct{}{}
	}
	mf := &mockFeaturesState{
		newestIsActivatedAtHeightFunc: func(featureID int16, height uint64) bool {
			_, isEnabled := enabledFeatures[featureID]
			switch settings.Feature(featureID) {
			case settings.BlockRewardDistribution:
				return height >= 1000 && isEnabled
			case settings.CappedRewards:
				return height >= 2000 && isEnabled
			case settings.XTNBuyBackCessation:
				return height >= 3000 && isEnabled
			default:
				return false
			}
		},
		newestActivationHeightFunc: func(featureID int16) (uint64, error) {
			_, enabled := enabledFeatures[featureID]
			if !enabled {
				return 0, keyvalue.ErrNotFound
			}
			switch settings.Feature(featureID) { //nolint:exhaustive // only relevant features
			case settings.BlockRewardDistribution:
				return 1000, nil
			case settings.CappedRewards:
				return 2000, nil
			case settings.XTNBuyBackCessation:
				return 3000, nil
			case settings.BoostBlockReward:
				return 4000, nil
			default:
				return 0, keyvalue.ErrNotFound
			}
		},
	}
	return mf
}

func newTestRewardsCalculator(features ...settings.Feature) *rewardCalculator {
	mf := makeMockFeaturesStateForRewardsCalc(features...)
	sets := settings.MustTestNetSettings()
	sets.MinXTNBuyBackPeriod = 3000
	sets.BlockRewardBoostPeriod = 1000
	c := newRewardsCalculator(sets, mf)
	return c
}

func TestFeature19RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
	)
	for i, test := range []struct {
		height  proto.Height
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
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And21RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.XTNBuyBackCessation,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{999, 6_3333_3333, makeTestNetRewards(t, gen, 6_3333_3333)},
		{1000, 6_3333_3333, makeTestNetRewards(t, gen, 2_1111_1111, 2_1111_1111, 2_1111_1111)},
		{1500, 6_3333_3333, makeTestNetRewards(t, gen, 2_1111_1111, 2_1111_1111, 2_1111_1111)},
		{2000, 6_3333_3333, makeTestNetRewards(t, gen, 2_3333_3333, 2_0000_0000, 2_0000_0000)},

		{2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		{3000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{3000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And21RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.XTNBuyBackCessation,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		// reward addresses remains the same because xtn buyback period is still continuing
		{3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},

		{4000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{4000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{4000, 4_2222_2222, makeTestNetRewards(t, gen, 3_1111_1111, 1_1111_1111)},
		{4000, 5_0000_0000, makeTestNetRewards(t, gen, 3_5000_0000, 1_5000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{4000, 10_1234_5678, makeTestNetRewards(t, gen, 8_1234_5678, 2_0000_0000)},

		{5000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{5000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{5000, 4_2222_2222, makeTestNetRewards(t, gen, 3_1111_1111, 1_1111_1111)},
		{5000, 5_0000_0000, makeTestNetRewards(t, gen, 3_5000_0000, 1_5000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{5000, 10_1234_5678, makeTestNetRewards(t, gen, 8_1234_5678, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BoostBlockReward,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{3999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 60_0000_0000)},
		{4999, 6_0000_0000, makeTestNetRewards(t, gen, 60_0000_0000)},

		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{5099, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeature19And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.BoostBlockReward,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{900, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 20_0000_0000, 20_0000_0000, 20_0000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{900, 6_5000_0000, makeTestNetRewards(t, gen, 6_5000_0000)},
		{1000, 6_5000_0000, makeTestNetRewards(t, gen, 2_1666_6668, 2_1666_6666, 2_1666_6666)},
		{4000, 6_5000_0000, makeTestNetRewards(t, gen, 21_6666_6680, 21_6666_6660, 21_6666_6660)},
		{5000, 6_5000_0000, makeTestNetRewards(t, gen, 2_1666_6668, 2_1666_6666, 2_1666_6666)},
		{900, 3_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000)},
		{1000, 3_0000_0000, makeTestNetRewards(t, gen, 1_0000_0000, 1_0000_0000, 1_0000_0000)},
		{4000, 3_0000_0000, makeTestNetRewards(t, gen, 10_0000_0000, 10_0000_0000, 10_0000_0000)},
		{5000, 3_0000_0000, makeTestNetRewards(t, gen, 1_0000_0000, 1_0000_0000, 1_0000_0000)},
		{900, 0, makeTestNetRewards(t, gen, 0)},
		{1000, 0, makeTestNetRewards(t, gen, 0, 0, 0)},
		{4000, 0, makeTestNetRewards(t, gen, 0, 0, 0)},
		{5000, 0, makeTestNetRewards(t, gen, 0, 0, 0)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And21And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.XTNBuyBackCessation,
		settings.BoostBlockReward,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 40_0000_0000, 20_0000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.BoostBlockReward,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 20_0000_0000, 20_0000_0000, 20_0000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{999, 6_3333_3333, makeTestNetRewards(t, gen, 6_3333_3333)},
		{1500, 6_3333_3333, makeTestNetRewards(t, gen, 2_1111_1111, 2_1111_1111, 2_1111_1111)},
		{2000, 6_3333_3333, makeTestNetRewards(t, gen, 2_3333_3333, 2_0000_0000, 2_0000_0000)},

		{2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		{3000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{3000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		{4000, 1_9999_9999, makeTestNetRewards(t, gen, 19_9999_9990)},
		{4000, 2_0000_0000, makeTestNetRewards(t, gen, 20_0000_0000)},
		{4000, 4_2222_2222, makeTestNetRewards(t, gen, 20_0000_0000, 11_1111_1110, 11_1111_1110)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 20_0000_0000, 20_0000_0000, 20_0000_0000)},
		{4000, 10_1234_5678, makeTestNetRewards(t, gen, 61_2345_6780, 20_0000_0000, 20_0000_0000)},

		{5000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{5000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{5000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{5000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And21And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.XTNBuyBackCessation,
		settings.BoostBlockReward,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		{999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		// reward addresses remains the same because xtn buyback period is still continuing
		{3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},

		{4000, 1_9999_9999, makeTestNetRewards(t, gen, 19_9999_9990)},
		{4000, 2_0000_0000, makeTestNetRewards(t, gen, 20_0000_0000)},
		{4000, 4_2222_2222, makeTestNetRewards(t, gen, 31_1111_1110, 11_1111_1110)},
		{4000, 5_0000_0000, makeTestNetRewards(t, gen, 35_0000_0000, 15_0000_0000)},
		{4000, 6_0000_0000, makeTestNetRewards(t, gen, 40_0000_0000, 20_0000_0000)},
		{4000, 10_1234_5678, makeTestNetRewards(t, gen, 81_2345_6780, 20_0000_0000)},

		{5000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{5000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{5000, 4_2222_2222, makeTestNetRewards(t, gen, 3_1111_1111, 1_1111_1111)},
		{5000, 5_0000_0000, makeTestNetRewards(t, gen, 3_5000_0000, 1_5000_0000)},
		{5000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{5000, 10_1234_5678, makeTestNetRewards(t, gen, 8_1234_5678, 2_0000_0000)},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}
