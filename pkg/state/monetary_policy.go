package state

import (
	"encoding/binary"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	rewardVotesRecordSize = 4 + 4
)

func rewardVotesKeyBytes() []byte {
	return []byte{rewardVotesKeyPrefix}
}

func rewardChangesKeyBytes() []byte {
	return []byte{rewardChangesKeyPrefix}
}

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
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return m.settings.InitialBlockReward, nil
		}
		return 0, err
	}
	if len(rewardsChanges) == 0 {
		return m.settings.InitialBlockReward, nil
	}
	return rewardsChanges[len(rewardsChanges)-1].Reward, nil
}

func (m *monetaryPolicy) votes() (rewardVotesRecord, error) {
	var record rewardVotesRecord
	recordBytes, err := m.hs.newestTopEntryData(rewardVotesKeyBytes())
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
	return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes(), recordBytes, blockID)
}

func (m *monetaryPolicy) resetBlockRewardVotes(blockID proto.BlockID) error {
	rec := rewardVotesRecord{0, 0}
	recordBytes, err := rec.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes(), recordBytes, blockID)
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
	next := NextRewardTerm(height, activation, m.settings, isCappedRewardsActivated)
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

type rewardChangesRecords []rewardChangesRecord

func (r *rewardChangesRecords) marshalBinary() ([]byte, error) {
	return cbor.Marshal(*r)
}

func (r *rewardChangesRecords) unmarshalBinary(recordBytes []byte) error {
	return cbor.Unmarshal(recordBytes, r)
}

func (m *monetaryPolicy) getRewardChanges() (rewardChangesRecords, error) {
	prevRecordBytes, err := m.hs.newestTopEntryData(rewardChangesKeyBytes())
	if err != nil {
		return nil, err
	}
	var changesRecords rewardChangesRecords

	if err = changesRecords.unmarshalBinary(prevRecordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal reward changes records")
	}
	return changesRecords, nil
}

func (m *monetaryPolicy) saveNewRewardChange(newReward uint64, height proto.Height, blockID proto.BlockID) error {
	changesRecords, err := m.getRewardChanges()
	if err != nil && !isNotFoundInHistoryOrDBErr(err) {
		return err
	}
	changesRecords = append(changesRecords, rewardChangesRecord{Height: height, Reward: newReward})
	newRecordBytes, err := changesRecords.marshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to save reward changes in height '%d' in block '%s'", height, blockID.String())
	}
	return m.hs.addNewEntry(rewardChanges, rewardChangesKeyBytes(), newRecordBytes, blockID)
}

func (m *monetaryPolicy) rewardAtHeight(height proto.Height, blockRewardActivationHeight proto.Height) (uint64, error) {
	changesRecords, err := m.getRewardChanges()
	if err != nil && !isNotFoundInHistoryOrDBErr(err) {
		return 0, err
	}
	// If the BlockReward feature was activated at the start of the blockchain, then its height will be 1.
	// But in the first block (genesis), we don't have a reward for the block, so we should increment this height
	if blockRewardActivationHeight == 1 {
		blockRewardActivationHeight++
	}
	changesRecords = append(rewardChangesRecords{{
		Height: blockRewardActivationHeight,
		Reward: m.settings.InitialBlockReward,
	}}, changesRecords...)

	curReward := uint64(0)
	for _, change := range changesRecords {
		if height < change.Height {
			break
		}
		curReward = change.Reward
	}
	return curReward, nil
}

func (m *monetaryPolicy) totalAmountAtHeight(
	height, initialTotalAmount uint64,
	blockRewardActivationHeight proto.Height,
) (uint64, error) {
	changesRecords, err := m.getRewardChanges()
	if err != nil && !isNotFoundInHistoryOrDBErr(err) {
		return 0, err
	}
	// If the BlockReward feature was activated at the start of the blockchain, then its height will be 1.
	// But in the first block (genesis), we don't have a reward for the block, so we should increment this height
	if blockRewardActivationHeight == 1 {
		blockRewardActivationHeight++
	}
	changesRecords = append(rewardChangesRecords{{
		Height: blockRewardActivationHeight,
		Reward: m.settings.InitialBlockReward,
	}}, changesRecords...)

	curTotalAmount := initialTotalAmount
	prevHeight := uint64(0)
	isNotLast := false
	for i := len(changesRecords) - 1; i >= 0; i-- {
		change := changesRecords[i]
		if height < change.Height {
			continue
		}
		if height >= change.Height && !isNotLast {
			curTotalAmount += change.Reward * (height - (change.Height - 1))
			isNotLast = true
		} else {
			curTotalAmount += change.Reward * (prevHeight - (change.Height - 1))
		}
		prevHeight = change.Height - 1
	}

	return curTotalAmount, nil
}

func NextRewardTerm(
	height, activation proto.Height,
	set *settings.BlockchainSettings,
	isCappedRewardsActivated bool,
) uint64 {
	blockRewardTerm := set.CurrentBlockRewardTerm(isCappedRewardsActivated)
	diff := height - activation
	return activation + ((diff/blockRewardTerm)+1)*blockRewardTerm
}
