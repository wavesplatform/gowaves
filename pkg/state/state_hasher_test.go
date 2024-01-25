package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
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
	to.entities.calculateHashes = true
	to.entities.balances.calculateHashes = true
	to.addBlock(t, blockID0)

	assetID, dErr := crypto.NewDigestFromBase58("AiNNtMkp21Utu8QzDCcc9zzdHmosLKq8qCHfwMR4GJ9E")
	require.NoError(t, dErr)
	to.createAsset(t, assetID)

	snapshotApplier := newBlockSnapshotsApplier(
		&blockSnapshotsApplierInfo{
			ci:                  &checkerInfo{blockID: blockID0},
			scheme:              proto.MainNetScheme,
			stateActionsCounter: nil,
		},
		newSnapshotApplierStorages(to.entities, to.rw),
	)
	swbErr := to.entities.balances.setWavesBalance(testGlobal.recipientInfo.addr.ID(), wavesValue{
		profile: balanceProfile{
			balance:  5,
			leaseIn:  0,
			leaseOut: 0,
		},
		leaseChange:   false,
		balanceChange: false,
	}, blockID0)
	require.NoError(t, swbErr)

	snapshotsSetFirst := []proto.AtomicSnapshot{
		&proto.WavesBalanceSnapshot{Address: testGlobal.issuerInfo.addr, Balance: 1},
		&proto.WavesBalanceSnapshot{Address: testGlobal.senderInfo.addr, Balance: 3},
		&proto.WavesBalanceSnapshot{Address: testGlobal.recipientInfo.addr, Balance: 5},
		&proto.AssetBalanceSnapshot{Address: testGlobal.issuerInfo.addr, AssetID: assetID, Balance: 21},
	}

	for _, s := range snapshotsSetFirst {
		err := s.Apply(&snapshotApplier)
		require.NoError(t, err)
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
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.issuerInfo.addr,
			AssetID: assetID,
			Balance: 0,
		},
		&proto.AssetBalanceSnapshot{
			Address: testGlobal.recipientInfo.addr,
			AssetID: assetID,
			Balance: 21,
		},
	}

	for _, s := range snapshotsSetSecond {
		err := s.Apply(&snapshotApplier)
		require.NoError(t, err)
	}

	wavesTmpSHRecords := to.entities.balances.wavesHashesState[blockID0]
	leaseTmpSHRecords := to.entities.balances.leaseHashesState[blockID0]
	assetsTmpSHRecords := to.entities.balances.assetsHashesState[blockID0]

	testGlobal.issuerInfo.Address()
	wavesKeyA := wavesBalanceKey{address: testGlobal.issuerInfo.addr.ID()}
	wavesKeyB := wavesBalanceKey{address: testGlobal.senderInfo.addr.ID()}
	wavesKeyC := wavesBalanceKey{address: testGlobal.recipientInfo.addr.ID()}
	leaseKeyA := wavesBalanceKey{address: testGlobal.senderInfo.addr.ID()}
	leaseKeyB := wavesBalanceKey{address: testGlobal.recipientInfo.addr.ID()}
	assetKeyA := assetBalanceKey{address: testGlobal.issuerInfo.addr.ID(), asset: proto.AssetIDFromDigest(assetID)}
	assetKeyB := assetBalanceKey{address: testGlobal.recipientInfo.addr.ID(), asset: proto.AssetIDFromDigest(assetID)}

	// do checks before filtering out zero diffs
	_, wavesFoundAshRecord := wavesTmpSHRecords.componentByKey[string(wavesKeyA.bytes())]
	assert.True(t, wavesFoundAshRecord)
	_, wavesFoundBshRecord := wavesTmpSHRecords.componentByKey[string(wavesKeyB.bytes())]
	assert.True(t, wavesFoundBshRecord)

	// initial balance 5, result balance 5 => diff 0, so no record
	_, wavesFoundCshRecord := wavesTmpSHRecords.componentByKey[string(wavesKeyC.bytes())]
	assert.False(t, wavesFoundCshRecord)

	_, leaseFoundAshRecord := leaseTmpSHRecords.componentByKey[string(leaseKeyA.bytes())]
	assert.True(t, leaseFoundAshRecord)
	_, leaseFoundBshRecord := leaseTmpSHRecords.componentByKey[string(leaseKeyB.bytes())]
	assert.True(t, leaseFoundBshRecord)

	_, assetFoundAshRecord := assetsTmpSHRecords.componentByKey[string(assetKeyA.bytes())]
	assert.True(t, assetFoundAshRecord)
	_, assetFoundBshRecord := assetsTmpSHRecords.componentByKey[string(assetKeyB.bytes())]
	assert.True(t, assetFoundBshRecord)

	// filter out zero diffs
	snapshotApplier.filterZeroDiffsSHOut(blockID0)
	/*
		for wavesKeyA (issuer) the initial balance 0, result balance 1
		for wavesKeyB (sender) the initial balance 0, result balance 3
		for wavesKeyC (recipient) the initial balance 5, result balance 5 => diff 0
		for leaseKeyA (issuer) the initial lease balances 0, result balances 0 => diff 0
		for leaseKeyB (issuer) the initial lease balances 0, result balances 0 => diff 0
		for assetKeyA (issuer) the initial balance 0, intermediate balance 21, result balance 0 => diff 0
		for assetKeyB (recipient) the initial balance 0, result balance 21
	*/

	// do checks after filtering out zero diffs
	_, wavesFoundAshRecord = wavesTmpSHRecords.componentByKey[string(wavesKeyA.bytes())]
	assert.True(t, wavesFoundAshRecord)
	_, wavesFoundBshRecord = wavesTmpSHRecords.componentByKey[string(wavesKeyB.bytes())]
	assert.True(t, wavesFoundBshRecord)

	_, wavesFoundCshRecord = wavesTmpSHRecords.componentByKey[string(wavesKeyC.bytes())]
	assert.False(t, wavesFoundCshRecord)

	_, leaseFoundAshRecord = leaseTmpSHRecords.componentByKey[string(leaseKeyA.bytes())]
	assert.False(t, leaseFoundAshRecord)
	_, leaseFoundBshRecord = leaseTmpSHRecords.componentByKey[string(leaseKeyB.bytes())]
	assert.False(t, leaseFoundBshRecord)

	_, assetFoundAshRecord = assetsTmpSHRecords.componentByKey[string(assetKeyA.bytes())]
	assert.False(t, assetFoundAshRecord)
	_, assetFoundBshRecord = assetsTmpSHRecords.componentByKey[string(assetKeyB.bytes())]
	assert.True(t, assetFoundBshRecord)
}
