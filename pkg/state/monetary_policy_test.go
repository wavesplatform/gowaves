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
	const (
		blockRewardActivationHeight = 0
		isCappedRewardsActivated    = false
	)
	mo, storage := createTestObjects(t, settings.MustMainNetSettings())

	h := proto.Height(99001)
	storage.addBlock(t, blockID0)
	err := mo.vote(700000000, h, blockRewardActivationHeight, isCappedRewardsActivated, blockID0)
	require.NoError(t, err)
	votes, err := mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)

	h++
	storage.addBlock(t, blockID1)
	err = mo.vote(500000000, h, blockRewardActivationHeight, isCappedRewardsActivated, blockID1)
	require.NoError(t, err)
	votes, err = mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(1), votes.decrease)
	storage.flush(t)
	votes, err = mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(1), votes.decrease)
}

func TestRollbackVote(t *testing.T) {
	const (
		blockRewardActivationHeight = 0
		isCappedRewardsActivated    = false
	)
	mo, storage := createTestObjects(t, settings.MustMainNetSettings())
	h := proto.Height(99001)
	storage.addBlock(t, blockID0)
	err := mo.vote(700000000, h, blockRewardActivationHeight, isCappedRewardsActivated, blockID0)
	require.NoError(t, err)
	votes, err := mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.newestVotes(h, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)

	storage.rollbackBlock(t, blockID0)
	votes, err = mo.newestVotes(h-1, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.newestVotes(h-1, blockRewardActivationHeight, isCappedRewardsActivated)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
}

func TestFinishRewardVoting(t *testing.T) {
	sets := settings.MustMainNetSettings()
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
		votes, err := mo.newestVotes(h, blockRewardActivationHeight, step.isCappedRewardsActivated)
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
			err = mo.updateBlockReward(id, h, blockRewardActivationHeight, step.isCappedRewardsActivated)
			require.NoError(t, err)
		}
	}
}

func TestRewardAtHeight(t *testing.T) {
	sets := settings.MustMainNetSettings()
	mo, storage := createTestObjects(t, sets)

	const (
		blockRewardActivationHeight = uint64(1)
		initialReward               = uint64(600000000)
		rewardIncrement             = uint64(100000000)
	)

	rewardsChanges := []struct {
		height    proto.Height
		newReward uint64
	}{
		{5, initialReward + rewardIncrement},
		{10, initialReward + 2*rewardIncrement},
		{15, initialReward + 3*rewardIncrement},
		{20, initialReward + 2*rewardIncrement},
	}
	ids := genRandBlockIds(t, len(rewardsChanges))
	for i, rewardChange := range rewardsChanges {
		storage.addBlock(t, ids[i])
		err := mo.saveNewRewardChange(rewardChange.newReward, rewardChange.height, ids[i])
		require.NoError(t, err)
	}

	tests := []struct {
		height         proto.Height
		expectedReward uint64
	}{
		{4, initialReward},
		{8, initialReward + rewardIncrement},
		{12, initialReward + 2*rewardIncrement},
		{15, initialReward + 3*rewardIncrement},
		{21, initialReward + 2*rewardIncrement},
	}

	for _, test := range tests {
		reward, err := mo.rewardAtHeight(test.height, blockRewardActivationHeight)
		require.NoError(t, err)
		assert.Equal(t, test.expectedReward, reward)
	}
}

func TestTotalWavesAmountAtHeightWithRewardsAtGenesis(t *testing.T) {
	sets := settings.MustMainNetSettings()
	mo, storage := createTestObjects(t, sets)

	const (
		blockRewardActivationHeight = uint64(1)
		initialReward               = uint64(600000000)
		initialAmount               = uint64(1000000000)
		rewardIncrement             = uint64(100000000)
	)

	rewardsChanges := []struct {
		height    proto.Height
		newReward uint64
	}{
		{5, initialReward + rewardIncrement},
		{10, initialReward + 2*rewardIncrement},
		{15, initialReward + 3*rewardIncrement},
		{20, initialReward + 2*rewardIncrement},
	}
	ids := genRandBlockIds(t, len(rewardsChanges))
	for i, rewardChange := range rewardsChanges {
		storage.addBlock(t, ids[i])
		err := mo.saveNewRewardChange(rewardChange.newReward, rewardChange.height, ids[i])
		require.NoError(t, err)
	}

	for _, test := range []struct {
		height              proto.Height
		expectedTotalAmount uint64
	}{
		{1, initialAmount},
		{2, initialAmount + initialReward},
		{4, initialAmount + initialReward*3},
		{5, initialAmount + initialReward*3 + initialReward + rewardIncrement},
		{8, initialAmount + initialReward*3 + (initialReward+rewardIncrement)*4},
		{12, initialAmount +
			initialReward*3 +
			(initialReward+rewardIncrement)*5 +
			(initialReward+2*rewardIncrement)*3,
		},
		{15, initialAmount +
			initialReward*3 +
			(initialReward+rewardIncrement)*5 +
			(initialReward+2*rewardIncrement)*5 +
			(initialReward + 3*rewardIncrement),
		},
		{21, initialAmount +
			initialReward*3 +
			(initialReward+rewardIncrement)*5 +
			(initialReward+2*rewardIncrement)*5 +
			(initialReward+3*rewardIncrement)*5 +
			(initialReward+2*rewardIncrement)*2,
		},
	} {
		reward, err := mo.totalAmountAtHeight(test.height, initialAmount, blockRewardActivationHeight, 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int(test.expectedTotalAmount), int(reward), "Error at height %d", test.height)
	}
}

func TestTotalWavesAmountAtHeight(t *testing.T) {
	sets := settings.MustMainNetSettings()
	mo, storage := createTestObjects(t, sets)

	const (
		blockRewardActivationHeight = uint64(10)
		initialReward               = uint64(600000000)
		initialAmount               = uint64(1000000000)
		rewardIncrement             = uint64(100000000)
	)

	rewardsChanges := []struct {
		height    proto.Height
		newReward uint64
	}{
		{15, initialReward + rewardIncrement},
		{20, initialReward + 2*rewardIncrement},
		{25, initialReward + 3*rewardIncrement},
		{30, initialReward + 2*rewardIncrement},
	}
	ids := genRandBlockIds(t, len(rewardsChanges))
	for i, rewardChange := range rewardsChanges {
		storage.addBlock(t, ids[i])
		err := mo.saveNewRewardChange(rewardChange.newReward, rewardChange.height, ids[i])
		require.NoError(t, err)
	}

	for _, test := range []struct {
		height              proto.Height
		expectedTotalAmount uint64
	}{
		{2, initialAmount},
		{4, initialAmount},
		{9, initialAmount},
		{10, initialAmount + initialReward},
		{14, initialAmount + 5*initialReward},
		{15, initialAmount + 5*initialReward + initialReward + rewardIncrement},
		{19, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement)},
		{20, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			initialReward + 2*rewardIncrement,
		},
		{24, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			5*(initialReward+2*rewardIncrement),
		},
		{25, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			5*(initialReward+2*rewardIncrement) + initialReward + 3*rewardIncrement,
		},
		{29, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			5*(initialReward+2*rewardIncrement) + 5*(initialReward+3*rewardIncrement),
		},
		{30, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			5*(initialReward+2*rewardIncrement) + 5*(initialReward+3*rewardIncrement) +
			initialReward + 2*rewardIncrement,
		},
		{33, initialAmount + 5*initialReward + 5*(initialReward+rewardIncrement) +
			5*(initialReward+2*rewardIncrement) + 5*(initialReward+3*rewardIncrement) +
			4*(initialReward+2*rewardIncrement),
		},
	} {
		reward, err := mo.totalAmountAtHeight(test.height, initialAmount, blockRewardActivationHeight, 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int(test.expectedTotalAmount), int(reward), "Error at height %d", test.height)
	}
}

func TestBoost(t *testing.T) {
	for i, test := range []struct {
		first, last          uint64
		reward               uint64
		changeHeight, height uint64
		expected             uint64
	}{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 6, 10, 13, 24},

		{8, 17, 6, 15, 19, 2*6 + 3*6*10},
		{8, 17, 5, 10, 14, 5 * 5 * 10},
		{8, 17, 4, 5, 8, 1*10*4 + 3*4},
	} {
		b := boostedReward{first: test.first, last: test.last}
		reward := b.reward(test.reward, test.changeHeight, test.height)
		assert.Equal(t, int(test.expected), int(reward), i+1)
		assert.Equal(t, int(test.expected), int(reward), i+1)
	}
}

func TestBoostedTotalWavesAmountAtHeight(t *testing.T) {
	sets := settings.MustMainNetSettings()
	mo, storage := createTestObjects(t, sets)

	const (
		blockRewardActivationHeight = uint64(10)
		initialReward               = uint64(600000000)
		initialAmount               = uint64(1000000000)
		rewardIncrement             = uint64(100000000)
	)

	rewardsChanges := []struct {
		height    proto.Height
		newReward uint64
	}{
		{15, initialReward + rewardIncrement},
		{20, initialReward + 2*rewardIncrement},
		{25, initialReward + 3*rewardIncrement},
		{30, initialReward + 2*rewardIncrement},
	}
	ids := genRandBlockIds(t, len(rewardsChanges))
	for i, rewardChange := range rewardsChanges {
		storage.addBlock(t, ids[i])
		err := mo.saveNewRewardChange(rewardChange.newReward, rewardChange.height, ids[i])
		require.NoError(t, err)
	}

	for _, test := range []struct {
		height              proto.Height
		expectedTotalAmount uint64
	}{
		{2, initialAmount},
		{4, initialAmount},
		{9, initialAmount},
		{10, initialAmount + initialReward},
		{14, initialAmount + 5*initialReward},
		{15, initialAmount + 5*initialReward + initialReward + rewardIncrement},
		{19, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement)},
		{20, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*(initialReward+2*rewardIncrement),
		},
		{24, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*5*(initialReward+2*rewardIncrement),
		},
		{25, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*5*(initialReward+2*rewardIncrement) +
			10*(initialReward+3*rewardIncrement),
		},
		{29, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*5*(initialReward+2*rewardIncrement) +
			3*10*(initialReward+3*rewardIncrement) + 2*(initialReward+3*rewardIncrement),
		},
		{30, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*5*(initialReward+2*rewardIncrement) +
			3*10*(initialReward+3*rewardIncrement) + 2*(initialReward+3*rewardIncrement) +
			initialReward + 2*rewardIncrement,
		},
		{33, initialAmount + 5*initialReward + 3*(initialReward+rewardIncrement) +
			2*10*(initialReward+rewardIncrement) + 10*5*(initialReward+2*rewardIncrement) +
			3*10*(initialReward+3*rewardIncrement) + 2*(initialReward+3*rewardIncrement) +
			4*(initialReward+2*rewardIncrement),
		},
	} {
		reward, err := mo.totalAmountAtHeight(test.height, initialAmount, blockRewardActivationHeight, 18, 27)
		require.NoError(t, err)
		assert.Equal(t, int(test.expectedTotalAmount), int(reward), "Error at height %d", test.height)
	}
}

func createTestObjects(t *testing.T, sets *settings.BlockchainSettings) (*monetaryPolicy, *testStorageObjects) {
	storage := createStorageObjects(t, true)
	mp := newMonetaryPolicy(storage.hs, sets)
	return mp, storage
}
