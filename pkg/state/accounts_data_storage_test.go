package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type accountsDataStorageTestObjects struct {
	stor             *storageObjects
	accountsDataStor *accountsDataStorage
}

func createAccountsDataStorgae() (*accountsDataStorageTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	accountsDataStor, err := newAccountsDataStorage(stor.db, stor.dbBatch, stor.rw, stor.stateDB)
	if err != nil {
		return nil, path, err
	}
	return &accountsDataStorageTestObjects{stor, accountsDataStor}, path, nil
}

func TestAppendEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	to.stor.flush(t)
	newEntry, err := to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry0, newEntry)
	to.stor.addBlock(t, blockID1)
	// Add entry with same key in diff block and check that the value changed.
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	to.stor.flush(t)
	newEntry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry1, newEntry)
}

func TestRollbackEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	to.stor.flush(t)
	// Latest entry should be from blockID1.
	entry, err := to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry1, entry)
	// Rollback block.
	err = to.stor.stateDB.rollbackBlock(blockID1)
	assert.NoError(t, err, "rollbackBlock() failed")
	// Make sure data entry is now from blockID0.
	entry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveIntegerEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(100500)}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	to.stor.flush(t)
	entry, err := to.accountsDataStor.retrieveIntegerEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveIntegerEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveBooleanEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BooleanDataEntry{Key: "TheKey", Value: true}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	to.stor.flush(t)
	entry, err := to.accountsDataStor.retrieveBooleanEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBooleanEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveStringEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.StringDataEntry{Key: "TheKey", Value: "TheValue"}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	to.stor.flush(t)
	entry, err := to.accountsDataStor.retrieveStringEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveStringEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveBinaryEntry(t *testing.T) {
	to, path, err := createAccountsDataStorgae()
	assert.NoError(t, err, "createAccountsDataStorgae() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BinaryDataEntry{Key: "TheKey", Value: []byte{0xaa, 0xff}}
	to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	to.stor.flush(t)
	entry, err := to.accountsDataStor.retrieveBinaryEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBinaryEntry() failed")
	assert.Equal(t, entry0, entry)
}
