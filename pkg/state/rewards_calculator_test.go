package state

import (
	"strconv"
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

func makeMockFeaturesStateForRewardsCalc(features ...settings.Feature) (*mockFeaturesState, func(proto.Height)) {
	var currentHeight proto.Height
	setCurrentHeight := func(height proto.Height) {
		currentHeight = height
	}
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
		newestIsActivatedFunc: func(featureID int16) (bool, error) {
			_, isEnabled := enabledFeatures[featureID]
			activations := map[settings.Feature]bool{}
			if currentHeight >= 1000 {
				activations[settings.BlockRewardDistribution] = isEnabled
			}
			if currentHeight >= 2000 {
				activations[settings.CappedRewards] = isEnabled
			}
			if currentHeight >= 3000 {
				activations[settings.XTNBuyBackCessation] = isEnabled
			}
			return activations[settings.Feature(featureID)], nil
		},
	}
	return mf, setCurrentHeight
}

func newTestRewardsCalculator(features ...settings.Feature) (*rewardCalculator, func(proto.Height)) {
	mf, fn := makeMockFeaturesStateForRewardsCalc(features...)
	sets := *settings.TestNetSettings
	sets.MinXTNBuyBackPeriod = 3000
	c := newRewardsCalculator(&sets, mf)
	return c, fn
}

func TestFeature19RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c, setCurrentHeight := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
	)
	for i, test := range []struct {
		height        proto.Height
		currentHeight proto.Height
		reward        uint64
		rewards       proto.Rewards
	}{
		{900, 900, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{900, 900, 6_5000_0000, makeTestNetRewards(t, gen, 6_5000_0000)},
		{1000, 1000, 6_5000_0000, makeTestNetRewards(t, gen, 2_1666_6668, 2_1666_6666, 2_1666_6666)},
		{900, 900, 3_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000)},
		{1000, 1000, 3_0000_0000, makeTestNetRewards(t, gen, 1_0000_0000, 1_0000_0000, 1_0000_0000)},
		{900, 900, 0, makeTestNetRewards(t, gen, 0)},
		{1000, 1000, 0, makeTestNetRewards(t, gen, 0, 0, 0)},
	} {
		setCurrentHeight(test.currentHeight)
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

	c, setCurrentHeight := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.XTNBuyBackCessation,
	)
	for i, test := range []struct {
		height        proto.Height
		currentHeight proto.Height
		reward        uint64
		rewards       proto.Rewards
	}{
		{999, 999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},
		{1000, 1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2999, 2999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3999, 3999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 4000, 6_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000, 3_0000_0000)},
		{5000, 5000, 6_0000_0000, makeTestNetRewards(t, gen, 3_0000_0000, 3_0000_0000)},
	} {
		setCurrentHeight(test.currentHeight)
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

	c, setCurrentHeight := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
	)
	for i, test := range []struct {
		height        proto.Height
		currentHeight proto.Height
		reward        uint64
		rewards       proto.Rewards
	}{
		{999, 999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		// test for compatibility with scala node behaviour
		{999, 1999, 6_3333_3333, makeTestNetRewards(t, gen, 2_1111_1111, 2_1111_1111, 2_1111_1111)},
		{999, 2000, 6_3333_3333, makeTestNetRewards(t, gen, 6_3333_3333)},
		{1500, 2000, 6_3333_3333, makeTestNetRewards(t, gen, 2_1111_1111, 2_1111_1111, 2_1111_1111)},
		{2000, 2000, 6_3333_3333, makeTestNetRewards(t, gen, 2_3333_3333, 2_0000_0000, 2_0000_0000)},

		{2000, 2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		{3000, 3000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{3000, 3000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{3000, 3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{3000, 3000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{3000, 3000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},
	} {
		setCurrentHeight(test.currentHeight)
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

	c, setCurrentHeight := newTestRewardsCalculator(
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.XTNBuyBackCessation,
	)
	for i, test := range []struct {
		height        proto.Height
		currentHeight proto.Height
		reward        uint64
		rewards       proto.Rewards
	}{
		{999, 999, 6_0000_0000, makeTestNetRewards(t, gen, 6_0000_0000)},

		{1000, 1000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{1999, 1999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		{2000, 2000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{2000, 2000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{2000, 2000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},
		{2000, 2000, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{2000, 2000, 10_1234_5678, makeTestNetRewards(t, gen, 6_1234_5678, 2_0000_0000, 2_0000_0000)},

		// reward addresses remains the same because xtn buyback period is still continuing
		{3000, 3000, 4_2222_2222, makeTestNetRewards(t, gen, 2_0000_0000, 1_1111_1111, 1_1111_1111)},

		{4000, 4000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{4000, 4000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{4000, 4000, 4_2222_2222, makeTestNetRewards(t, gen, 3_1111_1111, 1_1111_1111)},
		{5000, 5000, 5_0000_0000, makeTestNetRewards(t, gen, 3_5000_0000, 1_5000_0000)},
		{4000, 4000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{4000, 4000, 10_1234_5678, makeTestNetRewards(t, gen, 8_1234_5678, 2_0000_0000)},

		{5000, 5000, 1_9999_9999, makeTestNetRewards(t, gen, 1_9999_9999)},
		{5000, 5000, 2_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000)},
		{5000, 5000, 4_2222_2222, makeTestNetRewards(t, gen, 3_1111_1111, 1_1111_1111)},
		{5000, 5000, 5_0000_0000, makeTestNetRewards(t, gen, 3_5000_0000, 1_5000_0000)},
		{5000, 5000, 6_0000_0000, makeTestNetRewards(t, gen, 4_0000_0000, 2_0000_0000)},
		{5000, 5000, 10_1234_5678, makeTestNetRewards(t, gen, 8_1234_5678, 2_0000_0000)},
	} {
		setCurrentHeight(test.currentHeight)
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}
