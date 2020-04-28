package main

import (
	"flag"

	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

var (
	logLevel       = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath      = flag.String("state-path", "", "Path to node's state directory")
	blockchainType = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	height         = flag.Uint64("height", 0, "Height to rollback")
)

func main() {
	flag.Parse()
	err := setMaxOpenFiles(1024)
	if err != nil {
		zap.S().Fatalf("Failed to setup MaxOpenFiles: %v", err)
	}

	common.SetupLogger(*logLevel)

	cfg, err := settings.BlockchainSettingsByTypeName(*blockchainType)
	if err != nil {
		zap.S().Error(err)
		return
	}
	params := state.DefaultStateParams()
	state, err := state.NewState(*statePath, params, cfg)
	if err != nil {
		zap.S().Error(err)
		return
	}

	curHeight, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Infof("current height: %d", curHeight)

	err = state.RollbackToHeight(*height)
	if err != nil {
		zap.S().Error(err)
		return
	}

	curHeight, err = state.Height()
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Infof("current height: %d", curHeight)
}
