package storage_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestBinaryStorage_Known(t *testing.T) {
	d, err := ioutil.TempDir("", "abc")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	err = os.Mkdir(path.Join(d, "blocks_storage"), 0755)
	require.NoError(t, err)

	s := storage.NewBinaryStorage(d)
	known, err := s.Known()
	require.NoError(t, err)
	require.Len(t, known, 0)

	err = s.AddKnown(proto.NewTCPAddrFromString("127.0.0.1:6868"))
	require.NoError(t, err)

	// should return 1 peer
	known, err = s.Known()
	require.NoError(t, err)
	require.Len(t, known, 1)

	// add duplicate peer
	err = s.AddKnown(proto.NewTCPAddrFromString("127.0.0.1:6868"))
	require.NoError(t, err)

	// should return 1 peer too
	known, err = s.Known()
	require.NoError(t, err)
	require.Len(t, known, 1)
}

func TestBinaryStorage_All(t *testing.T) {
	d, err := ioutil.TempDir("", "all")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	err = os.Mkdir(path.Join(d, "blocks_storage"), 0755)
	require.NoError(t, err)

	s := storage.NewBinaryStorage(d)
	known, err := s.All()
	require.NoError(t, err)
	require.Len(t, known, 0)

	err = s.Add([]proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")})
	require.NoError(t, err)

	// should return 1 peer
	known, err = s.All()
	require.NoError(t, err)
	require.Len(t, known, 1)

	// add duplicate peer
	err = s.Add([]proto.TCPAddr{proto.NewTCPAddrFromString("127.0.0.1:6868")})
	require.NoError(t, err)

	// should return 1 peer too
	known, err = s.All()
	require.NoError(t, err)
	require.Len(t, known, 1)
}
