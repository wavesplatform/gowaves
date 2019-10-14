package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	blockRewardRecordSize = 8
	rewardVotesRecordSize = 4 + 4
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
		return errors.New("invalid data size")
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
		return errors.New("invalid data size")
	}
	r.increase = binary.BigEndian.Uint32(data[:4])
	r.decrease = binary.BigEndian.Uint32(data[4:])
	return nil
}

type monetaryPolicy struct {
	db       keyvalue.IterableKeyVal
	batch    keyvalue.Batch
	hs       *historyStorage
	settings *settings.BlockchainSettings
}

func newMonetaryPolicy(db keyvalue.IterableKeyVal, batch keyvalue.Batch, hs *historyStorage, settings *settings.BlockchainSettings) (*monetaryPolicy, error) {
	return &monetaryPolicy{db: db, batch: batch, hs: hs, settings: settings}, nil
}

func (m *monetaryPolicy) currentReward() (uint64, error) {
	key := []byte{blockRewardKey}
	var record blockRewardRecord
	b, err := m.hs.freshLatestEntryData(key, true)
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
	key := []byte{rewardVotesKey}
	var record rewardVotesRecord
	recordBytes, err := m.hs.freshLatestEntryData(key, true)
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

func (m *monetaryPolicy) addVote(desired int64, blockID crypto.Signature) error {
	if desired < 0 {
		return nil
	}
	target := uint64(desired)
	current, err := m.currentReward()
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
	key := []byte{rewardVotesKey}
	recordBytes, err := rec.marshalBinary()
	if err != nil {
		return err
	}
	return m.hs.addNewEntry(rewardVotes, key, recordBytes, blockID)
}
