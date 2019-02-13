package utils

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
	"time"
)

func TestKnownPeers_SaveDisk(t *testing.T) {
	fs := afero.NewMemMapFs()

	s, err := NewFileBasedStorage(fs, "/known_peers.json")
	require.NoError(t, err)

	knownPeers, err := NewKnownPeersInterval(s, 1*time.Second)
	require.NoError(t, err)
	knownPeers.Add(proto.PeerInfo{Addr: net.IPv4(10, 10, 10, 10), Port: 90}, proto.Version{})
	require.NoError(t, knownPeers.save())
	knownPeers.exitWithoutSave()

	f, err := fs.Open("/known_peers.json")
	require.NoError(t, err)
	b, err := afero.ReadAll(f)
	require.NoError(t, err)
	assert.Contains(t, string(b), "10.10.10.10:90")
}
