package state

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	rollbackEdge = 3500
	totalBlocks  = 4000
)

type historyTestObjects struct {
	stor *testStorageObjects
	fmt  *historyFormatter
}

func createHistory() (*historyTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	fmt, err := newHistoryFormatter(stor.stateDB, stor.rw)
	if err != nil {
		return nil, path, err
	}
	return &historyTestObjects{stor, fmt}, path, nil
}

func TestAddRecord(t *testing.T) {
	to, path, err := createHistory()
	assert.NoError(t, err, "createHistory() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	id := make([]byte, idSize)
	_, err = rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	firstRecord := append([]byte{0}, id...)
	history := &historyRecord{fixedSize: true, recordSize: idSize + 1}
	// Test record rewrite.
	err = to.fmt.addRecord(history, firstRecord)
	assert.NoError(t, err, "addRecord() failed")
	secondRecord := append([]byte{1}, id...)
	err = to.fmt.addRecord(history, secondRecord)
	assert.NoError(t, err, "addRecord() failed")
	assert.Equal(t, 1, len(history.records))
	if !bytes.Equal(history.records[0], secondRecord) {
		t.Errorf("History formatter did not rewrite record with same ID.")
	}
	// Test record append.
	_, err = rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	thirdRecord := append([]byte{2}, id...)
	err = to.fmt.addRecord(history, thirdRecord)
	assert.NoError(t, err, "addRecord() failed")
	assert.Equal(t, 2, len(history.records))
	if !bytes.Equal(history.records[0], secondRecord) {
		t.Errorf("History formatter did not rewrite record with same ID.")
	}
	if !bytes.Equal(history.records[1], thirdRecord) {
		t.Errorf("History formatter did not append record with new ID.")
	}
}

func TestNormalize(t *testing.T) {
	to, path, err := createHistory()
	assert.NoError(t, err, "createHistory() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	history := &historyRecord{fixedSize: true, recordSize: idSize}
	var idsToRollback []crypto.Signature
	for i := 0; i < totalBlocks; i++ {
		blockIDBytes := make([]byte, crypto.SignatureSize)
		_, err = rand.Read(blockIDBytes)
		assert.NoError(t, err, "rand.Read() failed")
		blockID, err := crypto.NewSignatureFromBytes(blockIDBytes)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		to.stor.addBlock(t, blockID)
		if i > rollbackEdge {
			idsToRollback = append(idsToRollback, blockID)
		}
		blockNum, err := to.stor.stateDB.blockIdToNum(blockID)
		assert.NoError(t, err, "blockIdToNum() failed")
		blockNumBytes := make([]byte, idSize)
		binary.BigEndian.PutUint32(blockNumBytes, blockNum)
		err = to.fmt.addRecord(history, blockNumBytes)
		assert.NoError(t, err, "addRecord() failed")
	}
	for _, id := range idsToRollback {
		err = to.stor.stateDB.rollbackBlock(id)
		assert.NoError(t, err, "rollbackBlock() failed")
	}
	changed, err := to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	assert.Equal(t, true, changed)
	height := to.stor.rw.recentHeight()
	oldRecordNumber := 0
	for _, record := range history.records {
		blockNum := binary.BigEndian.Uint32(record)
		blockID, err := to.stor.stateDB.blockNumToId(blockNum)
		assert.NoError(t, err, "blockNumToId() failed")
		recordHeight, err := to.stor.rw.newestHeightByBlockID(blockID)
		assert.NoError(t, err, "newestHeightByBlockID() failed")
		if recordHeight < height-rollbackMaxBlocks {
			oldRecordNumber++
		}
		if recordHeight > rollbackEdge {
			t.Errorf("History formatter did not erase invalid blocks.")
		}
	}
	if oldRecordNumber > 1 {
		t.Errorf("History formatter did not cut old blocks.")
	}
}
