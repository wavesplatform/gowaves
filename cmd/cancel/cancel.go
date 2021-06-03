package main

import (
	"errors"
	"flag"
	"os"
	"runtime"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

const (
	MB = 1024 * 1024
)

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	var (
		statePath      string
		blockchainType string
		extendedAPI    bool
	)

	common.SetupLogger("INFO")

	flag.StringVar(&statePath, "state-path", "", "Path to node's state folder")
	flag.StringVar(&blockchainType, "blockchain-type", "mainnet", "Blockchain type mainnet/testnet/stagenet, default value is mainnet")
	flag.BoolVar(&extendedAPI, "extended-api", false, "Open state with extended API")
	flag.Parse()

	if statePath == "" || len(strings.Fields(statePath)) > 1 {
		zap.S().Errorf("Invalid path to state '%s'", statePath)
		return errors.New("invalid state path")
	}

	ss, err := settings.BlockchainSettingsByTypeName(blockchainType)
	if err != nil {
		zap.S().Errorf("Failed to load blockchain settings: %v", err)
		return err
	}

	params := state.DefaultStateParams()
	params.VerificationGoroutinesNum = 2 * runtime.NumCPU()
	params.DbParams.WriteBuffer = 16 * MB
	params.StoreExtendedApiData = extendedAPI
	params.BuildStateHashes = true
	params.ProvideExtendedApi = false
	st, err := state.NewState(statePath, params, ss)
	if err != nil {
		zap.S().Errorf("Failed to open state at '%s': %v", statePath, err)
		return err
	}
	defer func() {
		if err := st.Close(); err != nil {
			zap.S().Fatalf("Failed to close State: %v", err)
		}
	}()

	ls, err := st.LeasesToStolenAliases()
	if err != nil {
		zap.S().Errorf("Failed to get leases to stolen aliases: %v", err)
		return err
	}
	for _, m := range ls {
		zap.S().Info(m)
	}
	zap.S().Info("DONE")
	return nil
}
