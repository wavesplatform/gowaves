package state

import (
	"encoding/binary"

	"github.com/mr-tron/base58"
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

func (ov *ordersVolumes) newestVolumeByID(orderID []byte) (*orderVolumeRecord, error) {
	key := ordersVolumeKey{orderID}
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

func (ov *ordersVolumes) addNewRecord(orderID []byte, record *orderVolumeRecord, blockID proto.BlockID) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	key := ordersVolumeKey{orderID}
	return ov.hs.addNewEntry(ordersVolume, key.bytes(), recordBytes, blockID)
}

// TODO remove it
func (ov *ordersVolumes) increaseFilled(orderID []byte, amountChange, feeChange uint64, blockID proto.BlockID) error {
	prevVolume, err := ov.newestVolumeByID(orderID)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // New record.
			return ov.addNewRecord(orderID, &orderVolumeRecord{amountFilled: amountChange, feeFilled: feeChange}, blockID)
		}
		return errors.Wrapf(err, "failed to increase filled for order %q", base58.Encode(orderID))
	}
	prevVolume.amountFilled += amountChange
	prevVolume.feeFilled += feeChange
	return ov.addNewRecord(orderID, prevVolume, blockID)
}

func (ov *ordersVolumes) storeFilled(orderID []byte, amountFilled, feeFilled uint64, blockID proto.BlockID) error {
	newVolume := &orderVolumeRecord{amountFilled: amountFilled, feeFilled: feeFilled}
	return ov.addNewRecord(orderID, newVolume, blockID)
}

func (ov *ordersVolumes) newestFilled(orderID []byte) (uint64, uint64, error) {
	volume, err := ov.newestVolumeByID(orderID)
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No fee volume filled yet.
			return 0, 0, nil
		}
		return 0, 0, errors.Wrapf(err, "failed to get filled for order %q", base58.Encode(orderID))
	}
	return volume.amountFilled, volume.feeFilled, nil
}
