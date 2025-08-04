package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ccoveille/go-safecast"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Failed to rollback", logging.Error(err))
		os.Exit(1)
	}
	slog.Info("Rollback completed successfully")
}

func run() error {
	var (
		lp               = logging.Parameters{}
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
	lp.Initialize()
	flag.Parse()
	if err := lp.Parse(); err != nil {
		return fmt.Errorf("failed to parse application parameters: %w", err)
	}

	slog.SetDefault(slog.New(logging.DefaultHandler(lp)))
	slog.Info("Gowaves Rollback", "version", versioning.Version)

	var cfg *settings.BlockchainSettings
	var err error
	if *cfgPath != "" {
		f, fErr := os.Open(*cfgPath)
		if fErr != nil {
			return fmt.Errorf("failed to open configuration file: %w", fErr)
		}
		defer func() { _ = f.Close() }()
		cfg, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	} else {
		cfg, err = settings.BlockchainSettingsByTypeName(*blockchainType)
		if err != nil {
			return fmt.Errorf("failed to load blockchain settins: %w", err)
		}
	}

	s, err := openState(*statePath, cfg, *buildExtendedAPI, *buildStateHashes, *disableBloomFilter)
	if err != nil {
		return fmt.Errorf("failed to open state: %w", err)
	}
	defer func() {
		if clErr := s.Close(); clErr != nil {
			slog.Error("Failed to close state", logging.Error(clErr))
		}
	}()

	curHeight, err := s.Height()
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	slog.Info("Current height", "height", curHeight)

	err = s.RollbackToHeight(*height)
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	curHeight, err = s.Height()
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	slog.Info("Current height", "height", curHeight)
	return nil
}

func openState(
	statePath string, cfg *settings.BlockchainSettings, buildExtendedAPI, buildStateHashes, disableBloomFilter bool,
) (state.State, error) {
	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	params := state.DefaultStateParams()
	const fdSigma = 10
	c, err := safecast.ToInt(maxFDs - fdSigma)
	if err != nil {
		return nil, fmt.Errorf("state initialization failed: %w", err)
	}
	params.DbParams.OpenFilesCacheCapacity = c
	params.DbParams.DisableBloomFilter = disableBloomFilter
	params.BuildStateHashes = buildStateHashes
	params.StoreExtendedApiData = buildExtendedAPI

	return state.NewState(statePath, true, params, cfg, false)
}
