package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

var version = "0.0.0"

const (
	defaultSyncInterval = 10
	defaultTimeout      = 30 * time.Second
)

func run() error {
	// Parse command line parameters and set up configuration
	var (
		logLevel       = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		importFile     = flag.String("import-file", "", "Path to binary blockchain file to import before starting synchronization.")
		node           = flag.String("node", "127.0.0.1:6870", "Address of the node's gRPC API endpoint. Default value: 127.0.0.1:6870.")
		interval       = flag.Int("sync-interval", defaultSyncInterval, "Synchronization interval, seconds. Default interval is 10 seconds.")
		lag            = flag.Int("lag", 1, "Synchronization lag behind the node, blocks. Default value 1 block.")
		address        = flag.String("address", ":6990", "Local network address to bind the HTTP API of the service on. Default value is :6990.")
		db             = flag.String("db", "", "Path to data base folder. No default value.")
		matchersList   = flag.String("matchers", "E3UwaHCQCySghK3zwNB8EDHoc3b8uhzGPFz3gHmWon4W,7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy,9cpfKN9suPNvfeUNphzxXMjcnn974eme8ZhWUjaktzU5", "Matcher's public keys in form of Base58 string, comma separated.")
		oracle         = flag.String("oracle", "3P661nhk56WzFHCmQNKXjZGADxLHNY3LxP3", "Address of the tickers oracle, default for MainNet")
		scheme         = flag.String("scheme", "W", "Blockchain scheme symbol. Defaults to 'W'.")
		symbolsFile    = flag.String("symbols", "", "Path to file of symbol substitutions. No default value.")
		rollback       = flag.Int("rollback", 0, "The height to rollback to before importing a blockchain file or staring the synchronization. Default value is 0 (no rollback).")
		profilerPort   = flag.Int("profiler-port", 0, "Start HTTP profiler on given port (port must be between 1024 and 65535)")
		cpuProfileFile = flag.String("cpu-profile", "", "Write CPU profile to the specified file")
	)
	flag.Parse()

	// Set up log
	logger, _ := common.SetupLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil && err == os.ErrInvalid {
			panic(fmt.Sprintf("Failed to close logging subsystem: %v\n", err))
		}
	}()

	// Get a channel that will be closed on shutdown signals (CTRL-C) or shutdown request
	interrupt := interruptListener()
	defer zap.S().Info("Shutdown complete")

	zap.S().Infof("Waves Market Data (WMD) version %s", version)

	// Enable http profiling server if requested
	if *profilerPort != 0 {
		go func() {
			listenAddr := fmt.Sprintf(":%d", *profilerPort)
			zap.S().Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof", http.StatusSeeOther)
			h := http.NewServeMux()
			h.Handle("/", profileRedirect)
			s := &http.Server{Addr: listenAddr, Handler: h, ReadHeaderTimeout: defaultTimeout, ReadTimeout: defaultTimeout}
			zap.S().Errorf("%v", s.ListenAndServe())
		}()
	}

	// Write cpu profile if requested
	if *cpuProfileFile != "" {
		f, err := os.Create(*cpuProfileFile)
		if err != nil {
			zap.S().Errorf("Unable to create CPU profile: %v", err)
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			zap.S().Errorf("Failed to start CPU profiling: %v", err)
			return err
		}
		defer func() {
			pprof.StopCPUProfile()
			err := f.Close()
			if err != nil {
				zap.S().Errorf("Failed to close CPU profile file: %v", err)
			}
		}()
	}

	if len(*scheme) != 1 {
		err := errors.Errorf("incorrect blockchain scheme '%s', expected one character", *scheme)
		zap.S().Errorf("Invalid configuration: %v", err)
		return err
	}
	sch := (*scheme)[0]

	if *node == "" {
		err := errors.New("empty node address")
		zap.S().Errorf("Failed to parse node's API address: %s", err.Error())
		return err
	}
	if *interval <= 0 {
		*interval = defaultSyncInterval
	}
	if *lag < 0 {
		*lag = 0
	}

	if *db == "" {
		err := errors.Errorf("no database path")
		zap.S().Errorf("Invalid configuration: %v", err)
		return err
	}

	storage := state.Storage{Path: *db, Scheme: sch}
	err := storage.Open()
	if err != nil {
		zap.S().Errorf("Failed to open the storage: %v", err)
		return err
	}
	defer func() {
		zap.S().Info("Closing the storage...")
		err := storage.Close()
		if err != nil {
			zap.S().Errorf("Failed to close the storage: %v", err)
		}
		zap.S().Info("Storage closed")
	}()

	if interruptRequested(interrupt) {
		return nil
	}

	if *rollback != 0 {
		zap.S().Infof("Rollback to height %d was requested, rolling back...", *rollback)
		rh, err := storage.SafeRollbackHeight(*rollback)
		if err != nil {
			zap.S().Errorf("Failed to find the correct height of rollback: %v", err)
			return nil
		}
		zap.S().Infof("Nearest correct height of rollback: %d", rh)
		err = storage.Rollback(rh)
		if err != nil {
			zap.S().Errorf("Failed to rollback to height %d: %v", rh, err)
			return nil
		}
		zap.S().Infof("Successfully rolled back to height %d", rh)
	}

	if interruptRequested(interrupt) {
		return nil
	}

	matchers := make([]crypto.PublicKey, 0)
	for _, ms := range strings.Split(*matchersList, ",") {
		pk, err := crypto.NewPublicKeyFromBase58(strings.TrimSpace(ms))
		if err != nil {
			zap.S().Errorf("Failed to parse matcher's public key '%s': %v", ms, err)
			return err
		}
		matchers = append(matchers, pk)
	}
	if len(matchers) == 0 {
		zap.S().Errorf("Empty matchers list")
		return err
	}

	oracleAddr, err := proto.NewAddressFromString(*oracle)
	if err != nil {
		zap.S().Errorf("Incorrect oracle's address: %v", err)
		return err
	}

	symbols, err := data.NewSymbolsFromFile(*symbolsFile, oracleAddr, sch)
	if err != nil {
		zap.S().Errorf("Failed to load symbol substitutions: %v", err)
		return nil
	}
	zap.S().Infof("Imported %d of symbol substitutions", symbols.Count())

	h, err := storage.Height()
	if err != nil {
		zap.S().Warnf("Failed to get current height: %s", err.Error())
	}
	zap.S().Infof("Last stored height: %d", h)

	if interruptRequested(interrupt) {
		return nil
	}

	if *importFile != "" {
		if _, err := os.Stat(*importFile); errors.Is(err, fs.ErrNotExist) {
			zap.S().Errorf("Failed to import blockchain from file: %v", err)
			return err
		}
		importer := internal.NewImporter(interrupt, sch, &storage, matchers)
		err := importer.Import(*importFile)
		if err != nil {
			zap.S().Errorf("Failed to import blockchain file '%s': %v", *importFile, err)
			return err
		}
	}

	if interruptRequested(interrupt) {
		return nil
	}

	var apiDone <-chan struct{}
	if *address != "" {
		api := internal.NewDataFeedAPI(interrupt, logger, &storage, *address, symbols)
		apiDone = api.Done()
	}

	if interruptRequested(interrupt) {
		return nil
	}

	var synchronizerDone <-chan struct{}
	s, err := internal.NewSynchronizer(interrupt, &storage, sch, matchers, *node, time.Duration(*interval)*time.Second, *lag, symbols)
	if err != nil {
		zap.S().Errorf("Failed to start synchronization: %v", err)
		return err
	}
	synchronizerDone = s.Done()

	if apiDone != nil {
		<-apiDone
		zap.S().Info("API shutdown complete")
	}
	if synchronizerDone != nil {
		<-synchronizerDone
		zap.S().Info("Synchronizer shutdown complete")
	}
	<-interrupt
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	debug.SetGCPercent(10)
	if err := run(); err != nil {
		os.Exit(1)
	}
}
