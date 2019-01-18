package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"os"
	"path/filepath"
	"testing"
)

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

func TestSingleAccountsState(t *testing.T) {
	db, closeDB := openDB(t, "account-state-db")
	defer closeDB()

	addr1, err := proto.NewAddressFromString("3N1746xR1R4hzWF2GXcMXS7mH9cm8yq6oZR")
	assert.NoError(t, err)
	acc1 := Account{Address: addr1}
	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	u1 := []AccountChange{{Account: acc1, Asset: asset1, In: 100, Out: 0}}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 1, u1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 100, int(b))
	}

	u2 := []AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 2, u2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		b, _, err := bs.balance(addr1, asset1)
		assert.NoError(t, err)
		assert.Equal(t, 50, int(b))
	}

	u3 := []AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 3, u3)
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
	err = rollbackBalances(snapshot, batch, 2)
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
	db, closeDB := openDB(t, "account-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)
	asset2, err := crypto.NewDigestFromBase58("HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj")
	assert.NoError(t, err)

	addr1, err := proto.NewAddressFromString("3N1746xR1R4hzWF2GXcMXS7mH9cm8yq6oZR")
	assert.NoError(t, err)
	acc1 := Account{Address: addr1}

	alias2, err := proto.NewAlias(scheme, "alias2")
	assert.NoError(t, err)
	addr2, err := proto.NewAddressFromString("3NB1Yz7fH1bJ2gVDjyJnuyKNTdMFARkKEpV")
	assert.NoError(t, err)
	acc2 := Account{Alias: *alias2}

	u0 := []AliasBind{{Alias: *alias2, Address: addr2}}
	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putAliasesStateUpdate(bs, batch, 1, u0)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)

	u1 := []AccountChange{{Account: acc1, Asset: asset1, In: 1000, Out: 0}, {Account: acc2, Asset: asset2, In: 2000, Out: 0}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	bs.issuers[assetIssuerKey{address: addr2, asset: asset2}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 2, u1)
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

	u2 := []AccountChange{
		{Account: acc1, Asset: asset1, In: 0, Out: 50},
		{Account: acc2, Asset: asset1, In: 50, Out: 0},
		{Account: acc2, Asset: asset2, In: 0, Out: 100},
		{Account: acc1, Asset: asset2, In: 100, Out: 0},
	}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	bs.issuers[assetIssuerKey{address: addr2, asset: asset2}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 3, u2)
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

	u3 := []AccountChange{{Account: acc1, Asset: asset1, In: 0, Out: 50}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	bs.issuers[assetIssuerKey{address: addr1, asset: asset1}] = struct{}{}
	err = putBalancesStateUpdate(bs, batch, 4, u3)
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
	err = rollbackBalances(snapshot, batch, 2)
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
	ai := assetInfo{name: "asset", issuer: pk, decimals: 8, reissuable: true, sponsored: true, supply: 123456}
	var ai2 assetInfo
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
	db, closeDB := openDB(t, "asset-state-db")
	defer closeDB()

	asset1, err := crypto.NewDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")
	assert.NoError(t, err)

	pk1, err := crypto.NewPublicKeyFromBase58("Hoox6WK7gxNFUYYUKz4oR1iGs7QxTWYPAjgs6RhbDLAL")
	assert.NoError(t, err)
	addr1, err := proto.NewAddressFromPublicKey(scheme, pk1)
	assert.NoError(t, err)

	u1 := []IssueChange{{AssetID: asset1, Name: "asset1", Issuer: pk1, Decimals: 2, Quantity: 100000, Reissuable: true}}
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

	u2 := []AssetChange{{AssetID: asset1, SetReissuable:true, Reissuable: false, Issued: 10000}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssetChanges(bs, batch, 2, u2)
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


	u3 := []AssetChange{{AssetID: asset1, Burned:5000}, {AssetID:asset1, SetSponsored:true, Sponsored:true}}
	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAssetChanges(bs, batch, 3, u3)
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
	err = rollbackAssetInfos(snapshot, batch, 3)
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
	err = rollbackAssetInfos(snapshot, batch, 2)
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
