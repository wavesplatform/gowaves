package state

import (
	"encoding/binary"
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	blockRewardRecordSize = 8
	rewardVotesRecordSize = 4 + 4
)

var (
	rewardVotesKeyBytes   = []byte{rewardVotesKeyPrefix}
	rewardChangesKeyBytes = []byte{rewardChangesKeyPrefix}
)

type rewardVotesRecord struct {
	increase uint32
	decrease uint32
}

func (r *rewardVotesRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, rewardVotesRecordSize)
	binary.BigEndian.PutUint32(res, r.increase)
	binary.BigEndian.PutUint32(res[4:], r.decrease)
	return res, nil
}

func (r *rewardVotesRecord) unmarshalBinary(data []byte) error {
	if len(data) != rewardVotesRecordSize {
		return errInvalidDataSize
	}
	r.increase = binary.BigEndian.Uint32(data[:4])
	r.decrease = binary.BigEndian.Uint32(data[4:])
	return nil
}

type monetaryPolicy struct {
	settings *settings.BlockchainSettings
	hs       *historyStorage
}

func newMonetaryPolicy(hs *historyStorage, settings *settings.BlockchainSettings) *monetaryPolicy {
	return &monetaryPolicy{hs: hs, settings: settings}
}

func (m *monetaryPolicy) reward() (uint64, error) {
	rewardsChanges, err := m.getRewardChanges()
	if isNotFoundInHistoryOrDBErr(err) || len(rewardsChanges) == 0 {
		return m.settings.InitialBlockReward, nil
	}
	if err != nil {
		return 0, err
	}
	return rewardsChanges[len(rewardsChanges)-1].Reward, nil
}

func (m *monetaryPolicy) votes() (rewardVotesRecord, error) {
	var record rewardVotesRecord
	recordBytes, err := m.hs.newestTopEntryData(rewardVotesKeyBytes)
	if isNotFoundInHistoryOrDBErr(err) {
		return record, nil
	}
	if err != nil {
		return record, err
	}
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return record, err
	}
	return record, nil
}

func (m *monetaryPolicy) vote(desired int64, height, activation proto.Height, isCappedRewardsActive bool, blockID proto.BlockID) error {
	start, end := m.blockRewardVotingPeriod(height, activation, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return nil
	}
	if desired < 0 { // there is no vote, nothing to count
		return nil
	}
	target := uint64(desired)
	current, err := m.reward()
	if err != nil {
		return err
	}
	rec, err := m.votes()
	if err != nil {
		return err
	}
	switch {
	case target > current:
		rec.increase++
	case target < current:
		rec.decrease++
	default:
		return nil // nothing to do, target == current
	}
	recordBytes, err := rec.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes, recordBytes, blockID)
}

func (m *monetaryPolicy) resetBlockRewardVotes(blockID proto.BlockID) error {
	rec := rewardVotesRecord{0, 0}
	recordBytes, err := rec.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes, recordBytes, blockID)
}

func (m *monetaryPolicy) updateBlockReward(lastBlockID, nextBlockID proto.BlockID, height proto.Height) error {
	votes, err := m.votes()
	if err != nil {
		return err
	}
	reward, err := m.reward()
	if err != nil {
		return err
	}
	threshold := uint32(m.settings.BlockRewardVotingThreshold())
	switch {
	case votes.increase >= threshold:
		reward += m.settings.BlockRewardIncrement
	case votes.decrease >= threshold:
		reward -= m.settings.BlockRewardIncrement
	default:
		return m.resetBlockRewardVotes(nextBlockID) // nothing to do, reward remains the same, reset votes on the next block
	}
	if err = m.saveNewRewardChange(reward, height, lastBlockID); err != nil {
		return err
	}
	// bind votes reset to the next block which is being applied
	return m.resetBlockRewardVotes(nextBlockID)
}

func (m *monetaryPolicy) blockRewardVotingPeriod(height, activation proto.Height, isCappedRewardsActivated bool) (start, end uint64) {
	next := m.settings.NextRewardTerm(height, activation, isCappedRewardsActivated)
	start = next - m.settings.BlockRewardVotingPeriod
	end = next - 1
	return start, end
}

func isBlockRewardVotingPeriod(start, end, height proto.Height) bool {
	return height >= start && height <= end
}

type rewardChangesRecord struct {
	Height uint64 `cbor:"0,keyasint"`
	Reward uint64 `cbor:"1,keyasint"`
}

func (m *monetaryPolicy) getRewardChanges() ([]rewardChangesRecord, error) {
	prevRecordBytes, err := m.hs.newestTopEntryData(rewardChangesKeyBytes)
	if err != nil {
		return nil, err
	}
	var changesRecords []rewardChangesRecord
	if err = cbor.Unmarshal(prevRecordBytes, &changesRecords); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal reward changes records")
	}
	return changesRecords, nil
}

func (m *monetaryPolicy) saveNewRewardChange(newReward uint64, height proto.Height, blockID proto.BlockID) error {
	changesRecords, err := m.getRewardChanges()
	if !isNotFoundInHistoryOrDBErr(err) && err != nil {
		return err
	}
	changesRecords = append(changesRecords, rewardChangesRecord{Height: height, Reward: newReward})
	newRecordBytes, err := cbor.Marshal(changesRecords)
	if err != nil {
		return errors.Wrapf(err, "failed to save reward changes in height '%d' in block '%s'", height, blockID.String())
	}
	return m.hs.addNewEntry(rewardChanges, rewardChangesKeyBytes, newRecordBytes, blockID)
}

func (m *monetaryPolicy) rewardAtHeight(height proto.Height, blockRewardActivationHeight proto.Height) (uint64, error) {
	changesRecords, err := m.getRewardChanges()
	if !isNotFoundInHistoryOrDBErr(err) && err != nil {
		return 0, err
	}
	changesRecords = append([]rewardChangesRecord{{blockRewardActivationHeight, m.settings.InitialBlockReward}}, changesRecords...)

	curReward := uint64(0)
	for _, change := range changesRecords {
		curReward = change.Reward
		if height < change.Height {
			break
		}
	}
	return curReward, nil
}

func (m *monetaryPolicy) totalAmountAtHeight(height, initialTotalAmount uint64, blockRewardActivationHeight proto.Height) (uint64, error) {
	changesRecords, err := m.getRewardChanges()
	if !isNotFoundInHistoryOrDBErr(err) && err != nil {
		return 0, err
	}
	changesRecords = append([]rewardChangesRecord{{blockRewardActivationHeight, m.settings.InitialBlockReward}}, changesRecords...)

	curTotalAmount := initialTotalAmount
	prevHeight := uint64(0)
	isNotLast := false
	for i := len(changesRecords) - 1; i >= 0; i-- {
		if height < changesRecords[i].Height {
			continue
		}
		if height > changesRecords[i].Height && !isNotLast {
			curTotalAmount += changesRecords[i].Reward * (height - changesRecords[i].Height)
			isNotLast = true
		} else {
			curTotalAmount += changesRecords[i].Reward * (prevHeight - changesRecords[i].Height)
		}
		prevHeight = changesRecords[i].Height
	}

	return curTotalAmount, nil
}
