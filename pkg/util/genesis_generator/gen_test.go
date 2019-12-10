package genesis_generator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestGenerate(t *testing.T) {
	rs, err := Generate(1558516864282, 'W', proto.MustKeyPair([]byte("test")), 9000000000000000)
	require.NoError(t, err)
	require.Equal(t, 1, rs.TransactionCount)
}
