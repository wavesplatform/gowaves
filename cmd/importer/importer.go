package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
	"go.uber.org/zap"
)

const (
	MiB = 1024 * 1024
)

var (
	logLevel                  = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	cfgPath                   = flag.String("cfg-path", "", "Path to blockchain settings JSON file for custom blockchains. Not set by default.")
	blockchainType            = flag.String("blockchain-type", "mainnet", "Blockchain type. Allowed values: mainnet/testnet/stagenet/custom. Default is 'mainnet'.")
	blockchainPath            = flag.String("blockchain-path", "", "Path to binary blockchain file.")
	balancesPath              = flag.String("balances-path", "", "Path to JSON with correct balances after applying blocks.")
	dataDirPath               = flag.String("data-path", "", "Path to directory with previously created state.")
	nBlocks                   = flag.Int("blocks-number", 1000, "Number of blocks to import.")
	verificationGoroutinesNum = flag.Int("verification-goroutines-num", runtime.NumCPU()*2, " Number of goroutines that will be run for verification of transactions/blocks signatures.")
	writeBufferSize           = flag.Int("write-buffer", 16, "Write buffer size in MiB.")
	buildDataForExtendedApi   = flag.Bool("build-extended-api", false, "Build and store additional data required for extended API in state. WARNING: this slows down the import, use only if you do really need extended API.")
	buildStateHashes          = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
	// Debug.
	cpuProfilePath = flag.String("cpuprofile", "", "Write cpu profile to this file.")
	memProfilePath = flag.String("memprofile", "", "Write memory profile to this file.")
)

func main() {
	flag.Parse()

	common.SetupLogger(*logLevel)
	zap.S().Infof("Gowaves Importer version: %s", versioning.Version)

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}

	if *blockchainPath == "" {
		zap.S().Fatalf("You must specify blockchain-path option.")
	}
	if *dataDirPath == "" {
		zap.S().Fatalf("You must specify data-path option.")
	}

	// Debug.
	if *cpuProfilePath != "" {
		f, err := os.Create(*cpuProfilePath)
		if err != nil {
			zap.S().Fatal("Could not create CPU profile: ", err)
		}
		defer func() { _ = f.Close() }()
		if err := pprof.StartCPUProfile(f); err != nil {
			zap.S().Fatal("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// https://godoc.org/github.com/coocood/freecache#NewCache
	debug.SetGCPercent(20)

	var ss *settings.BlockchainSettings
	if strings.ToLower(*blockchainType) == "custom" && *cfgPath != "" {
		f, err := os.Open(*cfgPath)
		if err != nil {
			zap.S().Fatalf("Failed to open custom blockchain settings: %v", err)
		}
		defer func() { _ = f.Close() }()
		ss, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			zap.S().Fatalf("Failed to read custom blockchain settings: %v", err)
		}
	} else {
		ss, err = settings.BlockchainSettingsByTypeName(*blockchainType)
		if err != nil {
			zap.S().Fatalf("Failed to load blockchain settings: %v", err)
		}
	}
	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = int(maxFDs - 10)
	params.VerificationGoroutinesNum = *verificationGoroutinesNum
	params.DbParams.WriteBuffer = *writeBufferSize * MiB
	params.StoreExtendedApiData = *buildDataForExtendedApi
	params.BuildStateHashes = *buildStateHashes
	// We do not need to provide any APIs during import.
	params.ProvideExtendedApi = false

	st, err := state.NewState(*dataDirPath, false, params, ss)
	if err != nil {
		zap.S().Fatalf("Failed to create state: %v", err)
	}

	defer func() {
		if err := st.Close(); err != nil {
			zap.S().Fatalf("Failed to close State: %v", err)
		}
	}()

	height, err := st.Height()
	if err != nil {
		zap.S().Fatalf("Failed to get current height: %v", err)
	}
	start := time.Now()
	if err := importer.ApplyFromFile(st, *blockchainPath, uint64(*nBlocks), height); err != nil {
		height, err1 := st.Height()
		if err1 != nil {
			zap.S().Fatalf("Failed to get current height: %v", err1)
		}
		zap.S().Fatalf("Failed to apply blocks after height %d: %v", height, err)
	}
	elapsed := time.Since(start)
	zap.S().Infof("Import took %s", elapsed)
	if len(*balancesPath) != 0 {
		if err := importer.CheckBalances(st, *balancesPath); err != nil {
			zap.S().Fatalf("Balances check failed: %v", err)
		}
	}

	// Debug.
	if *memProfilePath != "" {
		f, err := os.Create(*memProfilePath)
		if err != nil {
			zap.S().Fatal("Could not create memory profile: ", err)
		}
		defer func() { _ = f.Close() }()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			zap.S().Fatal("Could not write memory profile: ", err)
		}
	}
}
