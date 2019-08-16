package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	featureID = 1
)

type featuresTestObjects struct {
	stor     *testStorageObjects
	features *features
}

func createFeatures(sets *settings.BlockchainSettings) (*featuresTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	definedFeaturesInfo := make(map[settings.Feature]settings.FeatureInfo)
	definedFeaturesInfo[settings.Feature(featureID)] = settings.FeatureInfo{Implemented: true, Description: "test feature"}
	features, err := newFeatures(stor.db, stor.dbBatch, stor.hs, stor.stateDB, sets, definedFeaturesInfo)
	if err != nil {
		return nil, path, err
	}
	return &featuresTestObjects{stor, features}, path, nil
}

func TestAddFeatureVote(t *testing.T) {
	to, path, err := createFeatures(settings.MainNetSettings)
	assert.NoError(t, err, "createFeatures() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	err = to.features.addVote(featureID, blockID0)
	assert.NoError(t, err, "addVote() failed")
	votes, err := to.features.featureVotes(featureID)
	assert.NoError(t, err, "featureVotes() failed")
	assert.Equal(t, uint64(1), votes)
	votes, err = to.features.featureVotes(0)
	assert.NoError(t, err, "featureVotes() failed")
	assert.Equal(t, uint64(0), votes)
	to.stor.flush(t)
	votes, err = to.features.featureVotes(featureID)
	assert.NoError(t, err, "featureVotes() after flush() failed")
	assert.Equal(t, uint64(1), votes)
}

func TestApproveFeature(t *testing.T) {
	to, path, err := createFeatures(settings.MainNetSettings)
	assert.NoError(t, err, "createFeatures() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	approved, err := to.features.isApproved(featureID)
	assert.NoError(t, err, "isApproved failed")
	assert.Equal(t, false, approved)
	to.stor.addBlock(t, blockID0)
	blockNum, err := to.stor.stateDB.blockIdToNum(blockID0)
	assert.NoError(t, err, "blockIdToNum() failed")
	r := &approvedFeaturesRecord{1, blockNum}
	err = to.features.approveFeature(featureID, r)
	assert.NoError(t, err, "approveFeature() failed")
	to.stor.flush(t)
	approved, err = to.features.isApproved(featureID)
	assert.NoError(t, err, "isApproved failed")
	assert.Equal(t, true, approved)
	approvalHeight, err := to.features.approvalHeight(featureID)
	assert.NoError(t, err, "approvalHeight() failed")
	assert.Equal(t, uint64(1), approvalHeight)
}

func TestActivateFeature(t *testing.T) {
	to, path, err := createFeatures(settings.MainNetSettings)
	assert.NoError(t, err, "createFeatures() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	activated, err := to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, false, activated)
	blockNum, err := to.stor.stateDB.blockIdToNum(blockID0)
	assert.NoError(t, err, "blockIdToNum() failed")
	r := &activatedFeaturesRecord{1, blockNum}
	err = to.features.activateFeature(featureID, r)
	assert.NoError(t, err, "activateFeature() failed")
	to.stor.flush(t)
	activated, err = to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, true, activated)
	activationHeight, err := to.features.activationHeight(featureID)
	assert.NoError(t, err, "activationHeight() failed")
	assert.Equal(t, uint64(1), activationHeight)
}

func TestFinishVoting(t *testing.T) {
	settings := settings.MainNetSettings
	to, path, err := createFeatures(settings)
	assert.NoError(t, err, "createFeatures() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	height := settings.ActivationWindowSize(1)
	ids := genRandBlockIds(t, int(height*2))
	for _, id := range ids {
		to.stor.addBlock(t, id)
	}
	tests := []struct {
		curHeight        uint64
		votesNum         uint64
		isApproved       bool
		isActivated      bool
		approvalHeight   uint64
		activationHeight uint64
	}{
		{height, settings.VotesForFeatureElection(1) - 1, false, false, 0, 0},
		{height, settings.VotesForFeatureElection(1), true, false, height, 0},
		{height * 2, 0, true, true, height, height * 2},
	}
	for _, tc := range tests {
		// Add required amount of votes first.
		for i := uint64(0); i < tc.votesNum; i++ {
			err = to.features.addVote(featureID, ids[i])
			assert.NoError(t, err, "addVote() failed")
		}
		// Flush votes.
		to.stor.flush(t)
		// Call finishVoting().
		lastId := ids[tc.curHeight-1]
		err = to.features.finishVoting(tc.curHeight, lastId)
		assert.NoError(t, err, "finishVoting() failed")
		// Flush updates.
		to.stor.flush(t)
		// Check approval and activation.
		isApproved, err := to.features.isApproved(featureID)
		assert.NoError(t, err, "isApproved() failed")
		assert.Equal(t, tc.isApproved, isApproved)
		isActivated, err := to.features.isActivated(featureID)
		assert.NoError(t, err, "isActivated() failed")
		assert.Equal(t, tc.isActivated, isActivated)
		approvalHeight, err := to.features.approvalHeight(featureID)
		if tc.isApproved {
			assert.NoError(t, err, "approvalHeight() failed")
			assert.Equal(t, height, approvalHeight)
		} else {
			assert.Error(t, err, "approvalHeight() did not fail with unapproved feature")
		}
		activationHeight, err := to.features.activationHeight(featureID)
		if tc.isActivated {
			assert.NoError(t, err, "activationHeight() failed")
			assert.Equal(t, height*2, activationHeight)
		} else {
			assert.Error(t, err, "activationHeight() did not fail with not activated feature")
		}
	}
}
