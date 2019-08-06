package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSingleAccountsState(t *testing.T) {
	db, closeDB := openDB(t, "wmd-account-state-db")
	defer closeDB()

	pk1, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	require.NoError(t, err)
	acc1 := data.Account{Address: addr1}
	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	u0 := []data.IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 0, Reissuable: true, Quantity: 100}}
	u1 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 100, Out: 0}}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u0)
	require.NoError(t, err)
	err = putAccounts(bs, batch, 1, u1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 100, int(b))
	}

	u2 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 50, int(b))
	}

	u3 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 100, int(b))
	}

}

func TestMultipleAccountState(t *testing.T) {
	db, closeDB := openDB(t, "wmd-account-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	asset2, err := crypto.NewDigestFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	require.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	require.NoError(t, err)
	acc1 := data.Account{Address: addr1}

	alias2 := proto.NewAlias(scheme, "alias2")
	pk2, err := crypto.NewPublicKeyFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	require.NoError(t, err)
	addr2, err := proto.NewAddressFromPublicKey(scheme, pk2)
	require.NoError(t, err)
	acc2 := data.Account{Alias: *alias2}

	u01 := []data.AliasBind{{Alias: *alias2, Address: addr2}}
	u02 := []data.IssueChange{
		{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 0, Reissuable: true, Quantity: 1000},
		{AssetID: asset2, Name: "asset2", Issuer: pk2, Decimals: 0, Reissuable: false, Quantity: 2000},
	}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putAliases(bs, batch, 1, u01)
	require.NoError(t, err)
	err = putIssues(bs, batch, scheme, 1, u02)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)

	u1 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 1000, Out: 0}, {Account: acc2, Asset: asset2, In: 2000, Out: 0}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 1000, int(b))
		b, _, err = bs.balance(addr2, asset2)
		require.NoError(t, err)
		assert.Equal(t, 2000, int(b))
	}

	u2 := []data.AccountChange{
		{Account: acc1, Asset: asset1, In: 0, Out: 50},
		{Account: acc2, Asset: asset1, In: 50, Out: 0},
		{Account: acc2, Asset: asset2, In: 0, Out: 100},
		{Account: acc1, Asset: asset2, In: 100, Out: 0},
	}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 3, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 950, int(b))
		b, _, err = bs.balance(addr2, asset1)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
		b, _, err = bs.balance(addr2, asset2)
		require.NoError(t, err)
		assert.Equal(t, 1900, int(b))
		b, _, err = bs.balance(addr1, asset2)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	u3 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 4, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 900, int(b))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
		b, _, err = bs.balance(addr2, asset2)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}
}

func TestAssetInfoBytesRoundTrip(t *testing.T) {
	pk, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err)
	ai := asset{name: "asset", issuer: addr, decimals: 8, reissuable: true, sponsored: true, supply: 123456}
	var ai2 asset
	err = ai2.fromBytes(ai.bytes())
	require.NoError(t, err)
	assert.Equal(t, ai.name, ai2.name)
	assert.Equal(t, ai.issuer, ai2.issuer)
	assert.Equal(t, ai.decimals, ai2.decimals)
	assert.Equal(t, ai.reissuable, ai2.reissuable)
	assert.Equal(t, ai.sponsored, ai2.sponsored)
	assert.Equal(t, ai.supply, ai2.supply)
}

func TestAssetInfoIssueReissueRollback(t *testing.T) {
	db, closeDB := openDB(t, "wmd-asset-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	require.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	require.NoError(t, err)

	u1 := []data.IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 2, Quantity: 100000, Reissuable: true}}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
	}

	u2 := []data.AssetChange{{AssetID: asset1, SetReissuable: true, Reissuable: false, Issued: 10000}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 110000, int(ai.supply))
	}

	u3 := []data.AssetChange{{AssetID: asset1, Burned: 5000}, {AssetID: asset1, SetSponsored: true, Sponsored: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, true, ai.sponsored)
		assert.Equal(t, 105000, int(ai.supply))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 110000, int(ai.supply))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		_, ok, err = bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.False(t, ok)
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
	}
}

func TestAssetTransferRollback(t *testing.T) {
	db, closeDB := openDB(t, "wmd-asset-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	require.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	require.NoError(t, err)
	acc1 := data.Account{Address: addr1}

	u11 := []data.IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 2, Quantity: 100000, Reissuable: true}}
	u12 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 100000, Out: 0}}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u11)
	require.NoError(t, err)
	err = putAccounts(bs, batch, 1, u12)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 100000, int(b))
	}

	u2 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 10000}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 90000, int(b))
	}

	u3 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 5000, Out: 0}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 95000, int(b))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 3)
	require.NoError(t, err)
	err = rollbackAssets(snapshot, batch, 3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 90000, int(b))
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	require.NoError(t, err)
	err = rollbackAssets(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 100000, int(b))
	}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	require.NoError(t, err)
	err = rollbackAssets(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
		b, _, err := bs.balance(addr1, asset1)
		require.NoError(t, err)
		assert.Equal(t, 100000, int(b))
	}
}

func TestSponsorshipOneAccountAsMinerAndIssuer(t *testing.T) {
	db, closeDB := openDB(t, "wmd-asset-state-db")
	defer closeDB()

	asset, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)

	pk, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err)
	acc := data.Account{Address: addr}

	u11 := []data.IssueChange{{AssetID: asset, Name: "asset", Issuer: pk, Decimals: 0, Quantity: 1000, Reissuable: false}}
	u12 := []data.AccountChange{{Account: acc, Asset: asset, In: 1000, Out: 0}}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u11)
	require.NoError(t, err)
	err = putAccounts(bs, batch, 1, u12)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr, asset)
		require.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr, ai.issuer)
		assert.Equal(t, false, ai.sponsored)
		b, _, err := bs.balance(addr, asset)
		require.NoError(t, err)
		assert.Equal(t, 1000, int(b))
	}

	u2 := []data.AccountChange{{Account: acc, Asset: asset, In: 0, Out: 100}, {Account: acc, Asset: asset, In: 0, Out: 1}, {Account: acc, Asset: asset, In: 1, Out: 0, MinersReward: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr, asset)
		require.NoError(t, err)
		assert.Equal(t, 900, int(b))
	}

	u3 := []data.AssetChange{{AssetID: asset, SetSponsored: true, Sponsored: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ai, ok, err := bs.assetInfo(asset)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr, ai.issuer)
		assert.Equal(t, true, ai.sponsored)
	}

	u4 := []data.AccountChange{{Account: acc, Asset: asset, In: 0, Out: 100}, {Account: acc, Asset: asset, In: 0, Out: 1}, {Account: acc, Asset: asset, In: 1, Out: 0, MinersReward: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 4, u4)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr, asset)
		require.NoError(t, err)
		assert.Equal(t, 800, int(b))
	}
}

func TestSponsorshipIssuerNotMiner(t *testing.T) {
	db, closeDB := openDB(t, "wmd-asset-state-db")
	defer closeDB()

	asset, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	require.NoError(t, err)
	pk1, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	require.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	require.NoError(t, err)
	acc1 := data.Account{Address: addr1}
	pk2, err := crypto.NewPublicKeyFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	require.NoError(t, err)
	addr2, err := proto.NewAddressFromPublicKey(scheme, pk2)
	require.NoError(t, err)
	acc2 := data.Account{Address: addr2}

	u11 := []data.IssueChange{{AssetID: asset, Name: "asset", Issuer: pk1, Decimals: 0, Quantity: 1000, Reissuable: false}}
	u12 := []data.AccountChange{{Account: acc1, Asset: asset, In: 1000, Out: 0}}
	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u11)
	require.NoError(t, err)
	err = putAccounts(bs, batch, 1, u12)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset)
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = bs.isIssuer(addr2, asset)
		require.NoError(t, err)
		assert.False(t, ok)
		ai, ok, err := bs.assetInfo(asset)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, false, ai.sponsored)
		b, _, err := bs.balance(addr1, asset)
		require.NoError(t, err)
		assert.Equal(t, 1000, int(b))
		b, _, err = bs.balance(addr2, asset)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	u2 := []data.AccountChange{{Account: acc1, Asset: asset, In: 0, Out: 100}, {Account: acc1, Asset: asset, In: 0, Out: 1}, {Account: acc2, Asset: asset, In: 1, Out: 0, MinersReward: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset)
		require.NoError(t, err)
		assert.Equal(t, 899, int(b))
		b, _, err = bs.balance(addr2, asset)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	u3 := []data.AssetChange{{AssetID: asset, SetSponsored: true, Sponsored: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ai, ok, err := bs.assetInfo(asset)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr1, ai.issuer)
		assert.Equal(t, true, ai.sponsored)
	}

	u4 := []data.AccountChange{{Account: acc1, Asset: asset, In: 0, Out: 100}, {Account: acc1, Asset: asset, In: 0, Out: 1}, {Account: acc2, Asset: asset, In: 1, Out: 0, MinersReward: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 4, u4)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset)
		require.NoError(t, err)
		assert.Equal(t, 799, int(b))
		b, _, err = bs.balance(addr2, asset)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	u5 := []data.AccountChange{{Account: acc2, Asset: asset, In: 0, Out: 100}, {Account: acc2, Asset: asset, In: 0, Out: 1}, {Account: acc2, Asset: asset, In: 1, Out: 0, MinersReward: true}}
	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 5, u5)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset)
		require.NoError(t, err)
		assert.Equal(t, 800, int(b))
		b, _, err = bs.balance(addr2, asset)
		require.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}
}
