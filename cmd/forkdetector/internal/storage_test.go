package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestPeerKeyBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		ip    net.IP
		nonce uint64
	}{
		{net.IPv4(127, 0, 0, 1), 1234567890},
		{net.IPv4(8, 8, 8, 8), 0},
		{net.IPv4(1, 2, 3, 4), math.MaxUint64},
	}

	for _, tc := range tests {
		k := peerKey{prefix: peersPrefix, ip: tc.ip, nonce: tc.nonce}
		b := k.bytes()
		var ak peerKey
		if err := ak.fromByte(b); assert.NoError(t, err) {
			assert.Equal(t, peersPrefix, ak.prefix)
			assert.Equal(t, k.ip.To4(), ak.ip.To4())
			assert.Equal(t, k.nonce, ak.nonce)
		}
	}
}

func TestPeerInfoBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		port    uint16
		name    string
		version string
		block   string
		last    uint64
	}{
		{12345, "super puper node", "1.2.3", "3ikyAPfJ3HMGR48M4ULYWrHV6g4oW7eDToyagEjZhmFWDmGcL8trhWwjVbjo1Ykq4AB1EG9LUnTVSQ54iWyN7Gwx", 1234567890},
		{0, "", "0.0.0", "3ikyAPfJ3HMGR48M4ULYWrHV6g4oW7eDToyagEjZhmFWDmGcL8trhWwjVbjo1Ykq4AB1EG9LUnTVSQ54iWyN7Gwx", 1234567890},
	}
	for _, tc := range tests {
		v, err := proto.NewVersionFromString(tc.version)
		require.NoError(t, err)
		s, err := crypto.NewSignatureFromBase58(tc.block)
		require.NoError(t, err)
		i := peerInfo{port:tc.port, name:tc.name, version:*v, block:s, last:tc.last}
		b := i.bytes()
		var ai peerInfo
		if err := ai.fromBytes(b); assert.NoError(t, err) {
			assert.Equal(t, tc.port, ai.port)
			assert.Equal(t, tc.name, ai.name)
			assert.Equal(t, *v, ai.version)
			assert.ElementsMatch(t, s, ai.block)
			assert.Equal(t, tc.last, ai.last)
		}
	}
}

func openDB(t *testing.T, name string) (*leveldb.DB, func()) {
	path := filepath.Join(os.TempDir(), name)
	opts := opt.Options{ErrorIfExist: true}
	db, err := leveldb.OpenFile(path, &opts)
	assert.NoError(t, err)
	return db, func() {
		err = db.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(path)
		assert.NoError(t, err)
	}
}
