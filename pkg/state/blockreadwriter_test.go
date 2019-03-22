package state

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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

var (
	cachedBlocks []proto.Block
)

type readCommandType byte

const (
	readHeader readCommandType = iota
	readTx
	readBlock
	getIDByHeight
)

type readTask struct {
	taskType      readCommandType
	txID          []byte
	blockID       crypto.Signature
	height        uint64
	correctResult []byte
}

func readRealBlocks(t *testing.T, nBlocks int) ([]proto.Block, error) {
	if len(cachedBlocks) >= nBlocks {
		return cachedBlocks[:nBlocks], nil
	}
	dir, err := getLocalDir()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(dir, "testdata", "blocks-10000"))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = f.Close(); err != nil {
			t.Logf("Failed to close blockchain file: %v\n\n", err.Error())
		}
	}()

	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(f)
	var blocks []proto.Block
	for i := 0; i < nBlocks; i++ {
		if _, err := io.ReadFull(r, sb); err != nil {
			return nil, err
		}
		s := binary.BigEndian.Uint32(sb)
		bb := buf[:s]
		if _, err = io.ReadFull(r, bb); err != nil {
			return nil, err
		}
		var block proto.Block
		if err = block.UnmarshalBinary(bb); err != nil {
			return nil, err
		}
		if !crypto.Verify(block.GenPublicKey, block.BlockSignature, bb[:len(bb)-crypto.SignatureSize]) {
			return nil, errors.Errorf("Block %d has invalid signature", i)
		}
		blocks = append(blocks, block)
	}
	cachedBlocks = blocks
	return blocks, nil
}

func createBlockReadWriter(offsetLen, headerOffsetLen int) (*blockReadWriter, []string, error) {
	res := make([]string, 2)
	dbDir, err := ioutil.TempDir(os.TempDir(), "db_dir")
	if err != nil {
		return nil, res, err
	}
	db, err := keyvalue.NewKeyVal(dbDir)
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
	rw, err := newBlockReadWriter(rwDir, offsetLen, headerOffsetLen, db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	res = []string{dbDir, rwDir}
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
	transaction := block.Transactions
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transaction[0:4]))
		txBytes := transaction[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			t.Fatalf("Can not unmarshal tx: %v", err)
		}
		if err := rw.writeTransaction(tx.GetID(), transaction[:n+4]); err != nil {
			t.Fatalf("writeTransaction(): %v", err)
		}
		transaction = transaction[4+n:]
	}
	if err := rw.finishBlock(blockID); err != nil {
		t.Fatalf("finishBlock(): %v", err)
	}
	if err := rw.updateHeight(1); err != nil {
		t.Fatalf("Failed to update height: %v", err)
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
	if bytes.Compare(headerBytes, resHeaderBytes) != 0 {
		t.Error("Header bytes are not equal.")
	}
	resTransactions, err := rw.readTransactionsBlock(blockID)
	if err != nil {
		t.Fatalf("readTransactionsBlock(): %v", err)
	}
	if bytes.Compare(block.Transactions, resTransactions) != 0 {
		t.Error("Transaction bytes are not equal.")
	}
}

func writeBlocks(ctx context.Context, rw *blockReadWriter, blocks []proto.Block, readTasks chan<- *readTask) error {
	height := 0
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
		transaction := block.Transactions
		for i := 0; i < block.TransactionCount; i++ {
			n := int(binary.BigEndian.Uint32(transaction[0:4]))
			txBytes := transaction[4 : n+4]
			tx, err := proto.BytesToTransaction(txBytes)
			if err != nil {
				close(readTasks)
				return err
			}
			if err := rw.writeTransaction(tx.GetID(), transaction[:n+4]); err != nil {
				close(readTasks)
				return err
			}
			task = &readTask{taskType: readTx, txID: tx.GetID(), correctResult: transaction[:n+4]}
			tasksBuf = append(tasksBuf, task)
			transaction = transaction[4+n:]
		}
		if err := rw.finishBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		if err := rw.updateHeight(1); err != nil {
			close(readTasks)
			return err
		}
		if err := rw.flush(); err != nil {
			close(readTasks)
			return err
		}
		if err := rw.db.Flush(rw.dbBatch); err != nil {
			close(readTasks)
			return err
		}
		task = &readTask{taskType: readBlock, blockID: blockID, correctResult: block.Transactions}
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

func testReader(rw *blockReadWriter, readTasks <-chan *readTask) error {
	for task := range readTasks {
		switch task.taskType {
		case readHeader:
			headerBytes, err := rw.readBlockHeader(task.blockID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.correctResult, headerBytes) != 0 {
				return errors.New("Header bytes are not equal.")
			}
		case readBlock:
			resTransactions, err := rw.readTransactionsBlock(task.blockID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.correctResult, resTransactions) != 0 {
				return errors.New("Transactions bytes are not equal.")
			}
		case readTx:
			tx, err := rw.readTransaction(task.txID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.correctResult, tx) != 0 {
				return errors.New("Transaction bytes are not equal.")
			}
		case getIDByHeight:
			id, err := rw.blockIDByHeight(task.height)
			if err != nil {
				return err
			}
			if bytes.Compare(task.correctResult, id[:]) != 0 {
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

	blocks, err := readRealBlocks(t, blocksNumber)
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

	blocks, err := readRealBlocks(t, blocksNumber)
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
		err1 := writeBlocks(ctx, rw, blocks, readTasks)
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

	blocks, err := readRealBlocks(t, blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}

	for _, block := range blocks {
		writeBlock(t, rw, &block)
	}
	idToTest := blocks[blocksNumber-1].BlockSignature
	prevId := blocks[blocksNumber-2].BlockSignature

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
			if err.Error() == "leveldb: not found" {
				// Successfully removed.
				break
			}
			t.Fatalf("readBlockHeader(): %v", err)
		}
		_, err = rw.readTransactionsBlock(idToTest)
		if err != nil {
			if err.Error() == "leveldb: not found" {
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
}
