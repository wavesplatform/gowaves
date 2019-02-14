package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/state"
)

var (
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	balancesPath   = flag.String("balances-path", "", "Path to JSON with correct balances after applying blocks.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to import.")
)

func main() {
	flag.Parse()
	if len(*blockchainPath) == 0 {
		log.Fatalf("You must specify blockchain-path option.")
	}
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		log.Fatalf("Faied to create temp dir for data: %v\n", err)
	}
	manager, err := state.NewStateManager(dataDir, state.DefaultBlockStorageParams())
	if err != nil {
		log.Fatalf("Failed to create state manager: %v.\n", err)
	}

	defer func() {
		if err := manager.Close(); err != nil {
			log.Fatalf("Failed to close StateManager: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			log.Fatalf("Failed to clean data dir: %v\n", err)
		}
	}()

	start := time.Now()
	if err := importer.ApplyFromFile(manager, *blockchainPath, *nBlocks, false); err != nil {
		log.Fatalf("Failed to apply blocks: %v\n", err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Import took %s\n", elapsed)
	if len(*balancesPath) != 0 {
		if err := importer.CheckBalances(manager, *balancesPath); err != nil {
			log.Fatalf("CheckBalances(): %v\n", err)
		}
	}
}
