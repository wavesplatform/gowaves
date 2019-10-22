package serializer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonFallableSerializer_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	o := bytes.NewBuffer([]byte{1, 2, 3, 4, 5})
	s := NewNonFallable(buf)
	_, _ = o.WriteTo(s)

	require.EqualValues(t, 5, s.N())
	require.Equal(t, []byte{1, 2, 3, 4, 5}, buf.Bytes())
}
