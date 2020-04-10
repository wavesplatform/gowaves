package state

import (
	"encoding/binary"

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

type features struct {
	rw                  *blockReadWriter
	db                  keyvalue.IterableKeyVal
	hs                  *historyStorage
	settings            *settings.BlockchainSettings
	definedFeaturesInfo map[settings.Feature]settings.FeatureInfo
}

func newFeatures(
	rw *blockReadWriter,
	db keyvalue.IterableKeyVal,
	hs *historyStorage,
	settings *settings.BlockchainSettings,
	definedFeaturesInfo map[settings.Feature]settings.FeatureInfo,
) (*features, error) {
	return &features{rw, db, hs, settings, definedFeaturesInfo}, nil
}

// addVote adds vote for feature by its featureID at given blockID.
func (f *features) addVote(featureID int16, blockID proto.BlockID) error {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return err
	}
	prevVotes, err := f.featureVotes(featureID)
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
	recordBytes, err := f.hs.entryDataAtHeight(keyBytes, height, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist || recordBytes == nil {
		// 0 votes for unknown feature.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return f.votesFromRecord(recordBytes)
}

func (f *features) featureVotesStable(featureID int16) (uint64, error) {
	key := votesFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.latestEntryData(keyBytes, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
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
	recordBytes, err := f.hs.freshLatestEntryData(keyBytes, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		// 0 votes for unknown feature.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return f.votesFromRecord(recordBytes)
}

func (f *features) printActivationLog(featureID int16) {
	info, ok := f.definedFeaturesInfo[settings.Feature(featureID)]
	if ok {
		zap.S().Infof("Activating feature %d (%s)", featureID, info.Description)
	} else {
		zap.S().Warnf("Activating UNKNOWN feature %d", featureID)
	}
	if !ok || !info.Implemented {
		zap.S().Warn("FATAL: UNKNOWN/UNIMPLEMENTED feature has been activated on the blockchain!")
		zap.S().Warn("FOR THIS REASON THE NODE IS STOPPED AUTOMATICALLY.")
		zap.S().Fatalf("PLEASE, UPDATE THE NODE IMMEDIATELY!")
	}
}

func (f *features) activateFeature(featureID int16, r *activatedFeaturesRecord, blockID proto.BlockID) error {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return err
	}
	recordBytes, err := r.marshalBinary()
	if err != nil {
		return err
	}
	f.printActivationLog(featureID)
	return f.hs.addNewEntry(activatedFeature, keyBytes, recordBytes, blockID)
}

func (f *features) isActivatedForNBlocks(featureID int16, n int) (bool, error) {
	activationHeight, err := f.activationHeight(featureID)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	curBlockHeight := f.rw.addingBlockHeight()
	if curBlockHeight < uint64(n) {
		return false, nil
	}
	return curBlockHeight-uint64(n) >= activationHeight, nil
}

func (f *features) newestIsActivated(featureID int16) (bool, error) {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return false, err
	}
	_, err = f.hs.freshLatestEntryData(keyBytes, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (f *features) isActivated(featureID int16) (bool, error) {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return false, err
	}
	_, err = f.hs.latestEntryData(keyBytes, true)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
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

func (f *features) activatedFeaturesRecord(featureID int16) (*activatedFeaturesRecord, error) {
	key := activatedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return nil, err
	}
	recordBytes, err := f.hs.latestEntryData(keyBytes, true)
	if err != nil {
		return nil, err
	}
	var record activatedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (f *features) activationHeight(featureID int16) (uint64, error) {
	record, err := f.activatedFeaturesRecord(featureID)
	if err != nil {
		return 0, err
	}
	return record.activationHeight, nil
}

func (f *features) printApprovalLog(featureID int16) {
	info, ok := f.definedFeaturesInfo[settings.Feature(featureID)]
	if ok {
		zap.S().Infof("Approving feature %d (%s)", featureID, info.Description)
	} else {
		zap.S().Infof("Approving UNKNOWN feature %d", featureID)
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
	f.printApprovalLog(featureID)
	return f.hs.addNewEntry(approvedFeature, keyBytes, recordBytes, blockID)
}

func (f *features) isApproved(featureID int16) (bool, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return false, err
	}
	_, err = f.hs.latestEntryData(keyBytes, true)
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

func (f *features) approvalHeight(featureID int16) (uint64, error) {
	key := approvedFeaturesKey{featureID: featureID}
	keyBytes, err := key.bytes()
	if err != nil {
		return 0, err
	}
	recordBytes, err := f.hs.latestEntryData(keyBytes, true)
	if err != nil {
		return 0, err
	}
	var record approvedFeaturesRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return 0, err
	}
	return record.approvalHeight, nil
}

func (f *features) isElected(height uint64, featureID int16) (bool, error) {
	votes, err := f.featureVotes(featureID)
	if err != nil {
		return false, err
	}
	return votes >= f.settings.VotesForFeatureElection(height), nil
}

func (f *features) resetVotes(blockID proto.BlockID) error {
	iter, err := f.db.NewKeyIterator([]byte{votesFeaturesKeyPrefix})
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
	iter, err := f.db.NewKeyIterator([]byte{votesFeaturesKeyPrefix})
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
		alreadyApproved, err := f.isApproved(k.featureID)
		if err != nil {
			return err
		}
		if alreadyApproved {
			continue
		}
		elected, err := f.isElected(curHeight, k.featureID)
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
	iter, err := f.db.NewKeyIterator([]byte{approvedFeaturesKeyPrefix})
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
		alreadyActivated, err := f.isActivated(k.featureID)
		if err != nil {
			return err
		}
		if alreadyActivated {
			continue
		}
		approvalHeight, err := f.approvalHeight(k.featureID)
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
	iter, err := f.db.NewKeyIterator([]byte{votesFeaturesKeyPrefix})
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
