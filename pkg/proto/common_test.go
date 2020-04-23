package proto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
)

func TestStringWithUInt16LenBinaryRoundTrip(t *testing.T) {
	tests := []string{
		"",
		"a",
		"hello world",
	}
	for _, tc := range tests {
		buf := make([]byte, 2+len(tc))
		PutStringWithUInt16Len(buf, tc)
		s, err := StringWithUInt16Len(buf)
		assert.NoError(t, err)
		assert.Equal(t, tc, s)
	}
}

func TestSerializer_StringWithUInt16LenBinary(t *testing.T) {
	tests := []string{
		"",
		"a",
		"hello world",
	}
	for _, tc := range tests {
		buf := new(bytes.Buffer)
		ser := serializer.New(buf)
		_ = ser.StringWithUInt16Len(tc)
		s, err := StringWithUInt16Len(buf.Bytes())
		assert.NoError(t, err)
		assert.Equal(t, tc, s)
	}
}

func TestBoolBinaryRoundTrip(t *testing.T) {
	tests := []bool{true, false}
	for _, b := range tests {
		buf := make([]byte, 1)
		PutBool(buf, b)
		v, err := Bool(buf)
		assert.NoError(t, err)
		assert.Equal(t, b, v)
	}
}
