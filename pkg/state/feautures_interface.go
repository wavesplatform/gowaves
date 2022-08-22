package state

import "github.com/wavesplatform/gowaves/pkg/proto"

//go:generate moq -out feautures_moq_test.go . featuresState:mockFeaturesState
type featuresState interface {
	newestIsActivated(featureID int16) (bool, error)
	approveFeature(featureID int16, r *approvedFeaturesRecord, blockID proto.BlockID) error
	activateFeature(featureID int16, r *activatedFeaturesRecord, blockID proto.BlockID) error
	newestIsActivatedAtHeight(featureID int16, height uint64) bool
	newestIsApproved(featureID int16) (bool, error)
	addVote(featureID int16, blockID proto.BlockID) error
	newestActivationHeight(featureID int16) (uint64, error)
	newestApprovalHeight(featureID int16) (uint64, error)
	resetVotes(blockID proto.BlockID) error
	finishVoting(curHeight uint64, blockID proto.BlockID) error
	isActivatedAtHeight(featureID int16, height uint64) bool
	activationHeight(featureID int16) (uint64, error)
	isApproved(featureID int16) (bool, error)
	isApprovedAtHeight(featureID int16, height uint64) bool
	approvalHeight(featureID int16) (uint64, error)
	allFeatures() ([]int16, error)
	newestIsActivatedForNBlocks(featureID int16, n int) (bool, error)
	featureVotes(featureID int16) (uint64, error)
	featureVotesAtHeight(featureID int16, height uint64) (uint64, error)
	clearCache()
}
