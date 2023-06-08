package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func makeTestNetRewards(t *testing.T, pk crypto.PublicKey, amounts ...uint64) proto.Rewards {
	require.True(t, len(amounts) > 0 && len(amounts) <= 3)
	addresses := make([]proto.WavesAddress, 3)
	var err error
	addresses[0], err = proto.NewAddressFromPublicKey(proto.TestNetScheme, pk)
	require.NoError(t, err)
	copy(addresses[1:], settings.TestNetSettings.RewardAddresses)
	r := make(proto.Rewards, 0, 3)
	for i, a := range amounts {
		r = append(r, proto.NewReward(addresses[i], a))
	}
	return r
}

func TestFeature19RewardCalculation(t *testing.T) {
	gpk, err := crypto.NewPublicKeyFromBase58(testPK)
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
	header := &proto.BlockHeader{
		GeneratorPublicKey: gpk,
	}
	for _, test := range []struct {
		height  uint64
		reward  uint64
		rewards proto.Rewards
	}{
		{900, 6_0000_0000, makeTestNetRewards(t, gpk, 6_0000_0000)},
		{1000, 6_0000_0000, makeTestNetRewards(t, gpk, 2_0000_0000, 2_0000_0000, 2_0000_0000)},
		{900, 6_5000_0000, makeTestNetRewards(t, gpk, 6_5000_0000)},
		{1000, 6_5000_0000, makeTestNetRewards(t, gpk, 2_1666_6668, 2_1666_6666, 2_1666_6666)},
		{900, 3_0000_0000, makeTestNetRewards(t, gpk, 3_0000_0000)},
		{1000, 3_0000_0000, makeTestNetRewards(t, gpk, 1_0000_0000, 1_0000_0000, 1_0000_0000)},
		{900, 0, makeTestNetRewards(t, gpk, 0)},
		{1000, 0, makeTestNetRewards(t, gpk, 0, 0, 0)},
	} {
		actual, err := c.calculateRewards(header, test.height, test.reward)
		require.NoError(t, err)
		assert.ElementsMatch(t, test.rewards, actual)
	}
}
