package state

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	blockID       proto.BlockID
	height        uint64
	offset        uint64
	correctTx     proto.Transaction
	correctHeader proto.BlockHeader
	correctBlock  proto.Block
	correctID     proto.BlockID
}

func testSingleBlock(t *testing.T, to *testStorageObjects, block *proto.Block) {
	rw := to.rw
	to.addRealBlock(t, block)
	blockID := block.BlockID()
	resHeader, err := rw.readBlockHeader(blockID)
	if err != nil {
		t.Fatalf("readBlockHeader(): %v", err)
	}
	assert.Equal(t, block.BlockHeader, *resHeader)
	resBlock, err := rw.readBlock(blockID)
	if err != nil {
		t.Fatalf("readBlock(): %v", err)
	}
	assert.Equal(t, resBlock, block)
}

func writeBlocks(ctx context.Context, rw *blockReadWriter, blocks []proto.Block, readTasks chan<- *readTask, flush, protobuf bool) error {
	const scheme = proto.MainNetScheme

	height := 1
	offset := 0
	for _, block := range blocks {
		var tasksBuf []*readTask
		blockID := block.BlockID()
		if err := rw.startBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		task := &readTask{taskType: getIDByHeight, height: uint64(height), correctID: blockID}
		tasksBuf = append(tasksBuf, task)
		if err := rw.writeBlockHeader(&block.BlockHeader); err != nil {
			close(readTasks)
			return err
		}
		task = &readTask{taskType: readHeader, blockID: blockID, correctHeader: block.BlockHeader}
		tasksBuf = append(tasksBuf, task)
		for i := range block.Transactions {
			tx := block.Transactions[i]
			txID, err := tx.GetID(scheme)
			if err != nil {
				return err
			}
			var txBytes []byte
			if protobuf {
				txBytes, err = tx.MarshalSignedToProtobuf(scheme)
				if err != nil {
					return err
				}
			} else {
				txBytes, err = tx.MarshalBinary(scheme)
				if err != nil {
					return err
				}
			}
			if err := rw.writeTransaction(tx, false); err != nil {
				close(readTasks)
				return err
			}
			task = &readTask{taskType: readTx, txID: txID, correctTx: tx}
			tasksBuf = append(tasksBuf, task)
			task = &readTask{taskType: readTxHeight, txID: txID, height: uint64(height)}
			tasksBuf = append(tasksBuf, task)
			task = &readTask{taskType: readTxOffset, txID: txID, offset: uint64(offset)}
			tasksBuf = append(tasksBuf, task)
			offset += len(txBytes) + 4
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
		task = &readTask{taskType: readBlock, blockID: blockID, correctBlock: block}
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
			header, err := rw.readNewestBlockHeader(task.blockID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctHeader, *header) {
				return errors.New("headers are not equal")
			}
		case readTxHeight:
			height, err := rw.newestTransactionHeightByID(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.height, height) {
				return errors.New("heights are not equal")
			}
		case readTxOffset:
			info, err := rw.newestTransactionInfoByID(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.offset, info.offset) {
				return errors.New("transaction offsets are not equal")
			}
		case readTx:
			tx, _, err := rw.readNewestTransaction(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctTx, tx) {
				return errors.New("transactions are not equal")
			}
		case getIDByHeight:
			id, err := rw.newestBlockIDByHeight(task.height)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctID, id) {
				return errors.Errorf("block IDs are not equal: correct: %s, actual: %s", task.correctID.String(), id.String())
			}
		}
	}
	return nil
}

func testReader(rw *blockReadWriter, readTasks <-chan *readTask) error {
	for task := range readTasks {
		switch task.taskType {
		case readHeader:
			header, err := rw.readBlockHeader(task.blockID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctHeader, *header) {
				return errors.New("headers are not equal")
			}
		case readBlock:
			block, err := rw.readBlock(task.blockID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctBlock, *block) {
				return errors.New("blocks are not equal")
			}
		case readTxHeight:
			height, err := rw.transactionHeightByID(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.height, height) {
				return errors.New("heights are not equal")
			}
		case readTxOffset:
			info, err := rw.transactionInfoByID(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.offset, info.offset) {
				return errors.New("transaction offsets are not equal")
			}
		case readTx:
			tx, _, err := rw.readTransaction(task.txID)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctTx, tx) {
				return errors.New("transactions are not equal")
			}
		case getIDByHeight:
			id, err := rw.blockIDByHeight(task.height)
			if err != nil {
				return err
			}
			if !assert.ObjectsAreEqual(task.correctID, id) {
				return errors.Errorf("block IDs are not equal: correct: %s, actual: %s", task.correctID.String(), id.String())
			}
		}
	}
	return nil
}

func TestSimpleReadWrite(t *testing.T) {
	to := createStorageObjects(t, true)

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	for _, block := range blocks {
		testSingleBlock(t, to, &block)
	}
}

func TestSimultaneousReadWrite(t *testing.T) {
	to := createStorageObjects(t, true)

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
		err1 := writeBlocks(ctx, to.rw, blocks, readTasks, true, false)
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
			err1 := testReader(to.rw, readTasks)
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
	to := createStorageObjects(t, true)

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
		err1 := writeBlocks(ctx, to.rw, blocks, readTasks, false, false)
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
			err1 := testNewestReader(to.rw, readTasks)
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
	to := createStorageObjects(t, true)

	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}

	for _, block := range blocks {
		to.addRealBlock(t, &block)
	}
	rollbackHeight := blocksNumber - 2
	idToTest := blocks[rollbackHeight].BlockID()
	txs := blocks[rollbackHeight].Transactions
	txID, err := txs[0].GetID(proto.MainNetScheme)
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
		removeErr = to.rw.rollback(uint64(rollbackHeight))
	}()
	for {
		_, err = to.rw.readBlockHeader(idToTest)
		if err != nil {
			if err == io.EOF {
				// Successfully removed.
				break
			}
			t.Fatalf("readBlockHeader(): %v", err)
		}
		_, err = to.rw.readBlock(idToTest)
		if err != nil {
			if err == io.EOF {
				// Successfully removed.
				break
			}
			t.Fatalf("readBlock(): %v", err)
		}
	}
	wg.Wait()
	if removeErr != nil {
		t.Fatalf("Failed to remove blocks: %v", err)
	}
	_, _, err = to.rw.readTransaction(txID)
	if err != io.EOF {
		t.Fatalf("transaction from removed block wasn't deleted %v", err)
	}
}

func TestProtobufReadWrite(t *testing.T) {
	to := createStorageObjects(t, true)

	// Activate protobuf and convert MainNet blocks to fake 'protobuf' ones.
	// This is needed because blockReadWriter only accepts
	// protobuf blocks after setProtobufActivated().
	to.rw.setProtobufActivated()
	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	protobufBlocks := make([]proto.Block, len(blocks))
	copy(protobufBlocks, blocks)
	prevId := proto.NewBlockIDFromDigest(crypto.Digest{})
	for i := range protobufBlocks {
		// Change blocks version to protobuf since we activated protobuf.
		protobufBlocks[i].Version = proto.ProtobufBlockVersion
		// Update parents.
		protobufBlocks[i].Parent = prevId
		// Regenerate ID.
		err = protobufBlocks[i].GenerateBlockID(proto.MainNetScheme)
		if err != nil {
			t.Fatalf("GenerateBlockID() failed: %v", err)
		}
		prevId = protobufBlocks[i].BlockID()
	}

	var mtx sync.Mutex
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	errCounter := 0
	readTasks := make(chan *readTask, tasksChanBufferSize)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 := writeBlocks(ctx, to.rw, protobufBlocks, readTasks, true, true)
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
			err1 := testReader(to.rw, readTasks)
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

func TestFailedTransactionReadWrite(t *testing.T) {
	//TODO: add test on failed transaction
}

func TestSyncWithDb(t *testing.T) {
	to := createStorageObjects(t, true)

	// Add block.
	err := to.rw.startBlock(blockID0)
	assert.NoError(t, err, "startBlock() failed")
	err = to.rw.finishBlock(blockID0)
	assert.NoError(t, err, "finishBlock() failed")

	to.flush(t)

	assert.Equal(t, uint64(1), to.rw.height)

	err = to.rw.syncWithDb()
	assert.NoError(t, err, "syncWithDb() failed")

	// Block that is not present in DB should be removed after sync.
	assert.Equal(t, uint64(0), to.rw.height)
}
