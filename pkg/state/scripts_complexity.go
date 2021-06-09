package state

import (
	"math"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type estimatorVersionRecord struct {
	Version uint8 `cbor:"0,keyasint"`
}

type scriptsComplexity struct {
	hs *historyStorage
}

func newScriptsComplexity(hs *historyStorage) *scriptsComplexity {
	return &scriptsComplexity{hs: hs}
}

func (sc *scriptsComplexity) newestScriptComplexityByAddr(addr proto.Address, ev int, filter bool) (*ride.TreeEstimation, error) {
	key := accountScriptComplexityKey{ev, addr}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	record := new(ride.TreeEstimation)
	if err = cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account script complexities record")
	}
	return record, nil
}

func (sc *scriptsComplexity) originalEstimatorVersion(addr proto.Address, filter bool) (int, error) {
	key := accountOriginalEstimatorVersionKey{addr}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes(), filter)
	if err != nil {
		return 0, err
	}
	record := new(estimatorVersionRecord)
	if err := cbor.Unmarshal(recordBytes, record); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal original estimator version record")
	}
	return int(record.Version), nil
}

func (sc *scriptsComplexity) newestOriginalScriptComplexityByAddr(addr proto.Address, filter bool) (*ride.TreeEstimation, error) {
	ev, err := sc.originalEstimatorVersion(addr, filter)
	if err != nil {
		return nil, err
	}
	return sc.newestScriptComplexityByAddr(addr, ev, filter)
}

func (sc *scriptsComplexity) newestScriptComplexityByAsset(asset crypto.Digest, filter bool) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.newestTopEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	record := new(ride.TreeEstimation)
	if err = cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal asset script complexities record")
	}
	return record, nil
}

func (sc *scriptsComplexity) scriptComplexityByAsset(asset crypto.Digest, filter bool) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	record := new(ride.TreeEstimation)
	if err := cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal asset script complexities record")
	}
	return record, nil
}

func (sc *scriptsComplexity) scriptComplexityByAddress(addr proto.Address, ev int, filter bool) (*ride.TreeEstimation, error) {
	key := accountScriptComplexityKey{ev, addr}
	recordBytes, err := sc.hs.topEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	record := new(ride.TreeEstimation)
	if err := cbor.Unmarshal(recordBytes, record); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account script complexities record")
	}
	return record, nil
}

func (sc *scriptsComplexity) saveComplexitiesForAddr(addr proto.Address, estimations map[int]ride.TreeEstimation, blockID proto.BlockID) error {
	min := math.MaxUint8
	for v, e := range estimations {
		if v < min {
			min = v
		}
		recordBytes, err := cbor.Marshal(e)
		if err != nil {
			return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'", addr.String(), blockID.String())
		}
		key := accountScriptComplexityKey{v, addr}
		err = sc.hs.addNewEntry(accountScriptComplexity, key.bytes(), recordBytes, blockID)
		if err != nil {
			return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'", addr.String(), blockID.String())
		}
	}
	key := accountOriginalEstimatorVersionKey{addr}
	record := estimatorVersionRecord{uint8(min)}
	recordBytes, err := cbor.Marshal(record)
	if err != nil {
		return errors.Wrapf(err, "failed to save original estimator version for address '%s' in block '%s'", addr.String(), blockID.String())
	}
	err = sc.hs.addNewEntry(accountOriginalEstimatorVersion, key.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to save original estimator version for address '%s' in block '%s'", addr.String(), blockID.String())
	}
	return nil
}

func (sc *scriptsComplexity) saveComplexitiesForAsset(asset crypto.Digest, estimation ride.TreeEstimation, blockID proto.BlockID) error {
	recordBytes, err := cbor.Marshal(estimation)
	if err != nil {
		return errors.Wrapf(err, "failed to save complexity record for asset '%s' in block '%s'", asset.String(), blockID.String())
	}
	key := assetScriptComplexityKey{asset}
	err = sc.hs.addNewEntry(assetScriptComplexity, key.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to save complexity record for asset '%s' in block '%s'", asset.String(), blockID.String())
	}
	return nil
}
