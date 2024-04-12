package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
)

const (
	MiB = 1024 * 1024
)

func main() {
	err := run()
	if err != nil {
		zap.S().Error(capitalize(err.Error()))
		os.Exit(1)
	}
}

type cfg struct {
	logLevel                  *zapcore.Level
	cfgPath                   string
	blockchainType            string
	blockchainPath            string
	balancesPath              string
	dataDirPath               string
	nBlocks                   int
	verificationGoroutinesNum int
	writeBufferSize           int
	buildDataForExtendedAPI   bool
	buildStateHashes          bool
	lightNodeMode             bool
	snapshotsPath             string
	cpuProfilePath            string
	memProfilePath            string
}

func parseFlags() (cfg, error) {
	const (
		defaultBlocksNumber = 1000
		defaultBufferSize   = 16
	)
	c := cfg{}
	c.logLevel = zap.LevelFlag("log-level", zapcore.InfoLevel,
		"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	flag.StringVar(&c.cfgPath, "cfg-path", "",
		"Path to blockchain settings JSON file for custom blockchains. Not set by default.")
	flag.StringVar(&c.blockchainType, "blockchain-type", "mainnet",
		"Blockchain type. Allowed values: mainnet/testnet/stagenet/custom. Default is 'mainnet'.")
	flag.StringVar(&c.blockchainPath, "blockchain-path", "", "Path to binary blockchain file.")
	flag.StringVar(&c.balancesPath, "balances-path", "",
		"Path to JSON with correct balances after applying blocks.")
	flag.StringVar(&c.dataDirPath, "data-path", "", "Path to directory with previously created state.")
	flag.IntVar(&c.nBlocks, "blocks-number", defaultBlocksNumber, "Number of blocks to import.")
	flag.IntVar(&c.verificationGoroutinesNum, "verification-goroutines-num", runtime.NumCPU()*2,
		" Number of goroutines that will be run for verification of transactions/blocks signatures.")
	flag.IntVar(&c.writeBufferSize, "write-buffer", defaultBufferSize, "Write buffer size in MiB.")
	flag.BoolVar(&c.buildDataForExtendedAPI, "build-extended-api", false,
		"Build and store additional data required for extended API in state. "+
			"WARNING: this slows down the import, use only if you do really need extended API.")
	flag.BoolVar(&c.buildStateHashes, "build-state-hashes", false,
		"Calculate and store state hashes for each block height.")
	flag.BoolVar(&c.lightNodeMode, "light-node", false,
		"Run the node in the light mode in which snapshots are imported without validation")
	flag.StringVar(&c.snapshotsPath, "snapshots-path", "", "Path to binary snapshots file.")
	// Debug.
	flag.StringVar(&c.cpuProfilePath, "cpuprofile", "", "Write cpu profile to this file.")
	flag.StringVar(&c.memProfilePath, "memprofile", "", "Write memory profile to this file.")
	flag.Parse()

	if c.blockchainPath == "" {
		return cfg{}, errors.New("option blockchain-path is not specified, please specify it")
	}
	if c.dataDirPath == "" {
		return cfg{}, errors.New("option data-path is not specified, please specify it")
	}
	if c.lightNodeMode && c.snapshotsPath == "" {
		return cfg{}, errors.New("option snapshots-path is not specified in light mode, please specify it")
	}

	return c, nil
}

func (c *cfg) params(maxFDs int) state.StateParams {
	const clearance = 10
	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = maxFDs - clearance
	params.VerificationGoroutinesNum = c.verificationGoroutinesNum
	params.DbParams.WriteBuffer = c.writeBufferSize * MiB
	params.StoreExtendedApiData = c.buildDataForExtendedAPI
	params.BuildStateHashes = c.buildStateHashes
	params.ProvideExtendedApi = false // We do not need to provide any APIs during import.
	params.LightNodeMode = c.lightNodeMode
	return params
}

func (c *cfg) setupLogger() func() {
	logger := logging.SetupSimpleLogger(*c.logLevel)
	return func() {
		if sErr := logger.Sync(); sErr != nil && errors.Is(sErr, os.ErrInvalid) {
			zap.S().Errorf("Failed to close logging subsystem: %v", sErr)
		}
	}
}

func (c *cfg) setupCPUProfile() (func(), error) {
	if c.cpuProfilePath == "" {
		return func() {}, nil
	}
	f, err := os.Create(c.cpuProfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CPU profile: %w", err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}
	return func() {
		pprof.StopCPUProfile()
		if clErr := f.Close(); clErr != nil {
			zap.S().Errorf("Failed to close CPU profile: %v", clErr)
		}
	}, nil
}

func run() error {
	c, err := parseFlags()
	if err != nil {
		return err
	}

	logSync := c.setupLogger()
	defer logSync()

	zap.S().Infof("Gowaves Importer version: %s", versioning.Version)

	fds, err := riseFDLimit()
	if err != nil {
		return err
	}

	// Debug.
	cpfClose, err := c.setupCPUProfile()
	if err != nil {
		return err
	}
	defer cpfClose()

	// https://godoc.org/github.com/coocood/freecache#NewCache
	debug.SetGCPercent(20)

	ss, err := configureBlockchainSettings(c.blockchainType, c.cfgPath)
	if err != nil {
		return err
	}

	st, err := state.NewState(c.dataDirPath, false, c.params(fds), ss, false)
	if err != nil {
		return fmt.Errorf("failed to create state: %w", err)
	}
	defer func() {
		if clErr := st.Close(); clErr != nil {
			zap.S().Errorf("Failed to close State: %v", clErr)
		}
	}()

	imp, impClose, err := selectImporter(c, ss, st)
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}
	defer impClose()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	height, err := st.Height()
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}
	if height > 1 {
		zap.S().Infof("Skipping to height %d", height)
		if skErr := imp.SkipToHeight(ctx, height); skErr != nil {
			return fmt.Errorf("failed to skip to state height: %w", skErr)
		}
	}

	start := time.Now()
	if impErr := imp.Import(ctx, uint64(c.nBlocks)); impErr != nil {
		currentHeight, hErr := st.Height()
		if hErr != nil {
			zap.S().Fatalf("Failed to get current height: %v", hErr)
		}
		switch {
		case errors.Is(impErr, context.Canceled):
			zap.S().Infof("Interrupted by user, height %d", currentHeight)
		case errors.Is(impErr, io.EOF):
			zap.S().Info("End of blockchain file reached, height %d", currentHeight)
		default:
			zap.S().Fatalf("Failed to apply blocks after height %d: %v", currentHeight, impErr)
		}
	}
	elapsed := time.Since(start)
	zap.S().Infof("Import took %s", elapsed)

	if len(c.balancesPath) != 0 {
		if balErr := importer.CheckBalances(st, c.balancesPath); balErr != nil {
			return fmt.Errorf("failed to check balances: %w", balErr)
		}
	}

	// Debug.
	if mpfErr := configureMemProfile(c.memProfilePath); mpfErr != nil {
		return mpfErr
	}

	return nil
}

func selectImporter(c cfg, ss *settings.BlockchainSettings, st importer.State) (importer.Importer, func(), error) {
	if c.lightNodeMode {
		imp, err := importer.NewSnapshotsImporter(ss.AddressSchemeCharacter, st, c.blockchainPath, c.snapshotsPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create snapshots importer: %w", err)
		}
		return imp, func() {
			if clErr := imp.Close(); clErr != nil {
				zap.S().Errorf("Failed to close snapshots importer: %v", clErr)
			}
		}, nil
	}
	imp, err := importer.NewBlocksImporter(ss.AddressSchemeCharacter, st, c.blockchainPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create blocks importer: %w", err)
	}
	return imp, func() {
		if clErr := imp.Close(); clErr != nil {
			zap.S().Errorf("Failed to close blocks importer: %v", clErr)
		}
	}, nil
}

func configureMemProfile(memProfilePath string) error {
	if memProfilePath == "" {
		return nil
	}
	f, err := os.Create(filepath.Clean(memProfilePath))
	if err != nil {
		return fmt.Errorf("failed to create memory profile: %w", err)
	}
	defer func() {
		if clErr := f.Close(); clErr != nil {
			zap.S().Errorf("Failed to close memory profile: %v", clErr)
		}
	}()
	runtime.GC() // get up-to-date statistics
	if err = pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}
	return nil
}

func configureBlockchainSettings(blockchainType, cfgPath string) (*settings.BlockchainSettings, error) {
	var ss *settings.BlockchainSettings
	if strings.ToLower(blockchainType) == "custom" && cfgPath != "" {
		f, err := os.Open(filepath.Clean(cfgPath))
		if err != nil {
			return nil, fmt.Errorf("failed to open custom blockchain settings: %w", err)
		}
		defer func() {
			if clErr := f.Close(); clErr != nil {
				zap.S().Errorf("Failed to close custom blockchain settings: %v", clErr)
			}
		}()
		ss, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read custom blockchain settings: %w", err)
		}
		return ss, nil
	}
	ss, err := settings.BlockchainSettingsByTypeName(blockchainType)
	if err != nil {
		return nil, fmt.Errorf("failed to load blockchain settings: %w", err)
	}
	return ss, nil
}

func riseFDLimit() (int, error) {
	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		return 0, fmt.Errorf("failed to initialize importer: %w", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize importer: %w", err)
	}
	return int(maxFDs), nil
}

func capitalize(str string) string {
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
