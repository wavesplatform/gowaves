package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/storage"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	balancesPath   = flag.String("balances-path", "", "Path to JSON with correct balances after applying blocks.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to import.")
	batchSize      = flag.Int("batch-size", 1000, "Size of key value batch.")
)

func main() {
	flag.Parse()
	if len(*blockchainPath) == 0 {
		log.Fatalf("You must specify blockchain-path option.")
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

	manager, err := state.NewStateManager(stor, rw)
	if err != nil {
		log.Fatalf("Failed to create state manager: %v.\n", err)
	}
	start := time.Now()
	if err := state.Apply(*blockchainPath, *nBlocks, manager); err != nil {
		log.Fatalf("Failed to apply blocks: %v\n", err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Import took %s\n", elapsed)
	if len(*balancesPath) != 0 {
		if err := state.CheckBalances(*balancesPath, stor); err != nil {
			log.Fatalf("CheckBalances(): %v\n", err)
		}
	}
}
