package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
)

func main() {
	var (
		logLevel = zap.LevelFlag("log-level", zapcore.InfoLevel,
			"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		statePath        = flag.String("state-path", "", "Path to node's state directory")
		blockchainType   = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
		height           = flag.Uint64("height", 0, "Height to rollback")
		buildExtendedAPI = flag.Bool("build-extended-api", false,
			"Builds extended API. "+
				"Note that state must be re-imported in case it wasn't imported with similar flag set")
		buildStateHashes = flag.Bool("build-state-hashes", false,
			"Calculate and store state hashes for each block height.")
		cfgPath            = flag.String("cfg-path", "", "Path to configuration JSON file, only for custom blockchain.")
		disableBloomFilter = flag.Bool("disable-bloom", false, "Disable bloom filter for state.")
	)

	flag.Parse()

	logger := logging.SetupSimpleLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil && errors.Is(err, os.ErrInvalid) {
			panic(fmt.Sprintf("Failed to close logging subsystem: %v\n", err))
		}
	}()
	zap.S().Infof("Gowaves Rollback version: %s", versioning.Version)

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}

	var cfg *settings.BlockchainSettings
	if *cfgPath != "" {
		f, err := os.Open(*cfgPath)
		if err != nil {
			zap.S().Fatalf("Failed to open configuration file: %v", err)
		}
		defer func() { _ = f.Close() }()
		cfg, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			zap.S().Fatalf("Failed to read configuration file: %v", err)
		}
	} else {
		cfg, err = settings.BlockchainSettingsByTypeName(*blockchainType)
		if err != nil {
			zap.S().Error(err)
			return
		}
	}

	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = int(maxFDs - 10)
	params.DbParams.BloomFilterParams.Disable = *disableBloomFilter
	params.BuildStateHashes = *buildStateHashes
	params.StoreExtendedApiData = *buildExtendedAPI

	s, err := state.NewState(*statePath, true, params, cfg, false)
	if err != nil {
		zap.S().Error(err)
		return
	}
	defer func() {
		err = s.Close()
		if err != nil {
			zap.S().Errorf("Failed to close state: %v", err)
		}
	}()

	curHeight, err := s.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Infof("Current height: %d", curHeight)

	err = s.RollbackToHeight(*height)
	if err != nil {
		zap.S().Error(err)
		return
	}

	curHeight, err = s.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Infof("Current height: %d", curHeight)
}
