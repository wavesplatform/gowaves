package state

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	tasksChanBufferSize = 20
	readersNumber       = 5
	blocksNumber        = 1000
)

type readCommandType byte

const (
	readHeader readCommandType = iota
	readTxHeight
	readTxOffset
	readTx
	readBlock
	getIDByHeight
)

type readTask struct {
	taskType      readCommandType
	txID          []byte
	blockID       crypto.Signature
	height        uint64
	offset        uint64
	correctResult []byte
}

func createBlockReadWriter(offsetLen, headerOffsetLen int) (*blockReadWriter, []string, error) {
	res := make([]string, 2)
	dbDir, err := ioutil.TempDir(os.TempDir(), "db_dir")
	if err != nil {
		return nil, nil, err
	}
	res[0] = dbDir
	db, err := keyvalue.NewKeyVal(dbDir, defaultTestKeyValParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	rwDir, err := ioutil.TempDir(os.TempDir(), "rw_dir")
	if err != nil {
		return nil, res, err
	}
	res[1] = rwDir
	rw, err := newBlockReadWriter(rwDir, offsetLen, headerOffsetLen, db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	return rw, res, nil
}

func writeBlock(t *testing.T, rw *blockReadWriter, block *proto.Block) {
	blockID := block.BlockSignature
	if err := rw.startBlock(blockID); err != nil {
		t.Fatalf("startBlock(): %v", err)
	}
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		t.Fatalf("MarshalHeaderToBinary(): %v", err)
	}
	if err := rw.writeBlockHeader(blockID, headerBytes); err != nil {
		t.Fatalf("writeBlockHeader(): %v", err)
	}
	transaction := block.Transactions.BytesUnchecked()
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transaction[0:4]))
		txBytes := transaction[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			t.Fatalf("Can not unmarshal tx: %v", err)
		}
		txID, err := tx.GetID()
		if err != nil {
			t.Fatalf("tx.GetID(): %v\n", err)
		}
		if err := rw.writeTransaction(txID, transaction[4:n+4]); err != nil {
			t.Fatalf("writeTransaction(): %v", err)
		}
		transaction = transaction[4+n:]
	}
	if err := rw.finishBlock(blockID); err != nil {
		t.Fatalf("finishBlock(): %v", err)
	}
	if err := rw.flush(); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	if err := rw.db.Flush(rw.dbBatch); err != nil {
		t.Fatalf("Failed to flush DB: %v", err)
	}
}

func testSingleBlock(t *testing.T, rw *blockReadWriter, block *proto.Block) {
	writeBlock(t, rw, block)
	blockID := block.BlockSignature
	resHeaderBytes, err := rw.readBlockHeader(blockID)
	if err != nil {
		t.Fatalf("readBlockHeader(): %v", err)
	}
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		t.Fatalf("MarshalHeaderToBinary(): %v", err)
	}
	if !bytes.Equal(headerBytes, resHeaderBytes) {
		t.Error("Header bytes are not equal.")
	}
	resTransactions, err := rw.readTransactionsBlock(blockID)
	if err != nil {
		t.Fatalf("readTransactionsBlock(): %v", err)
	}
	if !bytes.Equal(block.Transactions.BytesUnchecked(), resTransactions) {
		t.Error("Transaction bytes are not equal.")
	}
}

func writeBlocks(ctx context.Context, rw *blockReadWriter, blocks []proto.Block, readTasks chan<- *readTask, flush bool) error {
	height := 1
	offset := 0
	for _, block := range blocks {
		var tasksBuf []*readTask
		blockID := block.BlockSignature
		if err := rw.startBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		task := &readTask{taskType: getIDByHeight, height: uint64(height), correctResult: blockID[:]}
		tasksBuf = append(tasksBuf, task)
		headerBytes, err := block.MarshalHeaderToBinary()
		if err != nil {
			close(readTasks)
			return err
		}
		if err := rw.writeBlockHeader(blockID, headerBytes); err != nil {
			close(readTasks)
			return err
		}
		task = &readTask{taskType: readHeader, blockID: blockID, correctResult: headerBytes}
		tasksBuf = append(tasksBuf, task)
		transaction := block.Transactions.BytesUnchecked()
		for i := 0; i < block.TransactionCount; i++ {
			n := int(binary.BigEndian.Uint32(transaction[0:4]))
			txBytes := transaction[4 : n+4]
			tx, err := proto.BytesToTransaction(txBytes)
			if err != nil {
				close(readTasks)
				return err
			}
			txID, err := tx.GetID()
			if err != nil {
				return err
			}
			if err := rw.writeTransaction(txID, transaction[4:n+4]); err != nil {
				close(readTasks)
				return err
			}
			task = &readTask{taskType: readTx, txID: txID, correctResult: transaction[4 : n+4]}
			tasksBuf = append(tasksBuf, task)
			task = &readTask{taskType: readTxHeight, txID: txID, height: uint64(height)}
			tasksBuf = append(tasksBuf, task)
			task = &readTask{taskType: readTxOffset, txID: txID, offset: uint64(offset)}
			tasksBuf = append(tasksBuf, task)
			offset += n + 4
			transaction = transaction[4+n:]
		}
		if err := rw.finishBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		if flush {
			if err := rw.flush(); err != nil {
				close(readTasks)
				return err
			}
			if err := rw.db.Flush(rw.dbBatch); err != nil {
				close(readTasks)
				return err
			}
		}
		task = &readTask{taskType: readBlock, blockID: blockID, correctResult: block.Transactions.BytesUnchecked()}
		tasksBuf = append(tasksBuf, task)
		for _, task := range tasksBuf {
			select {
			case <-ctx.Done():
				close(readTasks)
				return ctx.Err()
			case readTasks <- task:
			}
		}
		height++
	}
	close(readTasks)
	return nil
}

func testNewestReader(rw *blockReadWriter, readTasks <-chan *readTask) error {
	for task := range readTasks {
		switch task.taskType {
		case readHeader:
			headerBytes, err := rw.readNewestBlockHeader(task.blockID)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, headerBytes) {
				return errors.New("Header bytes are not equal.")
			}
		case readTxHeight:
			height, err := rw.newestTransactionHeightByID(task.txID)
			if err != nil {
				return err
			}
			if height != task.height {
				return errors.New("Transaction heights are not equal")
			}
		case readTxOffset:
			offset, err := rw.newestTransactionOffsetByID(task.txID)
			if err != nil {
				return err
			}
			if offset != task.offset {
				return errors.New("Transaction offsets are not equal")
			}
		case readTx:
			tx, err := rw.readNewestTransaction(task.txID)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, tx) {
				return errors.New("Transaction bytes are not equal.")
			}
		case getIDByHeight:
			id, err := rw.newestBlockIDByHeight(task.height)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, id[:]) {
				return errors.Errorf("Got wrong ID %s by height %d", string(id[:]), task.height)
			}
		}
	}
	return nil
}

func testReader(rw *blockReadWriter, readTasks <-chan *readTask) error {
	for task := range readTasks {
		switch task.taskType {
		case readHeader:
			headerBytes, err := rw.readBlockHeader(task.blockID)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, headerBytes) {
				return errors.New("Header bytes are not equal.")
			}
		case readBlock:
			resTransactions, err := rw.readTransactionsBlock(task.blockID)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, resTransactions) {
				return errors.New("Transactions bytes are not equal.")
			}
		case readTxHeight:
			height, err := rw.transactionHeightByID(task.txID)
			if err != nil {
				return err
			}
			if height != task.height {
				return errors.New("Transaction heights are not equal")
			}
		case readTxOffset:
			offset, err := rw.transactionOffsetByID(task.txID)
			if err != nil {
				return err
			}
			if offset != task.offset {
				return errors.New("Transaction offsets are not equal")
			}
		case readTx:
			tx, err := rw.readTransaction(task.txID)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, tx) {
				return errors.New("Transaction bytes are not equal.")
			}
		case getIDByHeight:
			id, err := rw.blockIDByHeight(task.height)
			if err != nil {
				return err
			}
			if !bytes.Equal(task.correctResult, id[:]) {
				return errors.Errorf("Got wrong ID %s by height %d", string(id[:]), task.height)
			}
		}
	}
	return nil
}

func TestSimpleReadWrite(t *testing.T) {
	rw, path, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}

	defer func() {
		if err := rw.close(); err != nil {
			t.Fatalf("Failed to close blockReadWriter: %v", err)
		}
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	for _, block := range blocks {
		testSingleBlock(t, rw, &block)
	}
}

func TestSimultaneousReadWrite(t *testing.T) {
	rw, path, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}

	defer func() {
		if err := rw.close(); err != nil {
			t.Fatalf("Failed to close blockReadWriter: %v", err)
		}
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	var mtx sync.Mutex
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	errCounter := 0
	readTasks := make(chan *readTask, tasksChanBufferSize)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 := writeBlocks(ctx, rw, blocks, readTasks, true)
		if err1 != nil {
			mtx.Lock()
			errCounter++
			mtx.Unlock()
			fmt.Printf("Writer error: %v\n", err1)
			cancel()
		}
	}()
	for i := 0; i < readersNumber; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err1 := testReader(rw, readTasks)
			if err1 != nil {
				mtx.Lock()
				errCounter++
				mtx.Unlock()
				fmt.Printf("Reader error: %v\n", err1)
				cancel()
			}
		}()
	}
	wg.Wait()
	if errCounter != 0 {
		t.Fatalf("Reader/writer error.")
	}
}

func TestReadNewest(t *testing.T) {
	rw, path, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}

	defer func() {
		if err := rw.close(); err != nil {
			t.Fatalf("Failed to close blockReadWriter: %v", err)
		}
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	var mtx sync.Mutex
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	errCounter := 0
	readTasks := make(chan *readTask, tasksChanBufferSize)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 := writeBlocks(ctx, rw, blocks, readTasks, false)
		if err1 != nil {
			mtx.Lock()
			errCounter++
			mtx.Unlock()
			fmt.Printf("Writer error: %v\n", err1)
			cancel()
		}
	}()
	for i := 0; i < readersNumber; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err1 := testNewestReader(rw, readTasks)
			if err1 != nil {
				mtx.Lock()
				errCounter++
				mtx.Unlock()
				fmt.Printf("Reader error: %v\n", err1)
				cancel()
			}
		}()
	}
	wg.Wait()
	if errCounter != 0 {
		t.Fatalf("Reader/writer error.")
	}
}

func TestSimultaneousReadDelete(t *testing.T) {
	rw, path, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}

	defer func() {
		if err := rw.close(); err != nil {
			t.Fatalf("Failed to close blockReadWriter: %v", err)
		}
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}

	for _, block := range blocks {
		writeBlock(t, rw, &block)
	}
	idToTest := blocks[blocksNumber-2].BlockSignature
	prevId := blocks[blocksNumber-3].BlockSignature
	txs, err := blocks[blocksNumber-2].Transactions.Transactions()
	if err != nil {
		t.Fatalf("Transactions() failed: %v", err)
	}
	txID, err := txs[0].GetID()
	if err != nil {
		t.Fatalf("GetID(): %v", err)
	}

	var wg sync.WaitGroup
	var removeErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Give some time to start reading before deleting.
		time.Sleep(time.Second)
		removeErr = rw.rollback(prevId, true)
	}()
	for {
		_, err = rw.readBlockHeader(idToTest)
		if err != nil {
			if err == keyvalue.ErrNotFound {
				// Successfully removed.
				break
			}
			t.Fatalf("readBlockHeader(): %v", err)
		}
		_, err = rw.readTransactionsBlock(idToTest)
		if err != nil {
			if err == keyvalue.ErrNotFound {
				// Successfully removed.
				break
			}
			t.Fatalf("readTransactionsBlock(): %v", err)
		}
	}
	wg.Wait()
	if removeErr != nil {
		t.Fatalf("Failed to remove blocks: %v", err)
	}
	_, err = rw.readTransaction(txID)
	if err != keyvalue.ErrNotFound {
		t.Fatalf("transaction from removed block wasn't deleted %v", err)
	}
}
