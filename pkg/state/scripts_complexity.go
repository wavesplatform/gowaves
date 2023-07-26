package state

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type estimationRecord struct {
	EstimatorVersion uint8               `cbor:"0,keyasint"`
	Estimation       ride.TreeEstimation `cbor:"1,keyasint"`
}

func (e *estimationRecord) marshalBinary() ([]byte, error) {
	type shadowed *estimationRecord
	return cbor.Marshal(shadowed(e))
}

func (e *estimationRecord) unmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty binary data, estimation doesn't exist")
	}
	type shadowed *estimationRecord
	return cbor.Unmarshal(data, shadowed(e))
}

type scriptsComplexity struct {
	hs *historyStorage
}

func newScriptsComplexity(hs *historyStorage) *scriptsComplexity {
	return &scriptsComplexity{hs: hs}
}
func (sc *scriptsComplexity) newestScriptEstimationRecordByAddr(addr proto.Address) (*estimationRecord, error) {
	key := accountScriptComplexityKey{addr.ID()}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	r := new(estimationRecord)
	if err = r.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account script complexities record")
	}
	return r, nil
}

func (sc *scriptsComplexity) newestScriptComplexityByAddr(addr proto.Address) (*ride.TreeEstimation, error) {
	r, err := sc.newestScriptEstimationRecordByAddr(addr)
	if err != nil {
		return nil, err
	}
	return &r.Estimation, nil
}

// newestOriginalScriptComplexityByAddr returns original estimated script complexity by the given address.
// For account scripts we have to use original estimation.
func (sc *scriptsComplexity) newestOriginalScriptComplexityByAddr(addr proto.Address) (*ride.TreeEstimation, error) {
	key := accountScriptOriginalComplexityKey{addr.ID()}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	r := new(estimationRecord)
	if err = r.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal original account script complexities record")
	}
	return &r.Estimation, nil
}

func (sc *scriptsComplexity) newestOriginalScriptComplexityExist(addr proto.Address) (bool, error) {
	key := accountScriptOriginalComplexityKey{addr.ID()}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) {
			return false, nil
		}
		return false, err
	}
	return len(recordBytes) != 0, nil
}

func (sc *scriptsComplexity) newestScriptComplexityByAsset(asset proto.AssetID) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	r := new(estimationRecord)
	if err = r.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal asset script complexities record")
	}
	return &r.Estimation, nil
}

func (sc *scriptsComplexity) scriptComplexityByAsset(asset proto.AssetID) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	r := new(estimationRecord)
	if err = r.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal asset script complexities record")
	}
	return &r.Estimation, nil
}

func (sc *scriptsComplexity) scriptComplexityByAddress(addr proto.Address) (*ride.TreeEstimation, error) {
	key := accountScriptComplexityKey{addr.ID()}
	recordBytes, err := sc.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	r := new(estimationRecord)
	if err = r.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account script complexities record")
	}
	return &r.Estimation, nil
}

func (sc *scriptsComplexity) saveComplexitiesForAddr(
	addr proto.Address,
	se scriptEstimation,
	blockID proto.BlockID,
) error {
	var (
		addrID                = addr.ID()
		complexityKey         = accountScriptComplexityKey{addrID}
		originalComplexityKey = accountScriptOriginalComplexityKey{addrID}
	)
	if se.scriptIsEmpty { // write empty data (nullify) in case when we've received an emtpy script
		err := sc.hs.addNewEntry(accountScriptComplexity, complexityKey.bytes(), nil, blockID)
		if err != nil {
			return errors.Wrapf(err, "failed erase estimation record for address '%s' in block '%s'",
				addr.String(), blockID.String(),
			)
		}
		err = sc.hs.addNewEntry(accountScriptOriginalComplexity, originalComplexityKey.bytes(), nil, blockID)
		if err != nil {
			return errors.Wrapf(err, "failed erase original estimation record for address '%s' in block '%s'",
				addr.String(), blockID.String(),
			)
		}
		return nil
	}
	// prepare record bytes
	complexityRecord := estimationRecord{
		EstimatorVersion: uint8(se.currentEstimatorVersion),
		Estimation:       se.estimation,
	}
	recordBytes, err := complexityRecord.marshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	// save new estimation record
	err = sc.hs.addNewEntry(accountScriptComplexity, complexityKey.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	// save or skip saving estimation record as an original estimation
	originalEstimationExist, err := sc.newestOriginalScriptComplexityExist(addr)
	if err != nil {
		return errors.Wrapf(err, "failed to check original estimator version existence for addr '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	if !originalEstimationExist { // this is new account script, we should save provided estimation as original
		err = sc.hs.addNewEntry(accountScriptOriginalComplexity, originalComplexityKey.bytes(), recordBytes, blockID)
		if err != nil {
			return errors.Wrapf(err, "failed to save original estimator version for address '%s' in block '%s'",
				addr.String(), blockID.String(),
			)
		}
	}
	return nil
}

func (sc *scriptsComplexity) saveComplexitiesForAsset(
	asset crypto.Digest,
	se scriptEstimation,
	blockID proto.BlockID,
) error {
	complexityRecord := estimationRecord{
		EstimatorVersion: uint8(se.currentEstimatorVersion),
		Estimation:       se.estimation,
	}
	recordBytes, err := complexityRecord.marshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to save complexity record for asset '%s' in block '%s'",
			asset.String(), blockID.String(),
		)
	}
	key := assetScriptComplexityKey{proto.AssetIDFromDigest(asset)}
	err = sc.hs.addNewEntry(assetScriptComplexity, key.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to save complexity record for asset '%s' in block '%s'",
			asset.String(), blockID.String(),
		)
	}
	return nil
}
