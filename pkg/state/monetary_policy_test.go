package state

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func TestBlockRewardRecord(t *testing.T) {
	for _, test := range []uint64{
		0,
		1,
		1234567890,
		math.MaxUint64,
	} {
		r1 := blockRewardRecord{reward: test}
		b, err := r1.marshalBinary()
		require.NoError(t, err)
		var r2 blockRewardRecord
		err = r2.unmarshalBinary(b)
		require.NoError(t, err)
		assert.Equal(t, r1, r2)
		assert.Equal(t, test, r2.reward)
	}
}

func TestRewardVotesRecord(t *testing.T) {
	for _, test := range []struct {
		dec uint32
		inc uint32
	}{
		{0, 0},
		{0, 1},
		{1, 1},
		{12345, 167890},
		{math.MaxUint32, math.MaxUint32},
	} {
		r1 := rewardVotesRecord{decrease: test.dec, increase: test.inc}
		b, err := r1.marshalBinary()
		require.NoError(t, err)
		var r2 rewardVotesRecord
		err = r2.unmarshalBinary(b)
		require.NoError(t, err)
		assert.Equal(t, r1, r2)
		assert.Equal(t, test.dec, r2.decrease)
		assert.Equal(t, test.inc, r2.increase)
	}
}

func TestAddVote(t *testing.T) {
	mo, storage := createTestObjects(t, settings.MainNetSettings)

	storage.addBlock(t, blockID0)
	err := mo.vote(700000000, 99001, 0, false, blockID0)
	require.NoError(t, err)
	votes, err := mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)

	storage.addBlock(t, blockID1)
	err = mo.vote(500000000, 99002, 0, false, blockID1)
	require.NoError(t, err)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(1), votes.decrease)
	storage.flush(t)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(1), votes.decrease)
}

func TestRollbackVote(t *testing.T) {
	mo, storage := createTestObjects(t, settings.MainNetSettings)

	storage.addBlock(t, blockID0)
	err := mo.vote(700000000, 99001, 0, false, blockID0)
	require.NoError(t, err)
	votes, err := mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)

	storage.rollbackBlock(t, blockID0)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(0), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
}

func TestFinishRewardVoting(t *testing.T) {
	sets := settings.MainNetSettings
	sets.FunctionalitySettings.BlockRewardTerm = 8
	sets.FunctionalitySettings.BlockRewardTermAfter20 = 4
	sets.FunctionalitySettings.BlockRewardVotingPeriod = 2
	mo, storage := createTestObjects(t, sets)

	const (
		initial = 600000000
		up      = 700000000
		down    = 500000000
	)
	tests := []struct {
		vote                     int64
		increase                 uint32
		decrease                 uint32
		reward                   uint64
		isCappedRewardsActivated bool
	}{
		//10 start of term
		{up, 0, 0, initial, false},              //11
		{up, 0, 0, initial, false},              //12
		{down, 0, 0, initial, false},            //13
		{down, 0, 0, initial, false},            //14
		{down, 0, 0, initial, false},            //15
		{up, 1, 0, initial, false},              //16
		{up, 2, 0, initial, false},              //17 end of term
		{down, 0, 0, initial + 50000000, false}, //18 start of term
		{up, 0, 0, initial + 50000000, false},   //20
		{down, 0, 0, initial + 50000000, false}, //21
		{down, 0, 0, initial + 50000000, false}, //22
		{up, 0, 0, initial + 50000000, false},   //23
		{down, 0, 0, initial + 50000000, false}, //24
		{down, 0, 1, initial + 50000000, false}, //25
		{down, 0, 2, initial + 50000000, false}, //26 end of term
		{up, 0, 0, initial, false},              //27 start of term
		{down, 0, 0, initial, false},            //28
		{up, 1, 0, initial, true},               //29
		{up, 2, 0, initial, true},               //30 end of term
		{down, 0, 0, initial + 50000000, true},  //31 start of term
	}
	ids := genRandBlockIds(t, len(tests)+1)
	const (
		blockRewardActivationHeight = 10
		initialHeight               = 11
	)
	for i, step := range tests {
		var (
			h   = proto.Height(initialHeight + i)
			id  = ids[i]
			msg = fmt.Sprintf("height %d", h)
		)
		storage.addBlock(t, id)
		err := mo.vote(step.vote, h, blockRewardActivationHeight, step.isCappedRewardsActivated, id)
		require.NoError(t, err, msg)
		votes, err := mo.votes()
		require.NoError(t, err, msg)
		assert.Equal(t, step.increase, votes.increase, "increase: "+msg)
		assert.Equal(t, step.decrease, votes.decrease, "decrease: "+msg)
		storage.flush(t)
		reward, err := mo.reward()
		require.NoError(t, err, msg)
		assert.Equal(t, step.reward, reward, fmt.Sprintf("unexpected reward %d: %s", reward, msg))
		_, end := mo.blockRewardVotingPeriod(h, blockRewardActivationHeight, step.isCappedRewardsActivated)
		if h == end {
			nextID := ids[i+1]
			storage.prepareBlock(t, nextID)
			err = mo.updateBlockReward(id, nextID)
			require.NoError(t, err)
		}
	}
}

func createTestObjects(t *testing.T, sets *settings.BlockchainSettings) (*monetaryPolicy, *testStorageObjects) {
	storage := createStorageObjects(t, true)
	mp := newMonetaryPolicy(storage.hs, sets)
	return mp, storage
}
