package proto

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringWithUInt16LenBinaryRoundTrip(t *testing.T) {
	tests := []string{
		"",
		"a",
		"sdlfjsalktjerqoitjg asjfdg",
	}
	for _, tc := range tests {
		buf := make([]byte, 2+len(tc))
		PutStringWithUInt16Len(buf, tc)
		s, err := StringWithUInt16Len(buf)
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
