package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func makeTestNetRewards(t *testing.T, gen proto.WavesAddress, amounts ...uint64) proto.Rewards {
	s := settings.MustTestNetSettings()
	require.True(t, len(amounts) > 0 && len(amounts) <= maxRewardPayoutsCount)
	require.NotNil(t, s.DAOAddress)
	require.NotNil(t, s.XTNBuybackAddress)
	addresses := []proto.WavesAddress{gen, *s.DAOAddress, *s.XTNBuybackAddress}
	r := make(proto.Rewards, 0, maxRewardPayoutsCount)
	for i, a := range amounts {
		r = append(r, proto.NewReward(addresses[i], a))
	}
	return r
}

func makeMockFeaturesStateForRewardsCalc(t *testing.T, features ...settings.Feature) featuresStateForRewardsCalculator {
	enabledFeatures := make(map[int16]struct{}, len(features))
	for _, f := range features {
		enabledFeatures[int16(f)] = struct{}{}
	}
	mf := NewMockFeaturesState(t)
	mf.EXPECT().newestIsActivatedAtHeight(mock.Anything, mock.Anything).RunAndReturn(
		func(featureID int16, height uint64) bool {
			_, isEnabled := enabledFeatures[featureID]
			switch settings.Feature(featureID) {
			case settings.BlockRewardDistribution:
				return height >= 1000 && isEnabled
			case settings.CappedRewards:
				return height >= 2000 && isEnabled
			case settings.XTNBuyBackCessation:
				return height >= 3000 && isEnabled
			case settings.AdjustedBlockRewardDistribution:
				return height >= 5000 && isEnabled
			case settings.SmallerMinimalGeneratingBalance, settings.NG, settings.MassTransfer, settings.SmartAccounts,
				settings.DataTransaction, settings.BurnAnyTokens, settings.FeeSponsorship, settings.FairPoS,
				settings.SmartAssets, settings.SmartAccountTrading, settings.Ride4DApps, settings.OrderV3,
				settings.ReducedNFTFee, settings.BlockReward, settings.BlockV5, settings.RideV5, settings.RideV6,
				settings.ConsensusImprovements, settings.LightNode, settings.BoostBlockReward,
				settings.DeterministicFinality, settings.InvokeExpression:
				return false
			default:
				panic(fmt.Sprintf("unknown feature ID %d", featureID))
			}
		}).Maybe()
	mf.EXPECT().newestActivationHeight(mock.Anything).RunAndReturn(func(featureID int16) (uint64, error) {
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
		case settings.AdjustedBlockRewardDistribution:
			return 5000, nil
		default:
			return 0, keyvalue.ErrNotFound
		}
	}).Maybe()
	return mf
}

func newTestRewardsCalculator(t *testing.T, features ...settings.Feature) *rewardCalculator {
	mf := makeMockFeaturesStateForRewardsCalc(t, features...)
	sets := settings.MustTestNetSettings()
	sets.MinXTNBuyBackPeriod = 3000
	sets.BlockRewardBoostPeriod = 1000
	c := newRewardsCalculator(sets, mf)
	return c
}

func TestFeature19RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		{900, 0, proto.Rewards{}},
		{1000, 0, proto.Rewards{}},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And21RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And21RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, err := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeature19And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		{900, 0, proto.Rewards{}},
		{1000, 0, proto.Rewards{}},
		{4000, 0, proto.Rewards{}},
		{5000, 0, proto.Rewards{}},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And21And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And21And23RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
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
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

// makeRewards builds the expected rewards from the given address and amount pairs skipping zero amounts
// the same way the rewards calculator does it.
func makeRewards(t *testing.T, pairs ...any) proto.Rewards {
	require.True(t, len(pairs)%2 == 0, "address and amount pairs expected")
	r := make(proto.Rewards, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		addr, ok := pairs[i].(proto.WavesAddress)
		require.True(t, ok, "address expected at position %d", i)
		amount, ok := pairs[i+1].(uint64)
		require.True(t, ok, "amount expected at position %d", i+1)
		if amount == 0 {
			continue
		}
		r = append(r, proto.NewReward(addr, amount))
	}
	return r
}

func TestFeatures19And20And26RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.AdjustedBlockRewardDistribution,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		// Before the activation of feature 26 the default distribution is in effect.
		{4999, 20_0000_0000, makeTestNetRewards(t, gen, 16_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4999, 6_0000_0000, makeTestNetRewards(t, gen, 2_0000_0000, 2_0000_0000, 2_0000_0000)},

		// At the full reward of 20 WAVES the shares are 8 (miner) / 10 (DAO) / 2 (XTN buy-back).
		{5000, 20_0000_0000, makeTestNetRewards(t, gen, 8_0000_0000, 10_0000_0000, 2_0000_0000)},
		// Above the full reward the addresses receive their maximum shares, the miner gets the rest.
		{5000, 26_0000_0000, makeTestNetRewards(t, gen, 14_0000_0000, 10_0000_0000, 2_0000_0000)},
		{5000, 100_0000_0000, makeTestNetRewards(t, gen, 88_0000_0000, 10_0000_0000, 2_0000_0000)},
		// Below the full reward the miner keeps its guaranteed 8 WAVES and the addresses share the
		// remainder as 5/6 and 1/6.
		{5000, 14_0000_0000, makeTestNetRewards(t, gen, 8_0000_0000, 5_0000_0000, 1_0000_0000)},
		// The remainder is divided first and multiplied after, the truncated part goes to the miner.
		{5000, 19_5000_0000, makeTestNetRewards(t, gen, 8_0000_0004, 9_5833_3330, 1_9166_6666)},
		// At exactly the guaranteed miner reward the addresses receive nothing and are not included.
		{5000, 8_0000_0000, makeTestNetRewards(t, gen, 8_0000_0000)},
		// Below the guaranteed miner reward the miner receives the whole reward.
		{5000, 7_5000_0000, makeTestNetRewards(t, gen, 7_5000_0000)},
		{5000, 0, proto.Rewards{}},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And21And26RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.XTNBuyBackCessation,
		settings.AdjustedBlockRewardDistribution,
	)
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		// The XTN buy-back is ceased at height 4000, its share goes to the miner from there on.
		{3999, 20_0000_0000, makeTestNetRewards(t, gen, 16_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 20_0000_0000, makeTestNetRewards(t, gen, 18_0000_0000, 2_0000_0000)},
		{5000, 20_0000_0000, makeTestNetRewards(t, gen, 10_0000_0000, 10_0000_0000)},
		{5000, 14_0000_0000, makeTestNetRewards(t, gen, 9_0000_0000, 5_0000_0000)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeatures19And20And23And26RewardCalculation(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	c := newTestRewardsCalculator(t,
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.BoostBlockReward,
		settings.AdjustedBlockRewardDistribution,
	)
	// Make the boost period long enough to outlive the activation of feature 26.
	c.settings.BlockRewardBoostPeriod = 3000
	for i, test := range []struct {
		height  proto.Height
		reward  uint64
		rewards proto.Rewards
	}{
		// Feature 23 is activated at height 4000 and boosts the reward tenfold.
		{3999, 20_0000_0000, makeTestNetRewards(t, gen, 16_0000_0000, 2_0000_0000, 2_0000_0000)},
		{4000, 20_0000_0000, makeTestNetRewards(t, gen, 160_0000_0000, 20_0000_0000, 20_0000_0000)},
		{4999, 20_0000_0000, makeTestNetRewards(t, gen, 160_0000_0000, 20_0000_0000, 20_0000_0000)},
		// Feature 26 supersedes feature 23, the reward is not boosted anymore.
		{5000, 20_0000_0000, makeTestNetRewards(t, gen, 8_0000_0000, 10_0000_0000, 2_0000_0000)},
		{6000, 20_0000_0000, makeTestNetRewards(t, gen, 8_0000_0000, 10_0000_0000, 2_0000_0000)},
	} {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			actual, cErr := c.calculateRewards(gen, test.height, test.reward)
			require.NoError(t, cErr)
			assert.ElementsMatch(t, test.rewards, actual)
		})
	}
}

func TestFeature26RequiresFeatures19And20(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	t.Run("without feature 19 the miner gets everything", func(t *testing.T) {
		c := newTestRewardsCalculator(t, settings.AdjustedBlockRewardDistribution)
		actual, cErr := c.calculateRewards(gen, 5000, 20_0000_0000)
		require.NoError(t, cErr)
		assert.ElementsMatch(t, makeTestNetRewards(t, gen, 20_0000_0000), actual)
	})
	t.Run("without feature 20 the reward is shared equally", func(t *testing.T) {
		c := newTestRewardsCalculator(t,
			settings.BlockRewardDistribution,
			settings.AdjustedBlockRewardDistribution,
		)
		actual, cErr := c.calculateRewards(gen, 5000, 20_0000_0000)
		require.NoError(t, cErr)
		expected := makeTestNetRewards(t, gen, 6_6666_6668, 6_6666_6666, 6_6666_6666)
		assert.ElementsMatch(t, expected, actual)
	})
}

func TestFeature26RewardCalculationWithSingleAddress(t *testing.T) {
	gen, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	features := []settings.Feature{
		settings.BlockRewardDistribution,
		settings.CappedRewards,
		settings.AdjustedBlockRewardDistribution,
	}
	t.Run("DAO address only", func(t *testing.T) {
		c := newTestRewardsCalculator(t, features...)
		dao := *c.settings.DAOAddress
		c.settings.XTNBuybackAddress = nil
		actual, cErr := c.calculateRewards(gen, 5000, 20_0000_0000)
		require.NoError(t, cErr)
		expected := makeRewards(t, gen, uint64(10_0000_0000), dao, uint64(10_0000_0000))
		assert.ElementsMatch(t, expected, actual)
	})
	t.Run("XTN buy-back address only", func(t *testing.T) {
		c := newTestRewardsCalculator(t, features...)
		xtn := *c.settings.XTNBuybackAddress
		c.settings.DAOAddress = nil
		actual, cErr := c.calculateRewards(gen, 5000, 20_0000_0000)
		require.NoError(t, cErr)
		expected := makeRewards(t, gen, uint64(18_0000_0000), xtn, uint64(2_0000_0000))
		assert.ElementsMatch(t, expected, actual)
	})
	t.Run("no reward addresses", func(t *testing.T) {
		c := newTestRewardsCalculator(t, features...)
		c.settings.DAOAddress = nil
		c.settings.XTNBuybackAddress = nil
		actual, cErr := c.calculateRewards(gen, 5000, 20_0000_0000)
		require.NoError(t, cErr)
		expected := makeRewards(t, gen, uint64(20_0000_0000))
		assert.ElementsMatch(t, expected, actual)
	})
}
