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
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	BATCH_SIZE = 1000
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")

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
		blocks = append(blocks, &block)
	}
	cached_blocks = blocks
	return blocks, nil
}

func createBlockReadWriter(offsetLen, headerOffsetLen int) (*BlockReadWriter, error) {
	// TODO remove (clean) all the temp dirs.
	dbDir, err := ioutil.TempDir("", "db_dir")
	if err != nil {
		return nil, err
	}
	keyVal, err := keyvalue.NewKeyVal(dbDir, BATCH_SIZE)
	if err != nil {
		return nil, err
	}
	rwDir, err := ioutil.TempDir("", "rw_dir")
	if err != nil {
		return nil, err
	}
	return NewBlockReadWriter(rwDir, offsetLen, headerOffsetLen, keyVal)
}

func TestReadWrite(t *testing.T) {
	rw, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter: %v", err)
	}
	blocks, err := readRealBlocks(10)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v", err)
	}
	block := blocks[5]
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
	transactions := block.Transactions
	for i := 0; i < block.TransactionCount; i++ {
		n := int(binary.BigEndian.Uint32(transactions[0:4]))
		txBytes := transactions[4 : n+4]
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			t.Fatalf("Can not unmarshal tx: %v", err)
		}
		if err := rw.WriteTransaction(tx.GetID(), txBytes); err != nil {
			t.Fatalf("WriteTransaction(): %v", err)
		}
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
	if bytes.Compare(transactions, resTransactions) != 0 {
		t.Error("Transaction bytes are not equal.")
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
