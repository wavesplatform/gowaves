package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

var (
	genesisCfgPath = flag.String("genesis-cfg-path", "", "Path to genesis JSON config for custom blockchains.")
	blockchainType = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/custom.")
	blockchainPath = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	balancesPath   = flag.String("balances-path", "", "Path to JSON with correct balances after applying blocks.")
	dataDirPath    = flag.String("data-path", "", "Path to directory with previously created state.")
	nBlocks        = flag.Int("blocks-number", 1000, "Number of blocks to import.")
)

func blockchainSettings() (*settings.BlockchainSettings, error) {
	switch *blockchainType {
	case "mainnet":
		return settings.MainNetSettings, nil
	case "testnet":
		return settings.TestNetSettings, nil
	case "custom":
		if *genesisCfgPath == "" {
			return nil, errors.New("for custom blockchains you have to specify path to your genesis JSON config")
		}
		return &settings.BlockchainSettings{Type: settings.Custom, GenesisCfgPath: *genesisCfgPath}, nil
	default:
		return nil, errors.New("invalid blockchain type")
	}
}

func main() {
	flag.Parse()
	if *blockchainPath == "" {
		log.Fatalf("You must specify blockchain-path option.")
	}
	ss, err := blockchainSettings()
	if err != nil {
		log.Fatalf("blockchainSettings: %v\n", err)
	}
	dataDir := *dataDirPath
	if dataDir == "" {
		tempDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
		if err != nil {
			log.Fatalf("Faied to create temp dir for data: %v\n", err)
		}
		dataDir = tempDir
	}
	state, err := state.NewState(dataDir, state.DefaultBlockStorageParams(), ss)
	if err != nil {
		log.Fatalf("Failed to create state: %v.\n", err)
	}

	defer func() {
		if err := state.Close(); err != nil {
			log.Fatalf("Failed to close State: %v\n", err)
		}
		if *dataDirPath == "" {
			if err := os.RemoveAll(dataDir); err != nil {
				log.Fatalf("Failed to clean data dir: %v\n", err)
			}
		}
	}()

	height, err := state.Height()
	if err != nil {
		log.Fatalf("Failed to get current height: %v\n", err)
	}
	start := time.Now()
	if err := importer.ApplyFromFile(state, *blockchainPath, uint64(*nBlocks), height); err != nil {
		height, err1 := state.Height()
		if err1 != nil {
			log.Fatalf("Failed to get current height: %v\n", err1)
		}
		log.Fatalf("Failed to apply blocks at height %d: %v\n", height, err)
	}
	elapsed := time.Since(start)
	fmt.Printf("Import took %s\n", elapsed)
	if len(*balancesPath) != 0 {
		if err := importer.CheckBalances(state, *balancesPath); err != nil {
			log.Fatalf("CheckBalances(): %v\n", err)
		}
	}
}
