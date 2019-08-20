package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type accountsDataStorageTestObjects struct {
	stor             *testStorageObjects
	accountsDataStor *accountsDataStorage
}

func (a *accountsDataStorageTestObjects) flush(t *testing.T) {
	a.accountsDataStor.flush()
	a.accountsDataStor.reset()
	a.stor.flush(t)
}

func createAccountsDataStorage() (*accountsDataStorageTestObjects, []string, error) {
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
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	newEntry, err := to.accountsDataStor.retrieveNewestEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry0, newEntry)
	to.flush(t)
	to.stor.addBlock(t, blockID1)
	// Add entry with same key in diff block and check that the value changed.
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	err = to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	assert.NoError(t, err)
	to.flush(t)
	newEntry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry1, newEntry)
}

func TestRollbackEntry(t *testing.T) {
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

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
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	err = to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	assert.NoError(t, err)
	// Latest entry should be from blockID1.
	entry, err := to.accountsDataStor.retrieveNewestEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry1, entry)
	// Flush and reset before rollback.
	to.flush(t)
	// Rollback block.
	err = to.stor.stateDB.rollbackBlock(blockID1)
	assert.NoError(t, err, "rollbackBlock() failed")
	// Make sure data entry is now from blockID0.
	entry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveIntegerEntry(t *testing.T) {
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(100500)}
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestIntegerEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestIntegerEntry() failed")
	assert.Equal(t, entry0, entry)
	to.flush(t)
	entry, err = to.accountsDataStor.retrieveIntegerEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveIntegerEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveBooleanEntry(t *testing.T) {
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BooleanDataEntry{Key: "TheKey", Value: true}
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestBooleanEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestBooleanEntry() failed")
	assert.Equal(t, entry0, entry)
	to.flush(t)
	entry, err = to.accountsDataStor.retrieveBooleanEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBooleanEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveStringEntry(t *testing.T) {
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.StringDataEntry{Key: "TheKey", Value: "TheValue"}
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestStringEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestStringEntry() failed")
	assert.Equal(t, entry0, entry)
	to.flush(t)
	entry, err = to.accountsDataStor.retrieveStringEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveStringEntry() failed")
	assert.Equal(t, entry0, entry)
}

func TestRetrieveBinaryEntry(t *testing.T) {
	to, path, err := createAccountsDataStorage()
	assert.NoError(t, err, "createAccountsDataStorage() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BinaryDataEntry{Key: "TheKey", Value: []byte{0xaa, 0xff}}
	err = to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestBinaryEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestBinaryEntry() failed")
	assert.Equal(t, entry0, entry)
	to.flush(t)
	entry, err = to.accountsDataStor.retrieveBinaryEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBinaryEntry() failed")
	assert.Equal(t, entry0, entry)
}
