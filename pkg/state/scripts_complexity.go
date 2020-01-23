package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	assetScriptComplexityRecordSize = 9
)

type assetScriptComplexityRecord struct {
	complexity uint64
	estimator  byte // estimator version used to calculate complexity
}

func (r *assetScriptComplexityRecord) marshalBinary() ([]byte, error) {
	buf := make([]byte, assetScriptComplexityRecordSize)
	binary.BigEndian.PutUint64(buf[:8], r.complexity)
	buf[8] = r.estimator
	return buf, nil
}

func (r *assetScriptComplexityRecord) unmarshalBinary(data []byte) error {
	if len(data) != assetScriptComplexityRecordSize {
		return errInvalidDataSize
	}
	r.complexity = binary.BigEndian.Uint64(data[:8])
	r.estimator = data[8]
	return nil
}

type accountScriptComplexityRecord struct {
	verifierComplexity uint64
	byFuncs            map[string]uint64
	estimator          byte // estimator version
}

func newAccountScriptComplexityRecord() accountScriptComplexityRecord {
	return accountScriptComplexityRecord{byFuncs: make(map[string]uint64)}
}

func (r *accountScriptComplexityRecord) binarySize() uint64 {
	// Version + verifierComplexity.
	size := uint64(9)
	for funcName := range r.byFuncs {
		// Function name length.
		size += 2
		// Name itself + value.
		size += uint64(len(funcName)) + 8
	}
	return size
}

func (r *accountScriptComplexityRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, r.binarySize())
	binary.BigEndian.PutUint64(res[:8], r.verifierComplexity)
	pos := 8
	for funcName, complexity := range r.byFuncs {
		shift := 2 + len(funcName)
		proto.PutStringWithUInt16Len(res[pos:pos+shift], funcName)
		pos += shift
		binary.BigEndian.PutUint64(res[pos:pos+8], complexity)
		pos += 8
	}
	res[pos] = r.estimator
	return res, nil
}

func (r *accountScriptComplexityRecord) unmarshalBinary(data []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = errInvalidDataSize
		}
	}()

	r.verifierComplexity = binary.BigEndian.Uint64(data[:8])
	pos := 8
	for pos < len(data)-1 {
		funcName, err := proto.StringWithUInt16Len(data[pos:])
		if err != nil {
			return err
		}
		pos += len(funcName) + 2
		value := binary.BigEndian.Uint64(data[pos : pos+8])
		pos += 8
		r.byFuncs[funcName] = value
	}
	r.estimator = data[pos]
	return nil
}

type scriptsComplexity struct {
	hs *historyStorage
}

func newScriptsComplexity(hs *historyStorage) (*scriptsComplexity, error) {
	return &scriptsComplexity{hs: hs}, nil
}

func (sc *scriptsComplexity) newestScriptComplexityByAddr(addr proto.Address, filter bool) (*accountScriptComplexityRecord, error) {
	key := accountScriptComplexityKey{addr}
	recordBytes, err := sc.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	record := newAccountScriptComplexityRecord()
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal account script complexity record: %v\n", err)
	}
	return &record, nil
}

func (sc *scriptsComplexity) newestScriptComplexityByAsset(asset crypto.Digest, filter bool) (*assetScriptComplexityRecord, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record assetScriptComplexityRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal asset script complexity record: %v\n", err)
	}
	return &record, nil
}

func (sc *scriptsComplexity) scriptComplexityByAsset(asset crypto.Digest, filter bool) (*assetScriptComplexityRecord, error) {
	key := assetScriptComplexityKey{asset}
	recordBytes, err := sc.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record assetScriptComplexityRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal asset script complexity record: %v\n", err)
	}
	return &record, nil
}

func (sc *scriptsComplexity) scriptComplexityByAddress(addr proto.Address, filter bool) (*accountScriptComplexityRecord, error) {
	key := accountScriptComplexityKey{addr}
	recordBytes, err := sc.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record accountScriptComplexityRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal account script complexity record: %v\n", err)
	}
	return &record, nil
}

func (sc *scriptsComplexity) saveComplexityForAddr(addr proto.Address, record *accountScriptComplexityRecord, blockID crypto.Signature) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	key := accountScriptComplexityKey{addr}
	return sc.hs.addNewEntry(accountScriptComplexity, key.bytes(), recordBytes, blockID)
}

func (sc *scriptsComplexity) saveComplexityForAsset(asset crypto.Digest, record *assetScriptComplexityRecord, blockID crypto.Signature) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	key := assetScriptComplexityKey{asset}
	return sc.hs.addNewEntry(assetScriptComplexity, key.bytes(), recordBytes, blockID)
}
