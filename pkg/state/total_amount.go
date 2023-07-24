package state //nolint: dupl // It similar with other records in history storage

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type totalWavesAmountStorage struct {
	hs *historyStorage
}

type totalWavesAmountRecord struct {
	Amount uint64 `cbor:"0,keyasint"`
}

func newTotalAmountStorage(hs *historyStorage) *totalWavesAmountStorage {
	return &totalWavesAmountStorage{hs}
}

func (s *totalWavesAmountStorage) saveTotalWavesAmount(
	amount uint64,
	height proto.Height,
	blockID proto.BlockID,
) error {
	key := totalWavesAmountKey{height: height}
	amountRecord := totalWavesAmountRecord{Amount: amount}
	recordBytes, err := cbor.Marshal(amountRecord)
	if err != nil {
		return errors.Wrapf(err, "failed to save total Waves amount in height '%d' in block '%s'", height, blockID.String())
	}
	return s.hs.addNewEntry(totalWavesAmount, key.bytes(), recordBytes, blockID)
}

func (s *totalWavesAmountStorage) totalWavesAmount(height proto.Height) (uint64, error) {
	key := totalWavesAmountKey{height: height}
	recordBytes, err := s.hs.topEntryData(key.bytes())
	if err != nil {
		return 0, err
	}
	amountRecord := new(totalWavesAmountRecord)
	if err = cbor.Unmarshal(recordBytes, amountRecord); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal total Waves amount record")
	}
	return amountRecord.Amount, nil
}
