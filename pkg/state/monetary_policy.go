package state

import (
	"cmp"
	"slices"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const (
	rewardVotesRecordsPackMaxSize = 1000 // one rewardVotesRecord should contain no more than 1000 votes records
)

func zeroVotesRecord() rewardVotesPair { return rewardVotesPair{0, 0} }

func rewardChangesKeyBytes() []byte {
	return []byte{rewardChangesKeyPrefix}
}

type rewardVotesPair struct {
	Increase uint16 `cbor:"0,keyasint,omitempty"`
	Decrease uint16 `cbor:"1,keyasint,omitempty"`
}

type rewardVotesRecord struct {
	Height proto.Height    `cbor:"0,keyasint,omitempty"`
	Votes  rewardVotesPair `cbor:"1,keyasint,omitempty"`
}

type rewardVotesPack struct {
	Records []rewardVotesRecord `cbor:"0,keyasint"`
}

func (r *rewardVotesPack) AppendVotesPair(height proto.Height, votes rewardVotesPair) error {
	if l := len(r.Records); l > 0 {
		if last := r.Records[l-1]; last.Height >= height { // increasing order is required
			return errors.Errorf("height %d of the new record must be greater than the height %d of the last record",
				height, last.Height,
			)
		}
		if l >= rewardVotesRecordsPackMaxSize { // sanity check
			return errors.Errorf("votes records pack must contain no more than %d records",
				rewardVotesRecordsPackMaxSize,
			)
		}
	}
	r.Records = append(r.Records, rewardVotesRecord{Height: height, Votes: votes})
	return nil
}

func (r *rewardVotesPack) VotesAtHeight(height proto.Height) (rewardVotesPair, bool) {
	i, found := slices.BinarySearchFunc(r.Records, height, func(r rewardVotesRecord, target proto.Height) int {
		return cmp.Compare(r.Height, target)
	})
	if !found {
		return zeroVotesRecord(), false
	}
	return r.Records[i].Votes, true
}

func (r *rewardVotesPack) marshalBinary() ([]byte, error) {
	return cbor.Marshal(r)
}

func (r *rewardVotesPack) unmarshalBinary(data []byte) error {
	return cbor.Unmarshal(data, r)
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

func (m *monetaryPolicy) newestRewardVotesPack(height proto.Height) (rewardVotesPack, error) {
	key := rewardVotesPackKey{height: height}
	var votesPack rewardVotesPack
	recordBytes, err := m.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return votesPack, err
	}
	if err = votesPack.unmarshalBinary(recordBytes); err != nil {
		return votesPack, err
	}
	return votesPack, nil
}

func (m *monetaryPolicy) rewardVotesPack(height proto.Height) (rewardVotesPack, error) {
	key := rewardVotesPackKey{height: height}
	var votesPack rewardVotesPack
	recordBytes, err := m.hs.topEntryData(key.bytes())
	if err != nil {
		return votesPack, err
	}
	if err = votesPack.unmarshalBinary(recordBytes); err != nil {
		return votesPack, err
	}
	return votesPack, nil
}

func (m *monetaryPolicy) newestVotes(
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
) (rewardVotesPair, error) {
	start, end := m.blockRewardVotingPeriod(height, blockRewardActivationHeight, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return zeroVotesRecord(), nil
	}
	votesPack, err := m.newestRewardVotesPack(height)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return zeroVotesRecord(), nil
		}
		return rewardVotesPair{}, err
	}
	votesPair, found := votesPack.VotesAtHeight(height)
	if !found {
		return zeroVotesRecord(), nil
	}
	return votesPair, nil
}

func (m *monetaryPolicy) votes(
	height proto.Height,
	blockRewardActivationHeight proto.Height,
	isCappedRewardsActive bool,
) (rewardVotesPair, error) {
	start, end := m.blockRewardVotingPeriod(height, blockRewardActivationHeight, isCappedRewardsActive)
	if !isBlockRewardVotingPeriod(start, end, height) { // voting is not started, do nothing
		return zeroVotesRecord(), nil
	}
	votesPack, err := m.rewardVotesPack(height)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return zeroVotesRecord(), nil
		}
		return rewardVotesPair{}, err
	}
	votesPair, found := votesPack.VotesAtHeight(height)
	if !found {
		return zeroVotesRecord(), nil
	}
	return votesPair, nil
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
	if desired < 0 { // there is no vote, nothing to count, so also nothing to save
		return nil
	}
	target := uint64(desired)
	current, err := m.reward()
	if err != nil {
		return err
	}
	prevHeight := height - 1
	votesPack, err := m.newestRewardVotesPack(prevHeight)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			votesPack = rewardVotesPack{} // no votes yet
		} else {
			return err // some other error
		}
	}
	rec, found := votesPack.VotesAtHeight(prevHeight)
	if !found {
		rec = zeroVotesRecord()
	}
	switch {
	case target > current:
		inc, incErr := common.AddInt(rec.Increase, 1)
		if incErr != nil {
			return errors.Wrapf(incErr, "failed to increment votes for increasing reward for block '%s' at height '%d'",
				blockID.String(), height,
			)
		}
		rec.Increase = inc
	case target < current:
		dec, decErr := common.AddInt(rec.Decrease, 1)
		if decErr != nil {
			return errors.Wrapf(decErr, "failed to increment votes for decreasing reward for block '%s' at height '%d'",
				blockID.String(), height,
			)
		}
		rec.Decrease = dec
	}
	if addErr := votesPack.AppendVotesPair(height, rec); addErr != nil {
		return errors.Wrapf(addErr, "failed to append votes for block '%s' at height '%d'",
			blockID.String(), height,
		)
	}
	return m.saveVotesPack(votesPack, blockID, height)
}

func (m *monetaryPolicy) saveVotesPack(pack rewardVotesPack, blockID proto.BlockID, height proto.Height) error {
	key := rewardVotesPackKey{height: height}
	recordBytes, err := pack.marshalBinary()
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
	threshold := m.settings.BlockRewardVotingThreshold()
	switch {
	case uint64(votes.Increase) >= threshold:
		reward += m.settings.BlockRewardIncrement
	case uint64(votes.Decrease) >= threshold:
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
