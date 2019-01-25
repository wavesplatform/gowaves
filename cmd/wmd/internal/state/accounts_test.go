package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestSingleAccountsState(t *testing.T) {
	db, closeDB := openDB(t, "wmd-account-state-db")
	defer closeDB()

	pk1, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	assert.NoError(t, err)
	acc1 := data.Account{Address: addr1}
	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	u0 := []data.IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 0, Reissuable: true, Quantity: 100}}
	u1 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 100, Out: 0}}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u0)
	assert.NoError(t, err)
	err = putAccounts(bs, batch, 1, u1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 100, int(b))
	}

	u2 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 50, int(b))
	}

	u3 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 3, u3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 100, int(b))
	}

}

func TestMultipleAccountState(t *testing.T) {
	db, closeDB := openDB(t, "wmd-account-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	asset2, err := crypto.NewDigestFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	assert.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	assert.NoError(t, err)
	acc1 := data.Account{Address: addr1}

	alias2, err := proto.NewAlias(scheme, "alias2")
	assert.NoError(t, err)
	pk2, err := crypto.NewPublicKeyFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	assert.NoError(t, err)
	addr2, err := proto.NewAddressFromPublicKey(scheme, pk2)
	assert.NoError(t, err)
	acc2 := data.Account{Alias: *alias2}

	u01 := []data.AliasBind{{Alias: *alias2, Address: addr2}}
	u02 := []data.IssueChange{
		{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 0, Reissuable: true, Quantity: 1000},
		{AssetID: asset2, Name: "asset2", Issuer: pk2, Decimals: 0, Reissuable: false, Quantity: 2000},
	}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putAliases(bs, batch, 1, u01)
	assert.NoError(t, err)
	err = putIssues(bs, batch, scheme, 1, u02)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)

	u1 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 1000, Out: 0}, {Account: acc2, Asset: asset2, In: 2000, Out: 0}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 2, u1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 1000, int(b))
		b, _, err = bs.balance(addr2, asset2)
		assert.NoError(t, err)
		assert.Equal(t, 2000, int(b))
	}

	u2 := []data.AccountChange{
		{Account: acc1, Asset: asset1, In: 0, Out: 50},
		{Account: acc2, Asset: asset1, In: 50, Out: 0},
		{Account: acc2, Asset: asset2, In: 0, Out: 100},
		{Account: acc1, Asset: asset2, In: 100, Out: 0},
	}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 3, u2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 950, int(b))
		b, _, err = bs.balance(addr2, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(b))
		b, _, err = bs.balance(addr2, asset2)
		assert.NoError(t, err)
		assert.Equal(t, 1900, int(b))
		b, _, err = bs.balance(addr1, asset2)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}

	u3 := []data.AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAccounts(bs, batch, 4, u3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 900, int(b))
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAccounts(snapshot, batch, 2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(b))
		b, _, err = bs.balance(addr2, asset2)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(b))
	}
}

func TestAssetInfoBytesRoundTrip(t *testing.T) {
	pk, err := crypto.NewPublicKeyFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	ai := asset{name: "asset", issuer: pk, decimals: 8, reissuable: true, sponsored: true, supply: 123456}
	var ai2 asset
	err = ai2.fromBytes(ai.bytes())
	assert.NoError(t, err)
	assert.Equal(t, ai.name, ai2.name)
	assert.Equal(t, ai.issuer, ai2.issuer)
	assert.Equal(t, ai.decimals, ai2.decimals)
	assert.Equal(t, ai.reissuable, ai2.reissuable)
	assert.Equal(t, ai.sponsored, ai2.sponsored)
	assert.Equal(t, ai.supply, ai2.supply)
}

func TestAssetInfoIssueReissueRollback1(t *testing.T) {
	db, closeDB := openDB(t, "wmd-asset-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	assert.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	assert.NoError(t, err)

	u1 := []data.IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 2, Quantity: 100000, Reissuable: true}}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putIssues(bs, batch, scheme, 1, u1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, pk1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
	}

	u2 := []data.AssetChange{{AssetID: asset1, SetReissuable: true, Reissuable: false, Issued: 10000}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 2, u2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, pk1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 110000, int(ai.supply))
	}

	u3 := []data.AssetChange{{AssetID: asset1, Burned: 5000}, {AssetID: asset1, SetSponsored: true, Sponsored: true}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssets(bs, batch, 3, u3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, pk1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, true, ai.sponsored)
		assert.Equal(t, 105000, int(ai.supply))
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, pk1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, false, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 110000, int(ai.supply))
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAssets(snapshot, batch, 2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		ok, err := bs.isIssuer(addr1, asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		ai, ok, err := bs.assetInfo(asset1)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "asset1", ai.name)
		assert.Equal(t, pk1, ai.issuer)
		assert.Equal(t, 2, int(ai.decimals))
		assert.Equal(t, true, ai.reissuable)
		assert.Equal(t, false, ai.sponsored)
		assert.Equal(t, 100000, int(ai.supply))
	}
}
