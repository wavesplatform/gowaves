package main

import (
	"flag"

	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"go.uber.org/zap"
)

var (
	logLevel         = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath        = flag.String("state-path", "", "Path to node's state directory")
	blockchainType   = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	height           = flag.Uint64("height", 0, "Height to rollback")
	buildExtendedApi = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be reimported in case it wasn't imported with similar flag set")
	buildStateHashes = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
)

func main() {
	flag.Parse()

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}

	common.SetupLogger(*logLevel)

	cfg, err := settings.BlockchainSettingsByTypeName(*blockchainType)
	if err != nil {
		zap.S().Error(err)
		return
	}
	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = int(maxFDs - 10)
	params.BuildStateHashes = *buildStateHashes
	params.StoreExtendedApiData = *buildExtendedApi
	s, err := state.NewState(*statePath, params, cfg)
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
