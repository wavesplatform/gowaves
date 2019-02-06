package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"os"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/storage"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	GENESIS_SIG = "5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	genesisSig     = flag.String("genesis-sig", "", "Signature of genesis block.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to import.")
	batchSize      = flag.Int("batch-size", 1000, "Size of key value batch.")
)

func Import(nBlocks int, manager *state.StateManager) error {
	f, err := os.Open(*blockchainPath)
	if err != nil {
		return err
	}

	defer func() {
		if err = f.Close(); err != nil {
			log.Fatalf("Failed to close blockchain file: %v\n\n", err.Error())
		}
	}()

	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(f)
	for i := 0; i < nBlocks; i++ {
		if _, err := io.ReadFull(r, sb); err != nil {
			return err
		}
		s := binary.BigEndian.Uint32(sb)
		block := buf[:s]
		if _, err = io.ReadFull(r, block); err != nil {
			return err
		}
		if err := manager.AcceptAndVerifyBlockBinary(block, true); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if len(*blockchainPath) == 0 {
		log.Fatalf("You must specify blockchain-path option.")
	}
	if len(*genesisSig) == 0 {
		log.Fatalf("You must specify genesis-sig option.")
	}
	rw, rwPath, err := storage.CreateTestBlockReadWriter(*batchSize, 8, 8)
	if err != nil {
		log.Fatalf("CreateTesBlockReadWriter: %v\n", err)
	}
	idsFile, err := rw.BlockIdsFilePath()
	if err != nil {
		log.Fatalf("Failed to get path of ids file: %v\n", err)
	}
	stor, storPath, err := storage.CreateTestAccountsStorage(idsFile)
	if err != nil {
		log.Fatalf("CreateTestAccountStorage: %v\n", err)
	}

	defer func() {
		if err := rw.Close(); err != nil {
			log.Fatalf("Failed to close BlockReadWriter: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(rwPath); err != nil {
			log.Fatalf("Failed to clean data dirs: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(storPath); err != nil {
			log.Fatalf("Failed to clean data dirs: %v\n", err)
		}
	}()

	genesis, err := crypto.NewSignatureFromBase58(*genesisSig)
	if err != nil {
		log.Fatalf("Failed to decode genesis signature: %v\n", err)
	}
	manager, err := state.NewStateManager(genesis, stor, rw)
	if err != nil {
		log.Fatalf("Failed to create state manager.\n")
	}
	if err := Import(*nBlocks, manager); err != nil {
		log.Fatalf("Failed to import: %v\n", err)
	}
}
