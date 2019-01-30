package storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	BATCH_SIZE             = 1000
	TASKS_CHAN_BUFFER_SIZE = 20
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to test on.")
	nReaders       = flag.Int("readers-number", 20, "Number of simultaneous readers.")

	cached_blocks []*proto.Block
)

func init() {
	flag.Parse()
	if len(*blockchainPath) == 0 {
		log.Fatal("You must specify blockchain-path for testing.")
	}
}

type ReadCommandType byte

const (
	ReadHeader ReadCommandType = iota
	ReadTx
	ReadBlock
	GetIDByHeight
)

type ReadTask struct {
	Type          ReadCommandType
	TxID          []byte
	BlockID       crypto.Signature
	Height        uint64
	CorrectResult []byte
}

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func readRealBlocks(t *testing.T, nBlocks int) ([]*proto.Block, error) {
	if len(cached_blocks) >= nBlocks {
		return cached_blocks[:nBlocks], nil
	}
	f, err := os.Open(*blockchainPath)
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
	var blocks []*proto.Block
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
		blocks = append(blocks, &block)
	}
	cached_blocks = blocks
	return blocks, nil
}

func createBlockReadWriter(dbDir, rwDir string, offsetLen, headerOffsetLen int) (*BlockReadWriter, error) {
	keyVal, err := keyvalue.NewKeyVal(dbDir, BATCH_SIZE)
	if err != nil {
		return nil, err
	}
	return NewBlockReadWriter(rwDir, offsetLen, headerOffsetLen, keyVal)
}

func testSingleBlock(t *testing.T, rw *BlockReadWriter, block *proto.Block) {
	blockID := block.BlockSignature
	if err := rw.StartBlock(blockID); err != nil {
		t.Fatalf("StartBlock(): %v", err)
	}
	headerBytes, err := block.MarshalHeaderToBinary()
	if err != nil {
		t.Fatalf("MarshalHeaderToBinary(): %v", err)
	}
	if err := rw.WriteBlockHeader(blockID, headerBytes); err != nil {
		t.Fatalf("WriteBlockHeader(): %v", err)
	}
	transaction := block.Transactions
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transaction[0:4]))
		txBytes := transaction[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			t.Fatalf("Can not unmarshal tx: %v", err)
		}
		if err := rw.WriteTransaction(tx.GetID(), transaction[:n+4]); err != nil {
			t.Fatalf("WriteTransaction(): %v", err)
		}
		transaction = transaction[4+n:]
	}
	if err := rw.FinishBlock(blockID); err != nil {
		t.Fatalf("FinishBlock(): %v", err)
	}
	resHeaderBytes, err := rw.ReadBlockHeader(blockID)
	if err != nil {
		t.Fatalf("ReadBlockHeader(): %v", err)
	}
	if bytes.Compare(headerBytes, resHeaderBytes) != 0 {
		t.Error("Header bytes are not equal.")
	}
	resTransactions, err := rw.ReadTransactionsBlock(blockID)
	if err != nil {
		t.Fatalf("ReadTransactionsBlock(): %v", err)
	}
	if bytes.Compare(block.Transactions, resTransactions) != 0 {
		t.Error("Transaction bytes are not equal.")
	}
}

func writeBlocks(ctx context.Context, rw *BlockReadWriter, blocks []*proto.Block, readTasks chan<- *ReadTask) error {
	height := 0
	for _, block := range blocks {
		var tasksBuf []*ReadTask
		blockID := block.BlockSignature
		if err := rw.StartBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		task := &ReadTask{Type: GetIDByHeight, Height: uint64(height), CorrectResult: blockID[:]}
		tasksBuf = append(tasksBuf, task)
		headerBytes, err := block.MarshalHeaderToBinary()
		if err != nil {
			close(readTasks)
			return err
		}
		if err := rw.WriteBlockHeader(blockID, headerBytes); err != nil {
			close(readTasks)
			return err
		}
		task = &ReadTask{Type: ReadHeader, BlockID: blockID, CorrectResult: headerBytes}
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
			if err := rw.WriteTransaction(tx.GetID(), transaction[:n+4]); err != nil {
				close(readTasks)
				return err
			}
			task = &ReadTask{Type: ReadTx, TxID: tx.GetID(), CorrectResult: transaction[:n+4]}
			tasksBuf = append(tasksBuf, task)
			transaction = transaction[4+n:]
		}
		if err := rw.FinishBlock(blockID); err != nil {
			close(readTasks)
			return err
		}
		task = &ReadTask{Type: ReadBlock, BlockID: blockID, CorrectResult: block.Transactions}
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

func testReader(rw *BlockReadWriter, readTasks <-chan *ReadTask) error {
	for task := range readTasks {
		switch task.Type {
		case ReadHeader:
			headerBytes, err := rw.ReadBlockHeader(task.BlockID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.CorrectResult, headerBytes) != 0 {
				return errors.New("Header bytes are not equal.")
			}
		case ReadBlock:
			resTransactions, err := rw.ReadTransactionsBlock(task.BlockID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.CorrectResult, resTransactions) != 0 {
				return errors.New("Transactions bytes are not equal.")
			}
		case ReadTx:
			tx, err := rw.ReadTransaction(task.TxID)
			if err != nil {
				return err
			}
			if bytes.Compare(task.CorrectResult, tx) != 0 {
				return errors.New("Transaction bytes are not equal.")
			}
		case GetIDByHeight:
			id, err := rw.BlockIDByHeight(task.Height)
			if err != nil {
				return err
			}
			if bytes.Compare(task.CorrectResult, id[:]) != 0 {
				return errors.Errorf("Got wrong ID %s by height %d", string(id[:]), task.Height)
			}
		}
	}
	return nil
}

func TestSimpleReadWrite(t *testing.T) {
	dbDir, err := ioutil.TempDir(os.TempDir(), "db_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rwDir, err := ioutil.TempDir(os.TempDir(), "rw_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rw, err := createBlockReadWriter(dbDir, rwDir, 8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}
	blocks, err := readRealBlocks(t, *nBlocks)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	for _, block := range blocks {
		testSingleBlock(t, rw, block)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("Failed to close BlockReadWriter: %v", err)
	}
	if err := os.RemoveAll(dbDir); err != nil {
		t.Fatalf("Failed to clean test data dirs: %v", err)
	}
	if err := os.RemoveAll(rwDir); err != nil {
		t.Fatalf("Failed to clean test data dirs: %v", err)
	}
}

func TestSimultaneousRWDelete(t *testing.T) {
	// TODO test deletion.
	dbDir, err := ioutil.TempDir(os.TempDir(), "db_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rwDir, err := ioutil.TempDir(os.TempDir(), "rw_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rw, err := createBlockReadWriter(dbDir, rwDir, 8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}
	blocks, err := readRealBlocks(t, *nBlocks)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	var mtx sync.Mutex
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	errCounter := 0
	readTasks := make(chan *ReadTask, TASKS_CHAN_BUFFER_SIZE)
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
	for i := 0; i < *nReaders; i++ {
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
	if err := rw.Close(); err != nil {
		t.Fatalf("Failed to close BlockReadWriter: %v", err)
	}
	if err := os.RemoveAll(dbDir); err != nil {
		t.Fatalf("Failed to clean test data dirs: %v", err)
	}
	if err := os.RemoveAll(rwDir); err != nil {
		t.Fatalf("Failed to clean test data dirs: %v", err)
	}
}
