package ride

import (
	"encoding/binary"
	"encoding/gob"
	"io"
	"math"

	"github.com/pkg/errors"
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

func (s *AnyScriptInvocationState) marshalTo(enc *gob.Encoder) error {
	var transformedErr error
	if s.Err != nil {
		switch err := s.Err.(type) {
		case evaluationError:
			err.OriginalError = errorString{err.OriginalError.Error()}
			transformedErr = err
		default:
			transformedErr = errorString{err.Error()}
		}
	}
	return enc.Encode(&AnyScriptInvocationState{Result: s.Result, Err: transformedErr})
}

type AnyScriptInvocationStates []AnyScriptInvocationState

func (s AnyScriptInvocationStates) MarshalBinary() ([]byte, error) {
	if l := len(s); l > math.MaxUint16 {
		return nil, errors.Errorf("too big AnyScriptInvocationStates slice size: got %d, max %d", l, math.MaxUint16)
	}
	var count [2]byte
	binary.BigEndian.PutUint16(count[:], uint16(len(s)))
	// TODO: check performance benefit of using bytebufferpool here
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)
	if _, err := b.Write(count[:]); err != nil {
		return nil, err
	}
	enc := gob.NewEncoder(b)
	for _, state := range s {
		if err := state.marshalTo(enc); err != nil {
			return nil, err
		}
	}
	out := make([]byte, b.Len())
	copy(out, b.Bytes())
	return out, nil
}

func (s *AnyScriptInvocationStates) UnmarshalFrom(r io.Reader) error {
	var countBytes [2]byte
	if _, err := io.ReadFull(r, countBytes[:]); err != nil {
		return err
	}
	count := binary.BigEndian.Uint16(countBytes[:])
	states := make(AnyScriptInvocationStates, 0, count)
	dec := gob.NewDecoder(r)
	for i := uint16(0); i < count; i++ {
		var state AnyScriptInvocationState
		if err := dec.Decode(&state); err != nil {
			return err
		}
		states = append(states, state)
	}
	*s = states
	return nil
}

// errorString is a simple error type which should be used only for error conversion inside AnyScriptInvocationState.marshalTo
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
	// TODO: implement comprehensive binary serialization and deserialization for AnyScriptInvocationState type
	types := [...]interface{}{
		// errors
		error(evaluationError{}),
		error(errorString{}),
		// ok results
		// in these results no need to marshal/unmarshal dAppResult.param or scriptExecutionResult.param
		// because we use it only in invoke() or reentrantInvoke(), so it's always nil in the end of script/DApp execution
		Result(dAppResult{}),
		Result(scriptExecutionResult{}),
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
