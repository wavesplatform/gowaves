package internal

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestRegisterNewAddresses(t *testing.T) {
	db, closeDB := openDB(t, "fd-registry-1")
	defer closeDB()

	storage := &storage{db: db}

	versions := []proto.Version{
		proto.NewVersion(0, 1, 0),
		proto.NewVersion(0, 2, 0),
		proto.NewVersion(0, 3, 0),
		proto.NewVersion(0, 4, 0),
		proto.NewVersion(0, 5, 0),
	}

	addr := &net.TCPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 1234}
	registry := NewRegistry('T', addr, versions, storage)
	addresses := []net.TCPAddr{
		{IP: net.ParseIP("1.2.3.4"), Port: 12345},
		{IP: net.ParseIP("4.3.2.1"), Port: 23456},
		{IP: net.ParseIP("4.3.2.1"), Port: 23456},
		{IP: net.ParseIP("127.0.0.1"), Port: 4444},
		{IP: net.ParseIP("8.8.8.8"), Port: 8080},
	}
	n := registry.AppendAddresses(addresses)
	assert.Equal(t, 4, n)
	pas1, err := registry.TakeAvailableAddresses()
	require.NoError(t, err)
	assert.Equal(t, 4, len(pas1))
	m := make(map[uint64]struct{})
	for _, pa := range pas1 {
		v, err := registry.SuggestVersion(pa)
		require.NoError(t, err)
		assert.Equal(t, proto.NewVersion(0, 5, 0), v)
		ip, _, err := splitAddr(pa)
		require.NoError(t, err)
		m[hash(ip)] = struct{}{}
	}
	assert.Equal(t, 4, len(m))

	for _, pa := range pas1 {
		err := registry.PeerDiscarded(pa)
		require.NoError(t, err)
	}
	pas2, err := registry.TakeAvailableAddresses()
	require.NoError(t, err)
	assert.Equal(t, 0, len(pas2))
}
