package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	orderVolumeRecordSize = 8 + 8
)

type orderVolumeRecord struct {
	feeFilled    uint64
	amountFilled uint64
}

func (o *orderVolumeRecord) marshalBinary() ([]byte, error) {
	buf := make([]byte, orderVolumeRecordSize)
	binary.BigEndian.PutUint64(buf[:8], o.feeFilled)
	binary.BigEndian.PutUint64(buf[8:16], o.amountFilled)
	return buf, nil
}

func (o *orderVolumeRecord) unmarshalBinary(data []byte) error {
	if len(data) != orderVolumeRecordSize {
		return errInvalidDataSize
	}
	o.feeFilled = binary.BigEndian.Uint64(data[:8])
	o.amountFilled = binary.BigEndian.Uint64(data[8:16])
	return nil
}

type ordersVolumes struct {
	hs *historyStorage
}

func newOrdersVolumes(hs *historyStorage) *ordersVolumes {
	return &ordersVolumes{hs: hs}
}

func (ov *ordersVolumes) newestVolumeById(orderId []byte) (*orderVolumeRecord, error) {
	key := ordersVolumeKey{orderId}
	recordBytes, err := ov.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var record orderVolumeRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal order volume record: %v\n", err)
	}
	return &record, nil
}

func (ov *ordersVolumes) addNewRecord(orderId []byte, record *orderVolumeRecord, blockID proto.BlockID) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	key := ordersVolumeKey{orderId}
	return ov.hs.addNewEntry(ordersVolume, key.bytes(), recordBytes, blockID)
}

func (ov *ordersVolumes) increaseFilledFee(orderId []byte, feeChange uint64, blockID proto.BlockID) error {
	prevVolume, err := ov.newestVolumeById(orderId)
	if err != nil {
		// New record.
		return ov.addNewRecord(orderId, &orderVolumeRecord{feeFilled: feeChange}, blockID)
	}
	prevVolume.feeFilled += feeChange
	return ov.addNewRecord(orderId, prevVolume, blockID)
}

func (ov *ordersVolumes) increaseFilledAmount(orderId []byte, amountChange uint64, blockID proto.BlockID) error {
	prevVolume, err := ov.newestVolumeById(orderId)
	if err != nil {
		// New record.
		return ov.addNewRecord(orderId, &orderVolumeRecord{amountFilled: amountChange}, blockID)
	}
	prevVolume.amountFilled += amountChange
	return ov.addNewRecord(orderId, prevVolume, blockID)
}

func (ov *ordersVolumes) newestFilledFee(orderId []byte) (uint64, error) {
	volume, err := ov.newestVolumeById(orderId)
	if err != nil {
		// No fee volume filled yet.
		return 0, nil
	}
	return volume.feeFilled, nil
}

func (ov *ordersVolumes) newestFilledAmount(orderId []byte) (uint64, error) {
	volume, err := ov.newestVolumeById(orderId)
	if err != nil {
		// No amount volume filled yet.
		return 0, nil
	}
	return volume.amountFilled, nil
}
