package main

import (
	"flag"
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
		slog.Error("Failed to parse application parameters", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(logging.DefaultHandler(lp)))
	slog.Info("Gowaves Rollback", "version", versioning.Version)

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		slog.Error("Initialization failed", "error", err)
		os.Exit(1)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		slog.Error("Initialization failed", "error", err)
		os.Exit(1)
	}

	var cfg *settings.BlockchainSettings
	if *cfgPath != "" {
		f, err := os.Open(*cfgPath)
		if err != nil {
			slog.Error("Failed to open configuration file", "error", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		cfg, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			slog.Error("Failed to read configuration file", "error", err)
			return
		}
	} else {
		cfg, err = settings.BlockchainSettingsByTypeName(*blockchainType)
		if err != nil {
			slog.Error("Failed to load blockchain settings", "error", err)
			return
		}
	}

	params := state.DefaultStateParams()
	const fdSigma = 10
	c, err := safecast.ToInt(maxFDs - fdSigma)
	if err != nil {
		slog.Error("Initialization failed", "error", err)
		return
	}
	params.DbParams.OpenFilesCacheCapacity = c
	params.DbParams.DisableBloomFilter = *disableBloomFilter
	params.BuildStateHashes = *buildStateHashes
	params.StoreExtendedApiData = *buildExtendedAPI

	s, err := state.NewState(*statePath, true, params, cfg, false)
	if err != nil {
		slog.Error("Failed to open state", "error", err)
		return
	}
	defer func() {
		err = s.Close()
		if err != nil {
			slog.Error("Failed to close state", "error", err)
		}
	}()

	curHeight, err := s.Height()
	if err != nil {
		slog.Error("Failed to get current height", "error", err)
		return
	}

	slog.Info("Current height", "height", curHeight)

	err = s.RollbackToHeight(*height)
	if err != nil {
		slog.Error("Failed to rollback", "error", err)
		return
	}

	curHeight, err = s.Height()
	if err != nil {
		slog.Error("Failed to get current height", "error", err)
		return
	}

	slog.Info("Current height", "height", curHeight)
}
