package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type accountsDataStorageTestObjects struct {
	stor             *testStorageObjects
	accountsDataStor *accountsDataStorage
}

func createAccountsDataStorage(t *testing.T, amend bool) *accountsDataStorageTestObjects {
	stor := createStorageObjects(t, amend)
	accountsDataStor := newAccountsDataStorage(stor.db, stor.dbBatch, stor.hs, true)
	return &accountsDataStorageTestObjects{stor, accountsDataStor}
}

func TestAppendEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)

	newEntry, err := to.accountsDataStor.retrieveNewestEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry0, newEntry)

	to.stor.flush(t)
	to.stor.addBlock(t, blockID1)
	// Add entry with same key in diff block and check that the value changed.
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	err = to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	assert.NoError(t, err)

	to.stor.flush(t)
	newEntry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry1, newEntry)
}

func TestRetrieveEntries(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID2)
	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry1 := &proto.IntegerDataEntry{Key: "AnotherKey", Value: int64(42)}
	err = to.accountsDataStor.appendEntry(addr0, entry1, blockID0)
	assert.NoError(t, err)
	to.stor.flush(t)
	properEntries := []proto.DataEntry{entry0, entry1}
	entries, err := to.accountsDataStor.retrieveEntries(addr0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, properEntries, entries)

	// Test how it works with rollback.
	entry2 := &proto.BooleanDataEntry{Key: "Next", Value: true}
	to.stor.addBlock(t, blockID1)
	err = to.accountsDataStor.appendEntry(addr0, entry2, blockID1)
	assert.NoError(t, err)
	properEntries = []proto.DataEntry{entry0, entry1, entry2}
	to.stor.flush(t)
	entries, err = to.accountsDataStor.retrieveEntries(addr0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, properEntries, entries)

	to.stor.rollbackBlock(t, blockID1)
	properEntries = []proto.DataEntry{entry0, entry1}
	entries, err = to.accountsDataStor.retrieveEntries(addr0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, properEntries, entries)
	to.stor.rollbackBlock(t, blockID0)

	properEntries = nil
	entries, err = to.accountsDataStor.retrieveEntries(addr0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, properEntries, entries)
}

func TestRollbackEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "Whatever", Value: int64(100500)}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	to.stor.addBlock(t, blockID1)
	entry1 := &proto.BooleanDataEntry{Key: "Whatever", Value: true}
	err = to.accountsDataStor.appendEntry(addr0, entry1, blockID1)
	assert.NoError(t, err)
	ok, err := to.accountsDataStor.newestEntryExists(addr0)
	assert.NoError(t, err)
	assert.True(t, ok)
	// Latest entry should be from blockID1.
	entry, err := to.accountsDataStor.retrieveNewestEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestEntry() failed")
	assert.Equal(t, entry1, entry)
	// Flush and reset before rollback.
	to.stor.flush(t)
	to.accountsDataStor.reset()
	// Rollback block.
	to.stor.rollbackBlock(t, blockID1)
	to.stor.flush(t)
	to.accountsDataStor.reset()
	// Make sure data entry is now from blockID0.
	entry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveEntry() failed")
	assert.Equal(t, entry0, entry)
	ok, err = to.accountsDataStor.newestEntryExists(addr0)
	assert.NoError(t, err)
	assert.True(t, ok)
	to.stor.flush(t)
	to.accountsDataStor.reset()
	to.stor.rollbackBlock(t, blockID0)
	to.stor.flush(t)
	to.accountsDataStor.reset()
	// Make sure there is no data entry
	entry, err = to.accountsDataStor.retrieveEntry(addr0, entry0.Key)
	assert.Error(t, err)
	assert.Nil(t, entry)
	ok, err = to.accountsDataStor.newestEntryExists(addr0)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestRetrieveIntegerEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.IntegerDataEntry{Key: "TheKey", Value: int64(100500)}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestIntegerEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestIntegerEntry() failed")
	assert.Equal(t, entry0, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveIntegerEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveIntegerEntry() failed")
	assert.Equal(t, entry0, entry)

	// Test uncertain.
	entry1 := &proto.IntegerDataEntry{Key: "Uncertain", Value: 123}
	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	entry, err = to.accountsDataStor.retrieveNewestIntegerEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestIntegerEntry failed")
	assert.Equal(t, entry1, entry)
	to.accountsDataStor.dropUncertain()
	_, err = to.accountsDataStor.retrieveNewestIntegerEntry(addr0, entry1.Key)
	assert.Error(t, err)

	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	err = to.accountsDataStor.commitUncertain(blockID0)
	assert.NoError(t, err)
	entry, err = to.accountsDataStor.retrieveNewestIntegerEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestIntegerEntry failed")
	assert.Equal(t, entry1, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveIntegerEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveIntegerEntry failed")
	assert.Equal(t, entry1, entry)
}

func TestRetrieveBooleanEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BooleanDataEntry{Key: "TheKey", Value: true}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestBooleanEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestBooleanEntry() failed")
	assert.Equal(t, entry0, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveBooleanEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBooleanEntry() failed")
	assert.Equal(t, entry0, entry)

	// Test uncertain.
	entry1 := &proto.BooleanDataEntry{Key: "Uncertain", Value: true}
	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	entry, err = to.accountsDataStor.retrieveNewestBooleanEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestBooleanEntry failed")
	assert.Equal(t, entry1, entry)
	to.accountsDataStor.dropUncertain()
	_, err = to.accountsDataStor.retrieveNewestBooleanEntry(addr0, entry1.Key)
	assert.Error(t, err)

	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	err = to.accountsDataStor.commitUncertain(blockID0)
	assert.NoError(t, err)
	entry, err = to.accountsDataStor.retrieveNewestBooleanEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestBooleanEntry failed")
	assert.Equal(t, entry1, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveBooleanEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveBooleanEntry failed")
	assert.Equal(t, entry1, entry)
}

func TestRetrieveStringEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.StringDataEntry{Key: "TheKey", Value: "TheValue"}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestStringEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestStringEntry() failed")
	assert.Equal(t, entry0, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveStringEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveStringEntry() failed")
	assert.Equal(t, entry0, entry)

	// Test uncertain.
	entry1 := &proto.StringDataEntry{Key: "Uncertain", Value: "whatever"}
	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	entry, err = to.accountsDataStor.retrieveNewestStringEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestStringEntry failed")
	assert.Equal(t, entry1, entry)
	to.accountsDataStor.dropUncertain()
	_, err = to.accountsDataStor.retrieveNewestStringEntry(addr0, entry1.Key)
	assert.Error(t, err)

	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	err = to.accountsDataStor.commitUncertain(blockID0)
	assert.NoError(t, err)
	entry, err = to.accountsDataStor.retrieveNewestStringEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestStringEntry failed")
	assert.Equal(t, entry1, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveStringEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveStringEntry failed")
	assert.Equal(t, entry1, entry)
}

func TestRetrieveBinaryEntry(t *testing.T) {
	to := createAccountsDataStorage(t, true)

	to.stor.addBlock(t, blockID0)
	addr0 := testGlobal.senderInfo.addr
	entry0 := &proto.BinaryDataEntry{Key: "TheKey", Value: []byte{0xaa, 0xff}}
	err := to.accountsDataStor.appendEntry(addr0, entry0, blockID0)
	assert.NoError(t, err)
	entry, err := to.accountsDataStor.retrieveNewestBinaryEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveNewestBinaryEntry() failed")
	assert.Equal(t, entry0, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveBinaryEntry(addr0, entry0.Key)
	assert.NoError(t, err, "retrieveBinaryEntry() failed")
	assert.Equal(t, entry0, entry)

	// Test uncertain.
	entry1 := &proto.BinaryDataEntry{Key: "Uncertain", Value: []byte("whatever")}
	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	entry, err = to.accountsDataStor.retrieveNewestBinaryEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestBinaryEntry failed")
	assert.Equal(t, entry1, entry)
	to.accountsDataStor.dropUncertain()
	_, err = to.accountsDataStor.retrieveNewestBinaryEntry(addr0, entry1.Key)
	assert.Error(t, err)

	to.accountsDataStor.appendEntryUncertain(addr0, entry1)
	err = to.accountsDataStor.commitUncertain(blockID0)
	assert.NoError(t, err)
	entry, err = to.accountsDataStor.retrieveNewestBinaryEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveNewestBinaryEntry failed")
	assert.Equal(t, entry1, entry)
	to.stor.flush(t)
	entry, err = to.accountsDataStor.retrieveBinaryEntry(addr0, entry1.Key)
	assert.NoError(t, err, "retrieveBinaryEntry failed")
	assert.Equal(t, entry1, entry)
}
