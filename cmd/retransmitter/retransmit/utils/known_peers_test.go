package utils_test

import (
	"net"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestKnownPeers(t *testing.T) {
	fs := afero.NewMemMapFs()

	s, err := utils.NewFileBasedStorage(fs, "/known_peers.json")
	require.NoError(t, err)

	knownPeers, err := utils.NewKnownPeersInterval(s, 1*time.Second)
	require.NoError(t, err)
	defer knownPeers.Stop()
	knownPeers.Add(proto.NewTCPAddr(net.IPv4(10, 10, 10, 10), 90), proto.Version{})

	assert.Equal(t, []string{"10.10.10.10:90"}, knownPeers.GetAll())
	assert.Len(t, knownPeers.Addresses(), 1)
}
