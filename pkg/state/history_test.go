package state

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	rollbackEdge = 3500
	totalBlocks  = 4000
)

type historyTestObjects struct {
	rb      *recentBlocks
	fmt     *historyFormatter
	stateDB *stateDB
}

func flushHistory(t *testing.T, to *historyTestObjects) {
	to.rb.flush()
	err := to.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	to.stateDB.reset()
}

func createHistory() (*historyTestObjects, []string, error) {
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, nil, err
	}
	res := []string{dbDir0}
	db, err := keyvalue.NewKeyVal(dbDir0, defaultTestBloomFilterParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	rb, err := newRecentBlocks(rollbackMaxBlocks, nil)
	if err != nil {
		return nil, res, err
	}
	fmt, err := newHistoryFormatter(crypto.SignatureSize+1, crypto.SignatureSize, stateDB, rb)
	if err != nil {
		return nil, res, err
	}
	return &historyTestObjects{rb, fmt, stateDB}, res, nil
}

func TestAddRecord(t *testing.T) {
	to, path, err := createHistory()
	assert.NoError(t, err, "createHistory() failed")

	defer func() {
		err = to.stateDB.close()
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
		err = to.stateDB.close()
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
			addBlock(t, to.stateDB, to.rb, blockID)
		}
	}
	history, err = to.fmt.normalize(history, true)
	assert.NoError(t, err, "normalize() failed")
	height, err := to.rb.height()
	assert.NoError(t, err, "height() failed")
	oldRecordNumber := 0
	for i := 0; i <= len(history)-crypto.SignatureSize; i += crypto.SignatureSize {
		record := history[i : i+crypto.SignatureSize]
		blockID, err := crypto.NewSignatureFromBytes(record)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		recordHeight, err := to.rb.blockIDToHeight(blockID)
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
