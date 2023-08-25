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

// newestScriptComplexityByAddr returns estimated script complexity by the given address.
// Note that verifier complexity remains unchanged even after callables updates.
func (sc *scriptsComplexity) newestScriptComplexityByAddr(addr proto.Address) (*ride.TreeEstimation, error) {
	r, err := sc.newestScriptEstimationRecordByAddr(addr)
	if err != nil {
		return nil, err
	}
	return &r.Estimation, nil
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
	var recordBytes []byte
	if !se.scriptIsEmpty { // fill recordBytes if script is not empty
		var err error
		// prepare record bytes
		complexityRecord := estimationRecord{
			EstimatorVersion: uint8(se.currentEstimatorVersion),
			Estimation:       se.estimation,
		}
		recordBytes, err = complexityRecord.marshalBinary()
		if err != nil {
			return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'",
				addr.String(), blockID.String(),
			)
		}
	}
	complexityKey := accountScriptComplexityKey{addr.ID()}
	// save new estimation record or write zero bytes if script is empty
	err := sc.hs.addNewEntry(accountScriptComplexity, complexityKey.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to save complexities record for address '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	return nil
}

func (sc *scriptsComplexity) updateCallableComplexitiesForAddr(
	addr proto.Address,
	se scriptEstimation,
	blockID proto.BlockID,
) error {
	old, err := sc.newestScriptEstimationRecordByAddr(addr)
	if err != nil {
		return errors.Wrapf(err, "failed to update callable complexities for addr '%s'", addr)
	}
	newEst := ride.TreeEstimation{
		Estimation: maxEstimationWithOldVerifierComplexity(se.estimation, old.Estimation.Verifier),
		Verifier:   old.Estimation.Verifier,
		Functions:  se.estimation.Functions,
	}
	// prepare record bytes
	complexityRecord := estimationRecord{
		EstimatorVersion: uint8(se.currentEstimatorVersion),
		Estimation:       newEst,
	}
	recordBytes, err := complexityRecord.marshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal complexities record for address '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	complexityKey := accountScriptComplexityKey{addr.ID()}
	// save new estimation record
	err = sc.hs.addNewEntry(accountScriptComplexity, complexityKey.bytes(), recordBytes, blockID)
	if err != nil {
		return errors.Wrapf(err, "failed to update complexities record for address '%s' in block '%s'",
			addr.String(), blockID.String(),
		)
	}
	return nil
}

func maxEstimationWithOldVerifierComplexity(update ride.TreeEstimation, oldVerifierComplexity int) int {
	if update.Verifier < update.Estimation { // fast path: one's callable complexity == max update complexity
		return max(oldVerifierComplexity, update.Estimation)
	}
	// trying to find the hardest callable in the tab of new estimations and determine max complexity
	maxCallableComplexity := 0
	for _, complexity := range update.Functions { // for account scripts (verifier) Functions map is empty
		if complexity > maxCallableComplexity {
			maxCallableComplexity = complexity
		}
	}
	return max(oldVerifierComplexity, maxCallableComplexity)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
