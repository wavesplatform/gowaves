package state

import (
	"encoding/binary"
	"errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"go.uber.org/zap"
)

const (
	activatedFeaturesRecordSize = 8
	approvedFeaturesRecordSize  = 8
	votesFeaturesRecordSize     = 8
)

type activatedFeaturesRecord struct {
	activationHeight uint64
}

func (r *activatedFeaturesRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, activatedFeaturesRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.activationHeight)
	return res, nil
}

func (r *activatedFeaturesRecord) unmarshalBinary(data []byte) error {
	if len(data) != activatedFeaturesRecordSize {
		return errInvalidDataSize
	}
	r.activationHeight = binary.BigEndian.Uint64(data[:8])
	return nil
}

type approvedFeaturesRecord struct {
	approvalHeight uint64
}

func (r *approvedFeaturesRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, approvedFeaturesRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.approvalHeight)
	return res, nil
}

func (r *approvedFeaturesRecord) unmarshalBinary(data []byte) error {
	if len(data) != approvedFeaturesRecordSize {
		return errInvalidDataSize
	}
	r.approvalHeight = binary.BigEndian.Uint64(data[:8])
	return nil
}

type votesFeaturesRecord struct {
	votesNum uint64
}

func (r *votesFeaturesRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, votesFeaturesRecordSize)
	binary.BigEndian.PutUint64(res[:8], r.votesNum)
	return res, nil
}

func (r *votesFeaturesRecord) unmarshalBinary(data []byte) error {
	if len(data) != votesFeaturesRecordSize {
		return errInvalidDataSize
	}
	r.votesNum = binary.BigEndian.Uint64(data[:8])
	return nil
}

type featureActivationState struct {
	activated bool
	height    uint64
}

type features struct {
	rw                  *blockReadWriter
	db                  keyvalue.IterableKeyVal
	hs                  *historyStorage
	settings            *settings.BlockchainSettings
	definedFeaturesInfo map[settings.Feature]settings.FeatureInfo
	activationCache     map[settings.Feature]featureActivationState
}

func newFeatures(rw *blockReadWriter, db keyvalue.IterableKeyVal, hs *historyStorage, stg *settings.BlockchainSettings,
	definedFeaturesInfo map[settings.Feature]settings.FeatureInfo) *features {
	return &features{
		rw:                  rw,
		db:                  db,
		hs:                  hs,
		settings:            stg,
		definedFeaturesInfo: definedFeaturesInfo,
		activationCache:     make(map[settings.Feature]featureActivationState),
	}
}

// addVote adds vote for feature by its featureID at given blockID.
func (f *features) addVote(featureID int16, blockID proto.BlockID) error {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return err
	}
	prevVotes, err := f.newestFeatureVotes(featureID)
	if err != nil {
		return err
	}
	record := &votesFeaturesRecord{prevVotes + 1}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return f.hs.addNewEntry(featureVote, keyBytes, recordBytes, blockID)
}

func (f *features) votesFromRecord(recordBytes []byte) (uint64, error) {
	var record votesFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.votesNum, nil
}

func (f *features) featureVotesAtHeight(featureID int16, height uint64) (uint64, error) {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.entryDataAtHeight(keyBytes, height)
	if err == keyvalue.ErrNotFound || err == errEmptyHist || recordBytes == nil {
		// 0 votes for unknown feature.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return f.votesFromRecord(recordBytes)
}

func (f *features) featureVotes(featureID int16) (uint64, error) {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.topEntryData(keyBytes)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// 0 votes for unknown feature.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return f.votesFromRecord(recordBytes)
}

func (f *features) newestFeatureVotes(featureID int16) (uint64, error) {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.newestTopEntryData(keyBytes)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// 0 votes for unknown feature.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return f.votesFromRecord(recordBytes)
}

func (f *features) printActivationLog(featureID int16, height uint64) {
	info, ok := f.definedFeaturesInfo[settings.Feature(featureID)]
	if ok {
		zap.S().Infof("Activating feature %d (%s) at height %d", featureID, info.Description, height)
	} else {
		zap.S().Warnf("Activating UNKNOWN feature %d at height %d", featureID, height)
	}
	if !ok || !info.Implemented {
		zap.S().Warn("FATAL: UNKNOWN/UNIMPLEMENTED feature has been activated on the blockchain!")
		zap.S().Warn("FOR THIS REASON THE NODE IS STOPPED AUTOMATICALLY.")
		zap.S().Fatalf("PLEASE, UPDATE THE NODE IMMEDIATELY!")
	}
}

func (f *features) activateFeature(featureID int16, r *activatedFeaturesRecord, blockID proto.BlockID) error {
	f.clearCache()
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return err
	}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	f.printActivationLog(featureID, r.activationHeight)
	return f.hs.addNewEntry(activatedFeature, keyBytes, recordBytes, blockID)
}

func (f *features) newestIsActivatedForNBlocks(featureID int16, n int) (bool, error) {
	activationHeight, err := f.newestActivationHeight(featureID)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
			return false, nil
		}
		return false, err
	}
	curBlockHeight := f.rw.addingBlockHeight()
	if curBlockHeight < uint64(n) {
		return false, nil
	}
	return curBlockHeight-uint64(n) >= activationHeight, nil
}

func (f *features) newestIsActivated(featureID int16) (bool, error) {
	if as, ok := f.activationCache[settings.Feature(featureID)]; ok {
		return as.activated, nil
	}
	r, err := f.newestActivatedFeaturesRecord(featureID)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
			f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: false}
			return false, nil
		}
		return false, err
	}
	f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: true, height: r.activationHeight}
	return true, nil
}

func (f *features) isActivated(featureID int16) (bool, error) {
	if as, ok := f.activationCache[settings.Feature(featureID)]; ok {
		return as.activated, nil
	}
	r, err := f.activatedFeaturesRecord(featureID)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) || errors.Is(err, errEmptyHist) {
			f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: false}
			return false, nil
		}
		return false, err
	}
	f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: true, height: r.activationHeight}
	return true, nil
}

func (f *features) newestIsActivatedAtHeight(featureID int16, height uint64) bool {
	activationHeight, err := f.newestActivationHeight(featureID)
	if err == nil {
		return height >= activationHeight
	}
	approvalHeight, err := f.newestApprovalHeight(featureID)
	if err == nil && height >= approvalHeight {
		return (height - approvalHeight) >= f.settings.ActivationWindowSize(height)
	}
	return false
}

func (f *features) isActivatedAtHeight(featureID int16, height uint64) bool {
	activationHeight, err := f.activationHeight(featureID)
	if err == nil {
		return height >= activationHeight
	}
	approvalHeight, err := f.approvalHeight(featureID)
	if err == nil && height >= approvalHeight {
		return (height - approvalHeight) >= f.settings.ActivationWindowSize(height)
	}
	return false
}

func (f *features) newestActivatedFeaturesRecord(featureID int16) (*activatedFeaturesRecord, error) {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return nil, err
	}
	recordBytes, err := f.hs.newestTopEntryData(keyBytes)
	if err != nil {
		return nil, err
	}
	var record activatedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (f *features) activatedFeaturesRecord(featureID int16) (*activatedFeaturesRecord, error) {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return nil, err
	}
	recordBytes, err := f.hs.topEntryData(keyBytes)
	if err != nil {
		return nil, err
	}
	var record activatedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (f *features) newestActivationHeight(featureID int16) (uint64, error) {
	if as, ok := f.activationCache[settings.Feature(featureID)]; ok {
		if as.activated {
			return as.height, nil
		}
		return 0, keyvalue.ErrNotFound
	}
	record, err := f.newestActivatedFeaturesRecord(featureID)
	if err != nil {
		return 0, err
	}
	f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: true, height: record.activationHeight}
	return record.activationHeight, nil
}

func (f *features) activationHeight(featureID int16) (uint64, error) {
	if as, ok := f.activationCache[settings.Feature(featureID)]; ok {
		if as.activated {
			return as.height, nil
		}
		return 0, keyvalue.ErrNotFound
	}
	record, err := f.activatedFeaturesRecord(featureID)
	if err != nil {
		return 0, err
	}
	f.activationCache[settings.Feature(featureID)] = featureActivationState{activated: true, height: record.activationHeight}
	return record.activationHeight, nil
}

func (f *features) printApprovalLog(featureID int16, height uint64) {
	info, ok := f.definedFeaturesInfo[settings.Feature(featureID)]
	if ok {
		zap.S().Infof("Approving feature %d (%s) at height %d", featureID, info.Description, height)
	} else {
		zap.S().Infof("Approving UNKNOWN feature %d at height %d", featureID, height)
	}
	if !ok || !info.Implemented {
		zap.S().Warn("WARNING: UNKNOWN/UNIMPLEMENTED feature has been approved on the blockchain!")
		zap.S().Warn("PLEASE UPDATE THE NODE AS SOON AS POSSIBLE!")
		zap.S().Warn("OTHERWISE THE NODE WILL BE STOPPED OR FORKED UPON FEATURE ACTIVATION.")
	}
}

func (f *features) approveFeature(featureID int16, r *approvedFeaturesRecord, blockID proto.BlockID) error {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return err
	}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	f.printApprovalLog(featureID, r.approvalHeight)
	return f.hs.addNewEntry(approvedFeature, keyBytes, recordBytes, blockID)
}

func (f *features) newestIsApproved(featureID int16) (bool, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return false, err
	}
	_, err = f.hs.newestTopEntryData(keyBytes)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (f *features) isApproved(featureID int16) (bool, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return false, err
	}
	_, err = f.hs.topEntryData(keyBytes)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (f *features) isApprovedAtHeight(featureID int16, height uint64) bool {
	approvalHeight, err := f.approvalHeight(featureID)
	if err == nil && height >= approvalHeight {
		return true
	}
	return false
}

func (f *features) newestApprovalHeight(featureID int16) (uint64, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.newestTopEntryData(keyBytes)
	if err != nil {
		return 0, err
	}
	var record approvedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.approvalHeight, nil
}

func (f *features) approvalHeight(featureID int16) (uint64, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.topEntryData(keyBytes)
	if err != nil {
		return 0, err
	}
	var record approvedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.approvalHeight, nil
}

func (f *features) newestIsElected(height uint64, featureID int16) (bool, error) {
	votes, err := f.newestFeatureVotes(featureID)
	if err != nil {
		return false, err
	}
	return votes >= f.settings.VotesForFeatureElection(height), nil
}

func (f *features) resetVotes(blockID proto.BlockID) error {
	iter, err := f.hs.newNewestTopEntryIterator(featureVote)
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		key := keyvalue.SafeKey(iter)
		// Reset features votes:
		// next voting period starts from scratch.
		newRecord := &votesFeaturesRecord{0}
		newRecordBytes, err := newRecord.marshalBinary()
		if err != nil {
			return err
		}
		if err := f.hs.addNewEntry(featureVote, key, newRecordBytes, blockID); err != nil {
			return err
		}
	}
	return nil
}

// Check voting results, update approval list, reset voting list.
func (f *features) approveFeatures(curHeight uint64, blockID proto.BlockID) error {
	iter, err := f.hs.newNewestTopEntryIterator(featureVote)
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		// Iterate the voting list.
		key := keyvalue.SafeKey(iter)
		var k votesFeaturesKey
		if err = k.unmarshal(key); err != nil {
			return err
		}
		alreadyApproved, err := f.newestIsApproved(k.featureID)
		if err != nil {
			return err
		}
		if alreadyApproved {
			continue
		}
		elected, err := f.newestIsElected(curHeight, k.featureID)
		if err != nil {
			return err
		}
		if elected {
			// Add feature to the list of approved.
			r := &approvedFeaturesRecord{curHeight}
			if err := f.approveFeature(k.featureID, r, blockID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Update activation list.
func (f *features) activateFeatures(curHeight uint64, blockID proto.BlockID) error {
	f.clearCache()
	iter, err := f.hs.newNewestTopEntryIterator(approvedFeature)
	if err != nil {
		return err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	for iter.Next() {
		// Iterate approved features.
		var k approvedFeaturesKey
		if err = k.unmarshal(keyvalue.SafeKey(iter)); err != nil {
			return err
		}
		alreadyActivated, err := f.newestIsActivated(k.featureID)
		if err != nil {
			return err
		}
		if alreadyActivated {
			continue
		}
		approvalHeight, err := f.newestApprovalHeight(k.featureID)
		if err != nil {
			return err
		}
		needToActivate := false
		if curHeight >= approvalHeight {
			needToActivate = (curHeight - approvalHeight) >= f.settings.ActivationWindowSize(curHeight)
		}
		if needToActivate {
			// Add feature to the list of activated.
			r := &activatedFeaturesRecord{curHeight}
			if err := f.activateFeature(k.featureID, r, blockID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *features) finishVoting(curHeight uint64, blockID proto.BlockID) error {
	if err := f.activateFeatures(curHeight, blockID); err != nil {
		return err
	}
	if err := f.approveFeatures(curHeight, blockID); err != nil {
		return err
	}
	return nil
}

func (f *features) allFeatures() ([]int16, error) {
	iter, err := f.hs.newTopEntryIterator(featureVote)
	if err != nil {
		return nil, err
	}
	defer func() {
		iter.Release()
		if err := iter.Error(); err != nil {
			zap.S().Fatalf("Iterator error: %v", err)
		}
	}()

	var list []int16
	for iter.Next() {
		// Iterate the voting list.
		key := keyvalue.SafeKey(iter)
		var k votesFeaturesKey
		if err = k.unmarshal(key); err != nil {
			return nil, err
		}
		list = append(list, k.featureID)
	}
	return list, nil
}

func (f *features) clearCache() {
	f.activationCache = make(map[settings.Feature]featureActivationState)
}
