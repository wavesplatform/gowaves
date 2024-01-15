package state

import (
	"github.com/opencontainers/go-digest"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestPushOneBlock(t *testing.T) {
	wa1, err := proto.NewAddressFromString("3MuhGCajV9HXunkyuQpwXvHTjTLaMy93g9Y")
	require.NoError(t, err)
	id1, err := proto.NewBlockIDFromBase58("6nxfNczjJh8gU25746FNX8qWkw6wVsKssJotvxTaUi2z")
	require.NoError(t, err)
	id2, err := proto.NewBlockIDFromBase58("7ZhWmPmpD8EGwFP9mhhipwLCfnXF5B1gbfHu6VEGqDq9")
	require.NoError(t, err)

	sh := newStateHasher()

	c1 := &wavesRecordForHashes{addr: &wa1, balance: 12345}
	c2 := &wavesRecordForHashes{addr: &wa1, balance: 67890}

	err = sh.push("key1", c1, id1)
	require.NoError(t, err)

	err = sh.push("key2", c2, id2)
	require.NoError(t, err)

	err = sh.stop()
	require.NoError(t, err)
	h1 := sh.stateHashAt(id1)
	assert.NotEqual(t, sh.emptyHash, h1)
	h2 := sh.stateHashAt(id2)
	assert.NotEqual(t, sh.emptyHash, h2)

	sh.reset()

	err = sh.push("key1", c1, id1)
	require.NoError(t, err)

	err = sh.stop()
	require.NoError(t, err)
	assert.Equal(t, h1, sh.stateHashAt(id1))
	assert.Equal(t, sh.emptyHash, sh.stateHashAt(id2))
}

// TODO do the actual initial balances in appender

func TestLegacyStateHashSupport(t *testing.T) {
	initialBalances := newInitialBalancesInBlock()

	snapshots1 := []proto.AtomicSnapshot {
		&proto.WavesBalanceSnapshot{},
	}

	snapshots := []proto.AtomicSnapshot {
			&proto.WavesBalanceSnapshot{Address:testAddr ,Balance: 1},
			&proto.WavesBalanceSnapshot{Address: same ,Balance: 5},
			&proto.WavesBalanceSnapshot{Address: ,Balance: 3},
			&proto.WavesBalanceSnapshot{Address: same ,Balance: 5},

			&proto.AssetBalanceSnapshot{
				Address: proto.WavesAddress{},
				AssetID: crypto.MustDigestFromBase58(assetStr),
				Balance: 3,
			},
			&proto.AssetBalanceSnapshot{
				Address: proto.WavesAddress{},
				AssetID: crypto.MustDigestFromBase58(assetStr),
				Balance: 0,
			},
			proto.AssetBalanceSnapshot{
				Address: proto.WavesAddress{},
				AssetID: crypto.MustDigestFromBase58(assetStr),
				Balance: 3,
			},

			proto.LeaseBalanceSnapshot{
				Address: proto.MustAddressFromString(testAddr),
				LeaseIn: 10,
				LeaseOut: 10,
			},
			proto.LeaseCancel{
				Address: proto.WavesAddress{},
				AssetID: crypto.Digest{},
				Balance: 0,
			},

			proto.LeaseBalanceSnapshot{
				Address: proto.WavesAddress{},
				AssetID: crypto.Digest{},
				Balance: 0,
			},
	}

	initialBalances.addIfNotExists(txSnapshots.regular)

}
