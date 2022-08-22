package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	featureID  = 1
	featureID1 = 2
	featureID2 = 3
)

type featuresTestObjects struct {
	stor     *testStorageObjects
	features *features
}

func createFeatures(t *testing.T, sets *settings.BlockchainSettings) *featuresTestObjects {
	stor := createStorageObjects(t, true)
	definedFeaturesInfo := make(map[settings.Feature]settings.FeatureInfo)
	definedFeaturesInfo[settings.Feature(featureID)] = settings.FeatureInfo{Implemented: true, Description: "test feature"}
	features, ok := stor.entities.features.(*features)
	require.True(t, ok)
	return &featuresTestObjects{stor, features}
}

func TestAddFeatureVote(t *testing.T) {
	to := createFeatures(t, settings.MainNetSettings)

	to.stor.addBlock(t, blockID0)
	err := to.features.addVote(featureID, blockID0)
	assert.NoError(t, err, "addVote() failed")
	votes, err := to.features.newestFeatureVotes(featureID)
	assert.NoError(t, err, "newestFeatureVotes() failed")
	assert.Equal(t, uint64(1), votes)
	votes, err = to.features.newestFeatureVotes(0)
	assert.NoError(t, err, "newestFeatureVotes() failed")
	assert.Equal(t, uint64(0), votes)
	to.stor.flush(t)
	votes, err = to.features.newestFeatureVotes(featureID)
	assert.NoError(t, err, "newestFeatureVotes() after flush() failed")
	assert.Equal(t, uint64(1), votes)
}

func TestApproveFeature(t *testing.T) {
	to := createFeatures(t, settings.MainNetSettings)

	approved, err := to.features.isApproved(featureID)
	assert.NoError(t, err, "isApproved failed")
	assert.Equal(t, false, approved)
	to.stor.addBlock(t, blockID0)
	r := &approvedFeaturesRecord{1}
	err = to.features.approveFeature(featureID, r, blockID0)
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
	to := createFeatures(t, settings.MainNetSettings)

	to.stor.addBlock(t, blockID0)
	activated, err := to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, false, activated)
	r := &activatedFeaturesRecord{1}
	err = to.features.activateFeature(featureID, r, blockID0)
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
	sets := settings.MainNetSettings
	to := createFeatures(t, sets)

	height := sets.ActivationWindowSize(1)
	ids := genRandBlockIds(t, int(height*3))
	tests := []struct {
		curHeight        uint64
		votesNum         uint64
		isApproved       bool
		isActivated      bool
		approvalHeight   uint64
		activationHeight uint64
	}{
		{height, sets.VotesForFeatureElection(1) - 1, false, false, 0, 0},
		{height * 2, sets.VotesForFeatureElection(1), true, false, height * 2, 0},
		{height * 3, 0, true, true, height * 2, height * 3},
	}
	heightCounter := uint64(0)
	for _, tc := range tests {
		// Reset votes as we have started next period.
		nextBlockId := ids[heightCounter]
		to.stor.addBlock(t, nextBlockId)
		err := to.features.resetVotes(nextBlockId)
		assert.NoError(t, err, "resetVotes() failed")
		// Add required amount of votes first.
		for i := uint64(0); i < tc.votesNum; i++ {
			to.stor.addBlock(t, ids[heightCounter])
			err = to.features.addVote(featureID, ids[heightCounter])
			assert.NoError(t, err, "addVote() failed")
			heightCounter++
		}
		var lastBlockId proto.BlockID
		// Add remaining blocks until curHeight.
		for ; heightCounter < tc.curHeight; heightCounter++ {
			to.stor.addBlock(t, ids[heightCounter])
			lastBlockId = ids[heightCounter]
		}
		// Flush votes.
		to.stor.flush(t)
		// Call finishVoting().
		err = to.features.finishVoting(tc.curHeight, lastBlockId)
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
			assert.Equal(t, tc.approvalHeight, approvalHeight)
		} else {
			assert.Error(t, err, "approvalHeight() did not fail with unapproved feature")
		}
		activationHeight, err := to.features.activationHeight(featureID)
		if tc.isActivated {
			assert.NoError(t, err, "activationHeight() failed")
			assert.Equal(t, tc.activationHeight, activationHeight)
		} else {
			assert.Error(t, err, "activationHeight() did not fail with not activated feature")
		}
	}
	// Check votes at height.
	for _, tc := range tests {
		votesNum, err := to.features.featureVotesAtHeight(featureID, tc.curHeight)
		assert.NoError(t, err)
		assert.Equal(t, tc.votesNum, votesNum)
	}
}

func TestAllFeatures(t *testing.T) {
	to := createFeatures(t, settings.MainNetSettings)

	to.stor.addBlock(t, blockID0)
	err := to.features.addVote(featureID1, blockID0)
	assert.NoError(t, err, "addVote() failed")
	err = to.features.addVote(featureID2, blockID0)
	assert.NoError(t, err, "addVote() failed")
	to.stor.flush(t)
	features, err := to.features.allFeatures()
	assert.NoError(t, err, "allFeatures() failed")
	assert.Equal(t, 2, len(features))
	assert.Equal(t, int16(featureID1), features[0])
	assert.Equal(t, int16(featureID2), features[1])
}

func TestRollbackActivation(t *testing.T) {
	to := createFeatures(t, settings.MainNetSettings)

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	to.stor.flush(t)

	activated, err := to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, false, activated)

	r := &activatedFeaturesRecord{2}
	err = to.features.activateFeature(featureID, r, blockID1)
	assert.NoError(t, err, "activateFeature() failed")
	to.stor.flush(t)

	activated, err = to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, true, activated)
	activationHeight, err := to.features.activationHeight(featureID)
	assert.NoError(t, err, "activationHeight() failed")
	assert.Equal(t, uint64(2), activationHeight)

	to.stor.fullRollbackBlockClearCache(t, blockID0)

	activated, err = to.features.isActivated(featureID)
	assert.NoError(t, err, "isActivated failed")
	assert.Equal(t, false, activated)
}
