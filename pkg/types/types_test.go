package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func TestWavesBalanceProfileToFullWavesBalanceTakesDepositIntoAccount(t *testing.T) {
	profile := &types.WavesBalanceProfile{
		Balance:    1_000,
		LeaseIn:    200,
		LeaseOut:   100,
		Deposit:    300,
		Generating: 700,
	}

	actual, err := profile.ToFullWavesBalance()
	require.NoError(t, err)
	require.Equal(t, &proto.FullWavesBalance{
		Regular:    1_000,
		Generating: 700,
		Available:  600,
		Effective:  800,
		LeaseIn:    200,
		LeaseOut:   100,
	}, actual)
}

func TestWavesBalanceProfileDepositCanMakeBalancesNegative(t *testing.T) {
	profile := &types.WavesBalanceProfile{
		Balance: 100,
		Deposit: 101,
	}

	_, err := profile.SpendableBalance()
	require.Error(t, err)
	_, err = profile.EffectiveBalance()
	require.Error(t, err)
}

func TestWavesBalanceProfileChallengedEffectiveBalance(t *testing.T) {
	profile := &types.WavesBalanceProfile{
		Balance:    1_000,
		Deposit:    300,
		Challenged: true,
	}

	effective, err := profile.EffectiveBalance()
	require.NoError(t, err)
	require.Zero(t, effective)
}
