package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type invokeResultRecord struct {
	res *proto.ScriptResult
}

func (r *invokeResultRecord) marshalBinary() ([]byte, error) {
	resBytes, err := r.res.MarshalWithAddresses()
	if err != nil {
		return nil, err
	}
	return resBytes, nil
}

func (r *invokeResultRecord) unmarshalBinary(data []byte) error {
	var res proto.ScriptResult
	if err := res.UnmarshalWithAddresses(data); err != nil {
		return err
	}
	r.res = &res
	return nil
}

type invokeResults struct {
	hs      *historyStorage
	aliases *aliases
}

func newInvokeResults(hs *historyStorage, aliases *aliases) (*invokeResults, error) {
	return &invokeResults{hs, aliases}, nil
}

func (ir *invokeResults) invokeResult(invokeID crypto.Digest, filter bool) (*proto.ScriptResult, error) {
	key := invokeResultKey{invokeID}
	recordBytes, err := ir.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return nil, err
	}
	var record invokeResultRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal invoke result: %v\n", err)
	}
	return record.res, nil
}

func (ir *invokeResults) saveResult(invokeID crypto.Digest, res *proto.ScriptResult, blockID crypto.Signature) error {
	key := invokeResultKey{invokeID}
	record := &invokeResultRecord{res}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := ir.hs.addNewEntry(invokeResult, key.bytes(), recordBytes, blockID); err != nil {
		return err
	}
	return nil
}
