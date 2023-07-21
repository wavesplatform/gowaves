package state

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type rewardsStorage struct {
	hs *historyStorage
}

type rewardRecord struct {
	Reward uint64 `cbor:"0,keyasint"`
}

func newRewardsStorage(hs *historyStorage) *rewardsStorage {
	return &rewardsStorage{hs}
}

func (s *rewardsStorage) saveReward(reward uint64, height proto.Height, blockID proto.BlockID) error {
	key := blockRewardAtHeightKey{height: height}
	rwRecord := rewardRecord{Reward: reward}
	recordBytes, err := cbor.Marshal(rwRecord)
	if err != nil {
		return errors.Wrapf(err, "failed to save reward in height '%d' in block '%s'", height, blockID.String())
	}
	return s.hs.addNewEntry(blockRewardAtHeight, key.bytes(), recordBytes, blockID)
}

func (s *rewardsStorage) reward(height proto.Height) (uint64, error) {
	key := blockRewardAtHeightKey{height: height}
	recordBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return 0, err
	}
	rwRecord := new(rewardRecord)
	if err = cbor.Unmarshal(recordBytes, rwRecord); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal account script complexities record")
	}
	return rwRecord.Reward, nil
}
