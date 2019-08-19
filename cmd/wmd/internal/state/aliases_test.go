package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestAliasState(t *testing.T) {
	db, closeDB := openDB(t, "wmd-alias-state-db")
	defer closeDB()

	alias1 := proto.NewAlias(scheme, "alias1")
	addr1, err := proto.NewAddressFromString("3N1746xR1R4hzWF2GXcMXS7mH9cm8yq6oZR")
	require.NoError(t, err)
	alias2 := proto.NewAlias(scheme, "alias2")
	addr2, err := proto.NewAddressFromString("3NB1Yz7fH1bJ2gVDjyJnuyKNTdMFARkKEpV")
	require.NoError(t, err)
	alias3 := proto.NewAlias(scheme, "-xxx-")
	addr3, err := proto.NewAddressFromString("3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh")
	require.NoError(t, err)
	u1 := []data.AliasBind{{Alias: *alias1, Address: addr1}}
	u2 := []data.AliasBind{{Alias: *alias2, Address: addr2}}
	u3 := []data.AliasBind{{Alias: *alias3, Address: addr3}}

	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch := new(leveldb.Batch)
	bs := newBlockState(snapshot)
	err = putAliases(bs, batch, 1, u1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		a, ok, err := bs.addressByAlias(*alias1)
		assert.True(t, ok)
		require.NoError(t, err)
		assert.Equal(t, addr1, a)
		_, ok, err = bs.addressByAlias(*alias2)
		assert.False(t, ok)
		require.NoError(t, err)
		_, ok, err = bs.addressByAlias(*alias3)
		assert.False(t, ok)
		require.NoError(t, err)
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAliases(bs, batch, 2, u2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		a, ok, err := bs.addressByAlias(*alias1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr1, a)
		a, ok, err = bs.addressByAlias(*alias2)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr2, a)
		_, ok, err = bs.addressByAlias(*alias3)
		require.NoError(t, err)
		assert.False(t, ok)
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	bs = newBlockState(snapshot)
	err = putAliases(bs, batch, 3, u3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		a, ok, err := bs.addressByAlias(*alias1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr1, a)
		a, ok, err = bs.addressByAlias(*alias2)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr2, a)
		a, ok, err = bs.addressByAlias(*alias3)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr3, a)
	}

	snapshot, err = db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackAliases(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		bs = newBlockState(snapshot)
		a, ok, err := bs.addressByAlias(*alias1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, addr1, a)
		_, ok, err = bs.addressByAlias(*alias2)
		require.NoError(t, err)
		assert.False(t, ok)
		_, ok, err = bs.addressByAlias(*alias3)
		require.NoError(t, err)
		assert.False(t, ok)
	}
}
