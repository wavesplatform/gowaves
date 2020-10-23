package ride

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Check that patches correctly.
func TestBuilderPatch(t *testing.T) {
	b := newBuilder()

	b.bool(true)
	patchStart := b.writeStub(2)
	b.bool(true)

	b.patch(patchStart, []byte{0xff, 0xff})

	require.Equal(t, b.bytes(), []byte{OpTrue, 0xff, 0xff, OpTrue})

}
