package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	blockRewardRecordSize = 8
	rewardVotesRecordSize = 4 + 4
)

var (
	rewardVotesKeyBytes = []byte{rewardVotesKeyPrefix}
	blockRewardKeyBytes = []byte{blockRewardKeyPrefix}
)

type blockRewardRecord struct {
	reward uint64
}

func (r *blockRewardRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, blockRewardRecordSize)
	binary.BigEndian.PutUint64(res, r.reward)
	return res, nil
}

func (r *blockRewardRecord) unmarshalBinary(data []byte) error {
	if len(data) != blockRewardRecordSize {
		return errInvalidDataSize
	}
	r.reward = binary.BigEndian.Uint64(data)
	return nil
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

type monetaryPolicySettings struct {
	bs *settings.BlockchainSettings
}

func newMonetaryPolicySettings(bs *settings.BlockchainSettings) monetaryPolicySettings {
	return monetaryPolicySettings{bs: bs}
}

func (m monetaryPolicySettings) InitialBlockReward() uint64      { return m.bs.InitialBlockReward }
func (m monetaryPolicySettings) BlockRewardIncrement() uint64    { return m.bs.BlockRewardIncrement }
func (m monetaryPolicySettings) BlockRewardVotingPeriod() uint64 { return m.bs.BlockRewardVotingPeriod }

func (m monetaryPolicySettings) BlockRewardTerm(isCappedRewardActivated bool) uint64 {
	if isCappedRewardActivated {
		return m.bs.BlockRewardTermAfter20
	}
	return m.bs.BlockRewardTerm
}

type monetaryPolicy struct {
	settings monetaryPolicySettings
	hs       *historyStorage
}

func newMonetaryPolicy(hs *historyStorage, settings *settings.BlockchainSettings) *monetaryPolicy {
	return &monetaryPolicy{hs: hs, settings: newMonetaryPolicySettings(settings)}
}

func (m *monetaryPolicy) reward() (uint64, error) {
	var record blockRewardRecord
	b, err := m.hs.newestTopEntryData(blockRewardKeyBytes)
	if isNotFoundInHistoryOrDBErr(err) {
		return m.settings.InitialBlockReward(), nil
	}
	if err != nil {
		return 0, err
	}
	if err := record.unmarshalBinary(b); err != nil {
		return 0, err
	}
	return record.reward, nil
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

func (m *monetaryPolicy) updateBlockReward(lastBlockID, nextBlockID proto.BlockID) error {
	votes, err := m.votes()
	if err != nil {
		return err
	}
	reward, err := m.reward()
	if err != nil {
		return err
	}
	threshold := uint32(m.settings.BlockRewardVotingPeriod())/2 + 1
	switch {
	case votes.increase >= threshold:
		reward += m.settings.BlockRewardIncrement()
	case votes.decrease >= threshold:
		reward -= m.settings.BlockRewardIncrement()
	default:
		return m.resetBlockRewardVotes(nextBlockID) // nothing to do, reward remains the same, reset votes on the next block
	}
	record := blockRewardRecord{reward}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	// bind block reward to the last applied block
	if err := m.hs.addNewEntry(blockReward, blockRewardKeyBytes, recordBytes, lastBlockID); err != nil {
		return err
	}
	// bind votes reset to the next block which is being applied
	return m.resetBlockRewardVotes(nextBlockID)
}

func (m *monetaryPolicy) blockRewardVotingPeriod(height, activation proto.Height, isCappedRewardsActivated bool) (start, end uint64) {
	blockRewardTerm := m.settings.BlockRewardTerm(isCappedRewardsActivated)
	diff := height - activation
	next := activation + ((diff/blockRewardTerm)+1)*blockRewardTerm
	start = next - m.settings.BlockRewardVotingPeriod()
	end = next - 1
	return start, end
}

func isBlockRewardVotingPeriod(start, end, height proto.Height) bool {
	return height >= start && height <= end
}
