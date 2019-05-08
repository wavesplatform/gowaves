package settings

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFromEnvironString(t *testing.T) {
	settings := &NodeSettings{}
	FromJavaEnvironString(settings, "-Dwaves.miner.quorum=0 -Dwaves.network.node-name=node01 -Dwaves.wallet.seed=wzd2MzQ8-Dlogback.stdout.level=TRACE -Dlogback.file.level=OFF -Dwaves.network.declared-address=10.147.77.193:6863")
	require.Equal(t, "10.147.77.193:6863", settings.DeclaredAddr)
}
