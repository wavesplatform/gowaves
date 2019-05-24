package state

import (
	"bytes"
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
	stor *storageObjects
	fmt  *historyFormatter
}

func createHistory() (*historyTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	fmt, err := newHistoryFormatter(crypto.SignatureSize+1, crypto.SignatureSize, stor.stateDB, stor.rb)
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

	var history []byte
	blockID := make([]byte, crypto.SignatureSize)
	_, err = rand.Read(blockID)
	assert.NoError(t, err, "rand.Read() failed")
	// Test record rewrite.
	firstRecord := append([]byte{0}, blockID...)
	history, err = to.fmt.addRecord(history, firstRecord)
	assert.NoError(t, err, "addRecord() failed")
	secondRecord := append([]byte{1}, blockID...)
	history, err = to.fmt.addRecord(history, secondRecord)
	assert.NoError(t, err, "addRecord() failed")
	if !bytes.Equal(history, secondRecord) {
		t.Errorf("History formatter did not rewrite record with same ID.")
	}
	// Test record append.
	_, err = rand.Read(blockID)
	assert.NoError(t, err, "rand.Read() failed")
	thirdRecord := append([]byte{2}, blockID...)
	history, err = to.fmt.addRecord(history, thirdRecord)
	assert.NoError(t, err, "addRecord() failed")
	if !bytes.Equal(history, append(secondRecord, thirdRecord...)) {
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

	var history []byte
	for i := 0; i < totalBlocks; i++ {
		blockIDBytes := make([]byte, crypto.SignatureSize)
		_, err = rand.Read(blockIDBytes)
		assert.NoError(t, err, "rand.Read() failed")
		blockID, err := crypto.NewSignatureFromBytes(blockIDBytes)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		history, err = to.fmt.addRecord(history, blockID[:])
		assert.NoError(t, err, "addRecord() failed")
		if i <= rollbackEdge {
			to.stor.addBlock(t, blockID)
		}
	}
	history, err = to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	height, err := to.stor.rb.height()
	assert.NoError(t, err, "height() failed")
	oldRecordNumber := 0
	for i := 0; i <= len(history)-crypto.SignatureSize; i += crypto.SignatureSize {
		record := history[i : i+crypto.SignatureSize]
		blockID, err := crypto.NewSignatureFromBytes(record)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		recordHeight, err := to.stor.rb.blockIDToHeight(blockID)
		assert.NoError(t, err, "blockIDToHeight failed")
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
