package state

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

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

func (sc *scriptsComplexity) newestScriptComplexityByAsset(asset crypto.Digest, ev int, filter bool) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{ev, asset}
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

func (sc *scriptsComplexity) scriptComplexityByAsset(asset crypto.Digest, ev int, filter bool) (*ride.TreeEstimation, error) {
	key := assetScriptComplexityKey{ev, asset}
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
	for v, e := range estimations {
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
	return nil
}

func (sc *scriptsComplexity) saveComplexitiesForAsset(asset crypto.Digest, estimations map[int]ride.TreeEstimation, blockID proto.BlockID) error {
	for v, e := range estimations {
		recordBytes, err := cbor.Marshal(e)
		if err != nil {
			return errors.Wrapf(err, "failed to save complexities record for asset '%s' in block '%s'", asset.String(), blockID.String())
		}
		key := assetScriptComplexityKey{v, asset}
		err = sc.hs.addNewEntry(assetScriptComplexity, key.bytes(), recordBytes, blockID)
		if err != nil {
			return errors.Wrapf(err, "failed to save complexities record for asset '%s' in block '%s'", asset.String(), blockID.String())
		}
	}
	return nil
}
