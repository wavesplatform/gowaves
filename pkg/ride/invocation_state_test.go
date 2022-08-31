package ride

import (
	"encoding"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// TODO: These tests check crunches for gob. See TODOs with proto.DataEntry.MarshalEntry and proto.DataEntry.UnmarshalEntry methods
// For convenient type deserialization to interface directly from binary we use gob package.
// gob uses encoding.BinaryMarshaler or encoding.BinaryUnmarshaler instead of own encoding if type implements these interfaces.
// In such case gob decoder panics because of nil proto.DataEntry interface value.

func TestDataEntryDoesNotImplementBinaryMarshalerInterface(t *testing.T) {
	_, ok := proto.DataEntry(nil).(encoding.BinaryMarshaler)
	require.False(t, ok)
}

func TestDataEntryDoesNotImplementBinaryUnmarshalerInterface(t *testing.T) {
	_, ok := proto.DataEntry(nil).(encoding.BinaryUnmarshaler)
	require.False(t, ok)
}
