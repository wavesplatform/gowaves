package internal

import (
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"testing"
	"time"
)

func TestRegisterNewAddresses(t *testing.T) {
	db, closeDB := openDB(t, "fd-registry-1")
	defer closeDB()

	storage := &storage{db: db}

	versions := []proto.Version{{Minor: 1}, {Minor: 2}, {Minor: 3}, {Minor: 4}, {Minor: 5}}

	registry := NewPublicAddressRegistry(storage, time.Second, time.Second, versions)
	addresses := []net.TCPAddr{
		{IP: net.ParseIP("1.2.3.4"), Port: 12345},
		{IP: net.ParseIP("4.3.2.1"), Port: 23456},
		{IP: net.ParseIP("4.3.2.1"), Port: 23456},
		{IP: net.ParseIP("127.0.0.1"), Port: 4444},
		{IP: net.ParseIP("8.8.8.8"), Port: 8080},
	}
	n, err := registry.RegisterNewAddresses(addresses)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	pas1, err := registry.FeasibleAddresses()
	require.NoError(t, err)
	assert.Equal(t, 4, len(pas1))
	m := make(map[uint64]struct{})
	for _, pa := range pas1 {
		assert.Equal(t, proto.Version{Minor: 5}, pa.version)
		m[registry.hashTCPAddr(pa.address)] = struct{}{}
	}
	assert.Equal(t, 4, len(m))

	pas2, err := registry.FeasibleAddresses()
	require.NoError(t, err)
	assert.Equal(t, 0, len(pas2))
}
