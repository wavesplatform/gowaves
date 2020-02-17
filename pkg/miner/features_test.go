package miner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVoteFeaturesSuccess(t *testing.T) {
	s := "13,14"
	rs, err := ParseVoteFeatures(s)
	require.NoError(t, err)

	require.Equal(t, Features{
		13,
		14,
	}, rs)

	require.Equal(t, []int16{13, 14}, FeaturesToInt16(rs))

	rs, err = ParseVoteFeatures("")
	require.NoError(t, err)
	require.Equal(t, Features{}, rs)

}

func TestParseVoteFeaturesFailure(t *testing.T) {
	s2 := "abc"
	_, err := ParseVoteFeatures(s2)
	require.Error(t, err)
}

func TestParseReward(t *testing.T) {
	rs, err := ParseReward("")
	require.NoError(t, err)
	require.EqualValues(t, 0, rs)

	rs, err = ParseReward("100500")
	require.NoError(t, err)
	require.EqualValues(t, 100500, rs)
}
