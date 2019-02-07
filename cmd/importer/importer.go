package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/wavesplatform/gowaves/pkg/state"
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
	start := time.Now()
	if err := state.CheckState(*blockchainPath, *balancesPath, *batchSize, *nBlocks); err != nil {
		log.Fatalf("CheckState(): %v\n", err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Import took %s\n", elapsed)
}
