package ride

import (
	"bytes"
	"encoding/gob"

	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type AnyScriptInvocationStateWithHeight struct {
	Height proto.Height
	State  AnyScriptInvocationState
}

type AnyScriptInvocationState struct {
	Err    error
	Result Result
}

func (s *AnyScriptInvocationState) MarshalBinary() ([]byte, error) {
	var transformedErr error
	if s.Err != nil {
		switch err := s.Err.(type) {
		case evaluationError:
			err.originalError = errorString{err.originalError.Error()}
			transformedErr = err
		default:
			transformedErr = errorString{err.Error()}
		}
	}
	// TODO: check performance benefit of using bytebufferpool here
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)
	if err := gob.NewEncoder(b).Encode(&AnyScriptInvocationState{Result: s.Result, Err: transformedErr}); err != nil {
		return nil, err
	}
	return append([]byte(nil), b.Bytes()...), nil
}

func (s *AnyScriptInvocationState) UnmarshalBinary(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(s)
}

// errorString is a simple error type which should be used only for error conversion inside AnyScriptInvocationState.MarshalBinary
type errorString struct {
	Message string
}

func (e errorString) Error() string {
	return e.Message
}

// TODO: consider using of sync.Once with initGOBForAnyScriptInvocationState inside marshal and unmarshal of AnyScriptInvocationState?
func init() {
	initGOBForAnyScriptInvocationState()
}

func initGOBForAnyScriptInvocationState() {
	types := [...]interface{}{
		// errors
		error(evaluationError{}),
		error(errorString{}),
		// ok results
		// in these results no need to marshal/unmarshal DAppResult.param or ScriptResult.param
		// because we use it only in invoke() or reentrantInvoke(), so it's always nil in the end of script/DApp execution
		Result(DAppResult{}),
		Result(ScriptResult{}),
		// Actions
		proto.ScriptAction(&proto.AttachedPaymentScriptAction{}),
		proto.ScriptAction(&proto.TransferScriptAction{}),
		proto.ScriptAction(&proto.IssueScriptAction{}),
		proto.ScriptAction(&proto.ReissueScriptAction{}),
		proto.ScriptAction(&proto.BurnScriptAction{}),
		proto.ScriptAction(&proto.SponsorshipScriptAction{}),
		proto.ScriptAction(&proto.LeaseScriptAction{}),
		proto.ScriptAction(&proto.LeaseCancelScriptAction{}),
		proto.ScriptAction(&proto.DataEntryScriptAction{}),
		// Entries
		proto.DataEntry(&proto.IntegerDataEntry{}),
		proto.DataEntry(&proto.BooleanDataEntry{}),
		proto.DataEntry(&proto.StringDataEntry{}),
		proto.DataEntry(&proto.BinaryDataEntry{}),
		proto.DataEntry(&proto.DeleteDataEntry{}),
	}
	for _, t := range &types {
		gob.Register(t)
	}
}
