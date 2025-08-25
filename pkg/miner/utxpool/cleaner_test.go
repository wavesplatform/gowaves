package utxpool

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestNewCleaner(t *testing.T) {
	require.NotNil(t, NewCleaner(nil, nil, nil, proto.TestNetScheme))
}
