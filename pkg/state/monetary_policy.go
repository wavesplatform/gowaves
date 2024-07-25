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

func zeroVotesRecord() rewardVotesRecord { return rewardVotesRecord{0, 0} }

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

// reward returns the current reward.
// If there are no reward changes, returns the initial reward from settings.
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

func (m *monetaryPolicy) newestVotes(
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
) (rewardVotesRecord, error) {
	start, end := m.blockRewardVotingPeriod(height, blockRewardActivationHeight, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return zeroVotesRecord(), nil
	}
	key := rewardVotesKey{height: height}
	var votesRecord rewardVotesRecord
	recordBytes, err := m.hs.newestTopEntryData(key.bytes())
	if isNotFoundInHistoryOrDBErr(err) {
		return votesRecord, nil
	}
	if err != nil {
		return votesRecord, err
	}
	if err = votesRecord.unmarshalBinary(recordBytes); err != nil {
		return votesRecord, err
	}
	return votesRecord, nil
}

func (m *monetaryPolicy) votes(
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
) (rewardVotesRecord, error) {
	start, end := m.blockRewardVotingPeriod(height, blockRewardActivationHeight, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return zeroVotesRecord(), nil
	}
	key := rewardVotesKey{height: height}
	var votesRecord rewardVotesRecord
	recordBytes, err := m.hs.topEntryData(key.bytes())
	if isNotFoundInHistoryOrDBErr(err) {
		return votesRecord, nil
	}
	if err != nil {
		return votesRecord, err
	}
	if err = votesRecord.unmarshalBinary(recordBytes); err != nil {
		return votesRecord, err
	}
	return votesRecord, nil
}

func (m *monetaryPolicy) vote(
	desired int64,
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
	blockID proto.BlockID,
) error {
	start, end := m.blockRewardVotingPeriod(height, blockRewardActivationHeight, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return nil // no need to save anything, because voting is not started
	}
	if desired < 0 { // there is no vote, nothing to count
		return m.saveVotes(zeroVotesRecord(), blockID, height)
	}
	target := uint64(desired)
	current, err := m.reward()
	if err != nil {
		return err
	}
	rec, err := m.newestVotes(height-1, blockRewardActivationHeight, isCappedRewardsActive)
	if err != nil {
		return err
	}
	switch {
	case target > current:
		rec.increase++
	case target < current:
		rec.decrease++
	}
	return m.saveVotes(rec, blockID, height)
}

func (m *monetaryPolicy) saveVotes(votes rewardVotesRecord, blockID proto.BlockID, height proto.Height) error {
	key := rewardVotesKey{height: height}
	recordBytes, err := votes.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, key.bytes(), recordBytes, blockID)
}

func (m *monetaryPolicy) updateBlockReward(
	lastBlockID proto.BlockID,
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
) error {
	votes, err := m.newestVotes(height, blockRewardActivationHeight, isCappedRewardsActive)
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
		return nil // nothing to do, reward remains the same
	}
	return m.saveNewRewardChange(reward, height, lastBlockID)
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
	blockRewardActivationHeight, boostFirst, boostLast proto.Height,
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

	br := &boostedReward{first: boostFirst, last: boostLast}

	return calculateTotalAmount(height, changesRecords, initialTotalAmount, br), nil
}

func calculateTotalAmount(
	relativeHeight proto.Height,
	changesRecords rewardChangesRecords, // changesRecords must be sorted in ascending order
	curTotalAmount uint64,
	br *boostedReward,
) uint64 {
	for i := len(changesRecords) - 1; i >= 0; i-- {
		change := changesRecords[i]
		if relativeHeight < change.Height {
			continue
		}
		curTotalAmount += br.reward(change.Reward, change.Height, relativeHeight)
		relativeHeight = change.Height - 1
	}
	return curTotalAmount
}

type boostedReward struct {
	first uint64
	last  uint64
}

func (b *boostedReward) reward(reward uint64, changeHeight, height proto.Height) uint64 {
	total := height - (changeHeight - 1)
	var boosted uint64

	// block with first boost == b.first -> boost start -> bs
	// block with last boost == b.last -> boost end -> bs
	// change height -> ch, height -> h
	// Cases:
	// 0. ----ch-------h-------(bs------be)-----> (first=bs, last=h)  ==> last <= first, no intersection
	// 1. ----ch------|(bs*****|h-------be)-----> (first=bs, last=h)  ==> last > first,  intersection ***
	// 2. ----(bs------|ch*****|h-------be)-----> (first=ch, last=h)  ==> last > first,  intersection ***
	// 3. ----(bs------|ch*****|be)------h------> (first=ch, last=be) ==> last > first,  intersection ***
	// 4. ----(bs-------be)------ch------h------> (first=ch, last=be) ==> last <= first, no intersection
	var (
		first = max(b.first-1, changeHeight-1) // -1 for case when boosted period == 1 block
		last  = min(b.last, height)
	)

	if last > first {
		boosted = last - first
	}

	var r uint64
	if boosted > 0 {
		r += boostedRewardMultiplier * reward * boosted
	}
	r += reward * (total - boosted)
	return r
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
