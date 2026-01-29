package state

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
	"github.com/wavesplatform/gowaves/pkg/settings"
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
			Balance:  5,
			LeaseIn:  0,
			LeaseOut: 0,
			Deposit:  0,
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

func TestScalaCompatibility(t *testing.T) {
	// Output from Scala test com/wavesplatform/state/StateHashSpec.scala:16
	address := proto.MustAddressFromString("3My3KZgFQ3CrVHgz6vGRt8687sH4oAA1qp8")
	address1 := proto.MustAddressFromString("3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh")
	assetID := crypto.MustDigestFromBase58("9ekQuYn92natMnMq8KqeGK3Nn7cpKd3BvPEGgD6fFyyz")
	pk, err := crypto.NewPublicKeyFromBase58("9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ")
	require.NoError(t, err)
	blsPK, err := bls.NewPublicKeyFromBase58("7QtCEETGT76GHP7gR3Qc9DQzNjJYbxn4UJ7Bz7RofMQx5RJY7mZNveuFNfgJYg2kLn")
	require.NoError(t, err)

	code := `
	{-# STDLIB_VERSION 2 #-}
	{-# CONTENT_TYPE EXPRESSION #-}
	{-# SCRIPT_TYPE ACCOUNT #-}
	true
	`
	script, errors := ridec.Compile(code, false, false)
	require.Nil(t, errors)

	bID := proto.NewBlockIDFromDigest(crypto.Digest{})

	leaseHasher := newStateHasher()
	err = leaseHasher.push("leaseBalance", &leaseBalanceRecordForHashes{
		addr:     &address,
		leaseIn:  10000,
		leaseOut: 10000,
	}, bID)
	require.NoError(t, err)
	err = leaseHasher.stop()
	require.NoError(t, err)
	assert.Equal(t,
		"PZWx1OT3QuQXA2Tu5l24GN3LxnlfWakj4rQdzyHJr68=",
		base64.StdEncoding.EncodeToString(leaseHasher.stateHashAt(bID).Bytes()))

	accountScriptHasher := newStateHasher()
	err = accountScriptHasher.push("accountScript", &accountScripRecordForHashes{
		addr:   &address,
		script: script,
	}, bID)
	require.NoError(t, err)
	err = accountScriptHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "ixFJABpqIXRbncERNnGqE02DARpi5/SGOg9VJuwy8W0=",
		base64.StdEncoding.EncodeToString(accountScriptHasher.stateHashAt(bID).Bytes()))

	assetScriptHasher := newStateHasher()
	err = assetScriptHasher.push("assetScript", &assetScripRecordForHashes{
		asset:  assetID,
		script: script,
	}, bID)
	require.NoError(t, err)
	err = assetScriptHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "76XNneo9mK5bO2/EjDQAhlztXinq5+0h/fb40HL7s+o=",
		base64.StdEncoding.EncodeToString(assetScriptHasher.stateHashAt(bID).Bytes()))

	aliasHasher := newStateHasher()
	err = aliasHasher.push("alias1", &aliasRecordForStateHashes{
		addr:  address,
		alias: []byte("test"),
	}, bID)
	require.NoError(t, err)
	err = aliasHasher.push("alias2", &aliasRecordForStateHashes{
		addr:  address,
		alias: []byte("test1"),
	}, bID)
	require.NoError(t, err)
	err = aliasHasher.push("alias3", &aliasRecordForStateHashes{
		addr:  address1,
		alias: []byte("test2"),
	}, bID)
	require.NoError(t, err)
	err = aliasHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "LgTVfXhl5/XLer00v+dhVT2GBHtD3rpWjhs9rxao6y8=",
		base64.StdEncoding.EncodeToString(aliasHasher.stateHashAt(bID).Bytes()))

	dataEntryHasher := newStateHasher()
	de := proto.StringDataEntry{
		Key:   "test",
		Value: "test",
	}
	vb, err := de.MarshalValue()
	require.NoError(t, err)
	err = dataEntryHasher.push("dataEntry", &dataEntryRecordForHashes{
		addr:  address.Bytes(),
		key:   []byte(de.Key),
		value: vb,
	}, bID)
	require.NoError(t, err)
	err = dataEntryHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "u0/DkX/iOy9g6jmaMtBa1IGAOIXlOfMJPxqyRYtfvo8=",
		base64.StdEncoding.EncodeToString(dataEntryHasher.stateHashAt(bID).Bytes()))

	leaseStatusHasher := newStateHasher()
	err = leaseStatusHasher.push("leaseStatus", &leaseRecordForStateHashes{
		id:       assetID,
		isActive: true,
	}, bID)
	require.NoError(t, err)
	err = leaseStatusHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "iacJITiqoPvN4eYHyb+22vyEevcXVf0Rlo3H4U+Pbvk=",
		base64.StdEncoding.EncodeToString(leaseStatusHasher.stateHashAt(bID).Bytes()))

	sponsorshipHasher := newStateHasher()
	err = sponsorshipHasher.push("sponsorship", &sponsorshipRecordForHashes{
		id:   assetID,
		cost: 1000,
	}, bID)
	require.NoError(t, err)
	err = sponsorshipHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "KSBmKoG2bDg8kdzC+iXrYJOIRK45cpDl9h0P0GeboPM=",
		base64.StdEncoding.EncodeToString(sponsorshipHasher.stateHashAt(bID).Bytes()))

	assetBalanceHasher := newStateHasher()
	err = assetBalanceHasher.push("assetBalance1", &assetRecordForHashes{
		addr:    &address,
		asset:   assetID,
		balance: 2000,
	}, bID)
	require.NoError(t, err)
	err = assetBalanceHasher.push("assetBalance2", &assetRecordForHashes{
		addr:    &address1,
		asset:   assetID,
		balance: 2000,
	}, bID)
	require.NoError(t, err)
	err = assetBalanceHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "TUKPNzY41ho40LeluH9drR5enLTIbD7EDyrAdxkIoG8=",
		base64.StdEncoding.EncodeToString(assetBalanceHasher.stateHashAt(bID).Bytes()))

	wavesBalanceHasher := newStateHasher()
	err = wavesBalanceHasher.push("wavesBalance", &wavesRecordForHashes{
		addr:    &address,
		balance: 1000,
	}, bID)
	require.NoError(t, err)
	err = wavesBalanceHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "I4gBHqU03gVAKOkpQo3dbLB1muwWqhfhONTPIX6fq4Y=",
		base64.StdEncoding.EncodeToString(wavesBalanceHasher.stateHashAt(bID).Bytes()))

	generatorsHasher := newStateHasher()
	err = generatorsHasher.push("generators", &commitmentsRecordForStateHashes{publicKey: pk, blsPublicKey: blsPK}, bID)
	require.NoError(t, err)
	err = generatorsHasher.stop()
	require.NoError(t, err)
	assert.Equal(t, "6pTQljIImIOjWn1Rq3EsD63lChYnLWqJ8kPjek8AbBc=",
		base64.StdEncoding.EncodeToString(generatorsHasher.stateHashAt(bID).Bytes()))

	sh := proto.StateHashV2{
		BlockID: bID,
		FieldsHashesV2: proto.FieldsHashesV2{
			FieldsHashesV1: proto.FieldsHashesV1{
				WavesBalanceHash:  wavesBalanceHasher.stateHashAt(bID),
				AssetBalanceHash:  assetBalanceHasher.stateHashAt(bID),
				DataEntryHash:     dataEntryHasher.stateHashAt(bID),
				AccountScriptHash: accountScriptHasher.stateHashAt(bID),
				AssetScriptHash:   assetScriptHasher.stateHashAt(bID),
				LeaseBalanceHash:  leaseHasher.stateHashAt(bID),
				LeaseStatusHash:   leaseStatusHasher.stateHashAt(bID),
				SponsorshipHash:   sponsorshipHasher.stateHashAt(bID),
				AliasesHash:       aliasHasher.stateHashAt(bID),
			},
			GeneratorsHash: generatorsHasher.stateHashAt(bID),
			// EUKq8xDt8hyATpY6mmPev2bVjVmJAFQzXdTVyky34CEr
			GeneratorsBalancesHash: crypto.MustFastHash(binary.BigEndian.AppendUint64(nil, 3000)),
		},
	}
	prevHash := crypto.MustDigestFromBase58("46e2hSbVy6YNqx4GH2ZwJW66jMD6FgXzirAUHDD6mVGi")
	err = sh.GenerateSumHash(prevHash.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "KdA4trKip6EpfUSzca42sLVqqjuHishcHDQZeYDC1Mo", sh.SumHash.String())
}

func TestCalculateCommittedGeneratorsBalancesStateHash(t *testing.T) {
	so := createStorageObjects(t, true)
	so.activateFeature(t, int16(settings.DeterministicFinality)) // add first block
	featureActivationHeight, err := so.entities.features.newestActivationHeight(int16(settings.DeterministicFinality))
	require.NoError(t, err)

	pk, err := crypto.NewPublicKeyFromBase58("9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(so.settings.AddressSchemeCharacter, pk)
	require.NoError(t, err)

	bID := proto.NewBlockIDFromDigest(crypto.Digest{42})
	const initialBalance = 3000
	so.prepareAndStartBlock(t, bID) // prepare and start second block
	// -----------
	blockHeight := so.rw.addingBlockHeight()
	periodStart, err := CurrentGenerationPeriodStart(
		featureActivationHeight, blockHeight, so.settings.GenerationPeriod,
	)
	require.NoError(t, err)
	so.setWavesBalance(t, addr, balanceProfile{initialBalance, 0, 0, 0}, bID)
	err = so.entities.commitments.store(periodStart, pk, bls.PublicKey{1, 2, 3, 4, 5}, bID)
	require.NoError(t, err)
	// -----------
	so.finishBlock(t, bID) // finish second block
	// no flush, should be possible to calculate SH for unflushed data
	sh, err := calculateCommittedGeneratorsBalancesStateHash(so.entities, true, blockHeight)
	require.NoError(t, err)
	// "EUKq8xDt8hyATpY6mmPev2bVjVmJAFQzXdTVyky34CEr" —> value below
	expectedSH := crypto.MustFastHash(binary.BigEndian.AppendUint64(nil, initialBalance))
	assert.Equal(t, expectedSH, sh)
}
