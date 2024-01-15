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

func TestLegacyStateHashSupport(t *testing.T) {
	to := createStorageObjects(t, true)
	to.addBlock(t, blockID0)
	to.entities.balances.calculateHashes = true

	var snapshotApplier = &blockSnapshotsApplier{
		info: &blockSnapshotsApplierInfo{
			ci:                  &checkerInfo{blockID: blockID0},
			scheme:              proto.MainNetScheme,
			stateActionsCounter: nil,
		},
		stor:              newSnapshotApplierStorages(to.entities, to.rw),
		txSnapshotContext: txSnapshotContext{},
		issuedAssets:      nil,
		scriptedAssets:    nil,
		newLeases:         nil,
		cancelledLeases:   make(map[crypto.Digest]struct{}),
	}
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

	snapshotsSetFirst := []proto.AtomicSnapshot{
		&proto.WavesBalanceSnapshot{Address: proto.MustAddressFromString(testGlobal.issuerInfo.Address().String()), Balance: 1},
		&proto.WavesBalanceSnapshot{Address: testGlobal.senderInfo.addr, Balance: 3},
		&proto.WavesBalanceSnapshot{Address: testGlobal.recipientInfo.addr, Balance: 5},
	}

	err = to.entities.balances.addInitialBalancesIfNotExists(snapshotsSetFirst)
	assert.NoError(t, err)

	for _, s := range snapshotsSetFirst {
		err := s.Apply(snapshotApplier)
		assert.NoError(t, err)
	}

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
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.senderInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
		&proto.LeaseBalanceSnapshot{
			Address:  testGlobal.recipientInfo.addr,
			LeaseIn:  0,
			LeaseOut: 0,
		},
	}
	err = to.entities.balances.addInitialBalancesIfNotExists(snapshotsSetSecond)
	assert.NoError(t, err)

	for _, s := range snapshotsSetSecond {
		err := s.Apply(snapshotApplier)
		assert.NoError(t, err)
	}

	to.entities.balances.filterZeroDiffsSHOut(blockID0)
}
