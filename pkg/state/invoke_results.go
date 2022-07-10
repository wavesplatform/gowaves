package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
	pb "google.golang.org/protobuf/proto"
)

type invokeResultRecord struct {
	res *proto.ScriptResult
}

func (r *invokeResultRecord) marshalBinary() ([]byte, error) {
	msg, err := r.res.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return pb.Marshal(msg)
}

func (r *invokeResultRecord) unmarshalBinary(scheme byte, data []byte) error {
	msg := new(g.InvokeScriptResult)
	if err := pb.Unmarshal(data, msg); err != nil {
		return err
	}
	var res proto.ScriptResult
	if err := res.FromProtobuf(scheme, msg); err != nil {
		return err
	}
	r.res = &res
	return nil
}

type invokeResults struct {
	hs *historyStorage
}

func newInvokeResults(hs *historyStorage) *invokeResults {
	return &invokeResults{hs}
}

func (ir *invokeResults) invokeResult(scheme byte, invokeID crypto.Digest) (*proto.ScriptResult, error) {
	key := invokeResultKey{invokeID}
	recordBytes, err := ir.hs.topEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var record invokeResultRecord
	if err := record.unmarshalBinary(scheme, recordBytes); err != nil {
		return nil, errors.Errorf("failed to unmarshal invoke result: %v\n", err)
	}
	return record.res, nil
}

func (ir *invokeResults) saveResult(invokeID crypto.Digest, res *proto.ScriptResult, blockID proto.BlockID) error {
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
