package state

import (
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
	to := createStorageObjects(t, true)
	// Add some entries and flush.
	to.addBlock(t, blockID0)
	to.entities.balances.calculateHashes = true

	err := to.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(), wavesValue{
		profile: balanceProfile{
			balance:  5,
			leaseIn:  0,
			leaseOut: 0,
		},
		leaseChange:   false,
		balanceChange: false,
	}, blockID0)
	assert.NoError(t, err)

	err = to.entities.balances.setAssetBalance(testGlobal.senderInfo.addr.ID(), proto.AssetIDFromDigest(testGlobal.asset0.assetID), 3, blockID0)
	assert.NoError(t, err)

	snapshotsSetFirst := []proto.AtomicSnapshot{
		&proto.WavesBalanceSnapshot{Address: proto.MustAddressFromString(testGlobal.issuerInfo.Address().String()), Balance: 1},
		&proto.WavesBalanceSnapshot{Address: testGlobal.senderInfo.addr, Balance: 3},
		&proto.WavesBalanceSnapshot{Address: testGlobal.recipientInfo.addr, Balance: 5},

		&proto.AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: crypto.MustDigestFromBase58(assetStr),
			Balance: 3,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: crypto.MustDigestFromBase58(assetStr),
			Balance: 0,
		},
		proto.AssetBalanceSnapshot{
			Address: testGlobal.senderInfo.addr,
			AssetID: crypto.MustDigestFromBase58(assetStr),
			Balance: 3,
		},
	}
	err = to.entities.balances.addInitialBalancesIfNotExists(snapshotsSetFirst)
	assert.NoError(t, err)

	to.entities.balances.addWavesBalanceChangeLegacySH(testGlobal.issuerInfo.Address().ID(), 1)
	to.entities.balances.addWavesBalanceChangeLegacySH(testGlobal.senderInfo.Address().ID(), 3)
	to.entities.balances.addWavesBalanceChangeLegacySH(testGlobal.recipientInfo.Address().ID(), 5)

	to.entities.balances.addAssetBalanceChangeLegacySH(testGlobal.senderInfo.Address().ID(), proto.AssetIDFromDigest(crypto.MustDigestFromBase58(assetStr)), 3)
	to.entities.balances.addAssetBalanceChangeLegacySH(testGlobal.issuerInfo.Address().ID(), proto.AssetIDFromDigest(crypto.MustDigestFromBase58(assetStr)), 0)
	to.entities.balances.addAssetBalanceChangeLegacySH(testGlobal.senderInfo.Address().ID(), proto.AssetIDFromDigest(crypto.MustDigestFromBase58(assetStr)), 3)

	snapshotsSetSecond := []proto.AtomicSnapshot{
		&proto.NewLeaseSnapshot{
			LeaseID:       crypto.MustDigestFromBase58(invokeId),
			Amount:        5,
			SenderPK:      testGlobal.senderInfo.pk,
			RecipientAddr: testGlobal.recipientInfo.addr,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 5,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  5,
			LeaseOut: 0,
		},
		&proto.CancelledLeaseSnapshot{
			LeaseID: crypto.MustDigestFromBase58(invokeId),
		},
	}
	err = to.entities.balances.addInitialBalancesIfNotExists(snapshotsSetSecond)
	assert.NoError(t, err)

	to.entities.balances.addLeasesBalanceChangeLegacySH(testGlobal.senderInfo.Address().ID(), 0, 5)
	to.entities.balances.addLeasesBalanceChangeLegacySH(testGlobal.recipientInfo.Address().ID(), 5, 0)
	l := &leasing{
		SenderPK:            testGlobal.senderInfo.pk,
		RecipientAddr:       testGlobal.recipientInfo.addr,
		Amount:              5,
		OriginHeight:        0,
		Status:              LeaseCancelled,
		OriginTransactionID: nil,
		CancelHeight:        0,
		CancelTransactionID: nil,
	}
	to.entities.balances.addCancelLeasesBalanceChangeLegacySH(l)

	to.entities.balances.filterZeroDiffsSHOut(blockID0)
}
