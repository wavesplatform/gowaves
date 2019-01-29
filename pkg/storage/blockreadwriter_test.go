package storage

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	BATCH_SIZE = 1000
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to test on.")

	cached_blocks []*proto.Block
)

func init() {
	flag.Parse()
	if len(*blockchainPath) == 0 {
		log.Fatal("You must specify blockchain-path for testing.")
	}
}

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func readRealBlocks(nBlocks int) ([]*proto.Block, error) {
	if len(cached_blocks) >= nBlocks {
		return cached_blocks[:nBlocks], nil
	}
	f, err := os.Open(*blockchainPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = f.Close()
		if err != nil {
			fmt.Printf("Failed to close blockchain file: %v\n\n", err.Error())
		}
	}()

	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(f)
	var blocks []*proto.Block
	for i := 0; i < nBlocks; i++ {
		_, err := io.ReadFull(r, sb)
		if err != nil {
			return nil, err
		}
		s := binary.BigEndian.Uint32(sb)
		bb := buf[:s]
		_, err = io.ReadFull(r, bb)
		if err != nil {
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

func testSingleBlock(rw *BlockReadWriter, block *proto.Block, t *testing.T) {
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
	if bytes.Compare(block.Transactions[:len(block.Transactions)-1], resTransactions) != 0 {
		t.Error("Transaction bytes are not equal.")
	}
}

func TestReadWrite(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "db_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rwDir, err := ioutil.TempDir("", "rw_dir")
	if err != nil {
		t.Fatalf("Can not create dir for test data: %v", err)
	}
	rw, err := createBlockReadWriter(dbDir, rwDir, 8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}
	blocks, err := readRealBlocks(*nBlocks)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	for _, block := range blocks {
		testSingleBlock(rw, block, t)
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

func TestRemoveBlocks(t *testing.T) {
	//rw, err := createBlockReadWriter(8, 8)
	//if err != nil {
	//	t.Fatalf("createBlockReadWriter: %v", err)
	//}
}

func TestConcurrentReadWrite(t *testing.T) {
	//rw, err := createBlockReadWriter(8, 8)
	//if err != nil {
	//	t.Fatalf("createBlockReadWriter: %v", err)
	//}
}

func TestBlockIDByHeight(t *testing.T) {
	//rw, err := createBlockReadWriter(8, 8)
	//if err != nil {
	//	t.Fatalf("createBlockReadWriter: %v", err)
	//}
}
