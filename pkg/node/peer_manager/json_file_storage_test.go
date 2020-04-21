package peer_manager

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestJsonFileStorage_Peers(t *testing.T) {
	d, err := ioutil.TempDir("", "abc")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	err = os.Mkdir(path.Join(d, "blocks_storage"), 0755)
	require.NoError(t, err)

	s, err := NewJsonFileStorage(d)
	require.NoError(t, err)
	peers, err := s.Peers()
	require.NoError(t, err)
	require.Len(t, peers, 0)

	addrs := []proto.TCPAddr{
		proto.NewTCPAddrFromString("127.0.0.1:8080"),
		proto.NewTCPAddrFromString("0.0.0.0:6862"),
	}

	err = s.SavePeers(addrs)
	require.NoError(t, err)

	s, err = NewJsonFileStorage(d)
	require.NoError(t, err)
	peers, err = s.Peers()
	require.NoError(t, err)
	require.Equal(t, addrs, peers)
}
