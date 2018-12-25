package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"os"
	"path/filepath"
	"testing"
)

const (
	scheme byte = 'T'
)

func TestAliasState1(t *testing.T) {
	alias1, err := proto.NewAlias(scheme, "alias1")
	assert.NoError(t, err)
	addr1, err := proto.NewAddressFromString("3N1746xR1R4hzWF2GXcMXS7mH9cm8yq6oZR")
	assert.NoError(t, err)
	alias2, err := proto.NewAlias(scheme, "alias2")
	assert.NoError(t, err)
	addr2, err := proto.NewAddressFromString("3NB1Yz7fH1bJ2gVDjyJnuyKNTdMFARkKEpV")
	assert.NoError(t, err)
	alias3, err := proto.NewAlias(scheme, "-xxx-")
	assert.NoError(t, err)
	addr3, err := proto.NewAddressFromString("3N5GRqzDBhjVXnCn44baHcz2GoZy5qLxtTh")
	assert.NoError(t, err)
	u1 := []AliasBind{{Alias: *alias1, Address: addr1}}
	u2 := []AliasBind{{Alias: *alias2, Address: addr2}}
	u3 := []AliasBind{{Alias: *alias3, Address: addr3}}

	path := filepath.Join(os.TempDir(), "alias-state-db")
	opts := opt.Options{ErrorIfExist: true}
	db, err := leveldb.OpenFile(path, &opts)
	assert.NoError(t, err)

	snapshot, err := db.GetSnapshot()
	assert.NoError(t, err)
	batch := new(leveldb.Batch)
	err = PutAliasesStateUpdate(snapshot, batch, 1, u1)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if 	snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		a, err := GetAddress(snapshot, *alias1)
		assert.NoError(t, err)
		assert.Equal(t, addr1, a)
		a, err = GetAddress(snapshot, *alias2)
		assert.EqualError(t, err, "failed to get an Address: leveldb: not found")
		assert.Equal(t, proto.Address{}, a)
		a, err = GetAddress(snapshot, *alias3)
		assert.EqualError(t, err, "failed to get an Address: leveldb: not found")
		assert.Equal(t, proto.Address{}, a)
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = PutAliasesStateUpdate(snapshot, batch, 2, u2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if 	snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		a, err := GetAddress(snapshot, *alias1)
		assert.NoError(t, err)
		assert.Equal(t, addr1, a)
		a, err = GetAddress(snapshot, *alias2)
		assert.NoError(t, err)
		assert.Equal(t, addr2, a)
		a, err = GetAddress(snapshot, *alias3)
		assert.EqualError(t, err, "failed to get an Address: leveldb: not found")
		assert.Equal(t, proto.Address{}, a)
	}

	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = PutAliasesStateUpdate(snapshot, batch, 3, u3)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if 	snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		a, err := GetAddress(snapshot, *alias1)
		assert.NoError(t, err)
		assert.Equal(t, addr1, a)
		a, err = GetAddress(snapshot, *alias2)
		assert.NoError(t, err)
		assert.Equal(t, addr2, a)
		a, err = GetAddress(snapshot, *alias3)
		assert.NoError(t, err)
		assert.Equal(t, addr3, a)
	}


	snapshot, err = db.GetSnapshot()
	assert.NoError(t, err)
	batch = new(leveldb.Batch)
	err = RollbackAliases(snapshot, batch, 2)
	assert.NoError(t, err)
	err = db.Write(batch, nil)
	assert.NoError(t, err)
	if 	snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		a, err := GetAddress(snapshot, *alias1)
		assert.NoError(t, err)
		assert.Equal(t, addr1, a)
		a, err = GetAddress(snapshot, *alias2)
		assert.EqualError(t, err, "failed to get an Address: leveldb: not found")
		assert.Equal(t, proto.Address{}, a)
		a, err = GetAddress(snapshot, *alias3)
		assert.EqualError(t, err, "failed to get an Address: leveldb: not found")
		assert.Equal(t, proto.Address{}, a)
	}

	err = db.Close()
	assert.NoError(t, err)
	err = os.RemoveAll(path)
	assert.NoError(t, err)
}
