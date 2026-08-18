package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ccoveille/go-safecast/v2"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
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
		compressionAlgo  keyvalue.CompressionAlgo
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
	flag.TextVar(&compressionAlgo, "db-compression-algo", keyvalue.CompressionDefault,
		fmt.Sprintf("Set the compression algorithm for the state database. Supported: %v",
			keyvalue.CompressionAlgoStrings(),
		),
	)
	lp.Initialize()
	flag.Parse()
	if err := lp.Parse(); err != nil {
		return fmt.Errorf("failed to parse application parameters: %w", err)
	}

	slog.SetDefault(slog.New(logging.DefaultHandler(lp)))
	slog.Info("Gowaves Rollback", "version", versioning.Version)

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

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
			return fmt.Errorf("failed to load blockchain settings: %w", err)
		}
	}

	s, err := openState(ctx, *statePath, cfg, *buildExtendedAPI, *buildStateHashes, *disableBloomFilter, compressionAlgo)
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

	err = s.RollbackToHeight(*height, false)
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
	ctx context.Context,
	statePath string,
	cfg *settings.BlockchainSettings,
	buildExtendedAPI, buildStateHashes, disableBloomFilter bool,
	compressionAlgo keyvalue.CompressionAlgo,
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
	c, err := safecast.Convert[int](maxFDs - fdSigma)
	if err != nil {
		return nil, fmt.Errorf("state initialization failed: %w", err)
	}
	params.DbParams.OpenFilesCacheCapacity = c
	params.DbParams.DisableBloomFilter = disableBloomFilter
	params.BuildStateHashes = buildStateHashes
	params.StoreExtendedApiData = buildExtendedAPI
	params.DbParams.CompressionAlgo = compressionAlgo

	return state.NewState(ctx, statePath, true, params, cfg, false, nil)
}
