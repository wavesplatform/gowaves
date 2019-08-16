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

func createHistory(recordSize int) (*historyTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	fmt, err := newHistoryFormatter(recordSize, idSize, stor.stateDB, stor.rw)
	if err != nil {
		return nil, path, err
	}
	return &historyTestObjects{stor, fmt}, path, nil
}

func TestAddRecord(t *testing.T) {
	to, path, err := createHistory(idSize + 1)
	assert.NoError(t, err, "createHistory() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	var history []byte
	id := make([]byte, idSize)
	_, err = rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	// Test record rewrite.
	firstRecord := append([]byte{0}, id...)
	history, err = to.fmt.addRecord(history, firstRecord)
	assert.NoError(t, err, "addRecord() failed")
	secondRecord := append([]byte{1}, id...)
	history, err = to.fmt.addRecord(history, secondRecord)
	assert.NoError(t, err, "addRecord() failed")
	if !bytes.Equal(history, secondRecord) {
		t.Errorf("History formatter did not rewrite record with same ID.")
	}
	// Test record append.
	_, err = rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	thirdRecord := append([]byte{2}, id...)
	history, err = to.fmt.addRecord(history, thirdRecord)
	assert.NoError(t, err, "addRecord() failed")
	if !bytes.Equal(history, append(secondRecord, thirdRecord...)) {
		t.Errorf("History formatter did not append record with new ID.")
	}
}

func TestNormalize(t *testing.T) {
	to, path, err := createHistory(idSize)
	assert.NoError(t, err, "createHistory() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	var history []byte
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
		history, err = to.fmt.addRecord(history, blockNumBytes)
		assert.NoError(t, err, "addRecord() failed")
	}
	for _, id := range idsToRollback {
		err = to.stor.stateDB.rollbackBlock(id)
		assert.NoError(t, err, "rollbackBlock() failed")
	}
	history, err = to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	height := to.stor.rw.recentHeight()
	oldRecordNumber := 0
	for i := 0; i <= len(history)-idSize; i += idSize {
		record := history[i : i+idSize]
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
