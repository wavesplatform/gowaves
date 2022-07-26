package state

import (
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
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

type monetaryPolicy struct {
	settings *settings.BlockchainSettings
	hs       *historyStorage
}

func newMonetaryPolicy(hs *historyStorage, settings *settings.BlockchainSettings) *monetaryPolicy {
	return &monetaryPolicy{hs: hs, settings: settings}
}

func (m *monetaryPolicy) reward() (uint64, error) {
	var record blockRewardRecord
	b, err := m.hs.newestTopEntryData(blockRewardKeyBytes)
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
		return m.settings.InitialBlockReward, nil
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
	if err == keyvalue.ErrNotFound || err == errEmptyHist {
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

func (m *monetaryPolicy) vote(desired int64, height, activation uint64, blockID proto.BlockID) error {
	if isStartOfTerm(height, activation, m.settings.FunctionalitySettings) {
		rec := rewardVotesRecord{0, 0}
		recordBytes, err := rec.marshalBinary()
		if err != nil {
			return err
		}
		return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes, recordBytes, blockID)
	}
	if desired < 0 { // there is no vote, nothing to count
		return nil
	}
	if !isVotingPeriod(height, activation, m.settings.FunctionalitySettings) { // voting is not started, do nothing
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
		return nil
	}
	recordBytes, err := rec.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, rewardVotesKeyBytes, recordBytes, blockID)
}

func (m *monetaryPolicy) updateBlockReward(h uint64, blockID proto.BlockID) error {
	votes, err := m.votes()
	if err != nil {
		return err
	}
	reward, err := m.reward()
	if err != nil {
		return err
	}
	threshold := uint32(m.settings.BlockRewardVotingPeriod)/2 + 1
	switch {
	case votes.increase >= threshold:
		reward += m.settings.BlockRewardIncrement
	case votes.decrease >= threshold:
		reward -= m.settings.BlockRewardIncrement
	}
	record := blockRewardRecord{reward}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(blockReward, blockRewardKeyBytes, recordBytes, blockID)
}

func blockRewardTermBoundaries(height, activation uint64, settings settings.FunctionalitySettings) (uint64, uint64) {
	diff := height - activation
	next := activation + ((diff/settings.BlockRewardTerm)+1)*settings.BlockRewardTerm
	start := next - settings.BlockRewardVotingPeriod
	end := next - 1
	return start, end
}

func isVotingPeriod(height, activation uint64, settings settings.FunctionalitySettings) bool {
	diff := height - activation
	next := activation + ((diff/settings.BlockRewardTerm)+1)*settings.BlockRewardTerm
	start := next - settings.BlockRewardVotingPeriod
	end := next - 1
	return height >= start && height <= end
}

func isStartOfTerm(height, activation uint64, settings settings.FunctionalitySettings) bool {
	diff := height - activation
	start := activation + (diff/settings.BlockRewardTerm)*settings.BlockRewardTerm
	return height == start
}
