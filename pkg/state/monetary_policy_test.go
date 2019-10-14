package state

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
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
	mo, storage, path, err := createTestObjects(settings.MainNetSettings)
	require.NoError(t, err)
	defer func() {
		storage.close(t)
		err = util.CleanTemporaryDirs(path)
		require.NoError(t, err)
	}()

	storage.addBlock(t, blockID0)
	err = mo.addVote(700000000, blockID0)
	require.NoError(t, err)
	votes, err := mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)
	assert.Equal(t, uint32(0), votes.decrease)
	storage.flush(t)
	votes, err = mo.votes()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), votes.increase)

	storage.addBlock(t, blockID1)
	err = mo.addVote(500000000, blockID1)
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

func createTestObjects(sets *settings.BlockchainSettings) (*monetaryPolicy, *testStorageObjects, []string, error) {
	storage, path, err := createStorageObjects()
	if err != nil {
		return nil, nil, path, err
	}
	mp, err := newMonetaryPolicy(storage.db, storage.dbBatch, storage.hs, sets)
	if err != nil {
		return nil, storage, path, err
	}
	return mp, storage, path, nil
}
