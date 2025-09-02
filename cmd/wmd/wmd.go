package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
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
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var version = "0.0.0"

const (
	defaultSyncInterval = 10
	defaultTimeout      = 30 * time.Second
)

func run() error {
	// Parse command line parameters and set up configuration
	var (
		lp         = logging.Parameters{}
		importFile = flag.String("import-file", "",
			"Path to binary blockchain file to import before starting synchronization.")
		node = flag.String("node", "127.0.0.1:6870",
			"Address of the node's gRPC API endpoint. Default value: 127.0.0.1:6870.")
		interval = flag.Int("sync-interval", defaultSyncInterval,
			"Synchronization interval, seconds. Default interval is 10 seconds.")
		lag = flag.Int("lag", 1,
			"Synchronization lag behind the node, blocks. Default value 1 block.")
		address = flag.String("address", ":6990",
			"Local network address to bind the HTTP API of the service on. Default value is :6990.")
		db           = flag.String("db", "", "Path to data base folder. No default value.")
		matchersList = flag.String("matchers",
			"E3UwaHCQCySghK3zwNB8EDHoc3b8uhzGPFz3gHmWon4W,"+
				"7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy,"+
				"9cpfKN9suPNvfeUNphzxXMjcnn974eme8ZhWUjaktzU5",
			"Matcher's public keys in form of Base58 string, comma separated.")
		oracle = flag.String("oracle", "3P661nhk56WzFHCmQNKXjZGADxLHNY3LxP3",
			"Address of the tickers oracle, default for MainNet")
		scheme      = flag.String("scheme", "W", "Blockchain scheme symbol. Defaults to 'W'.")
		symbolsFile = flag.String("symbols", "", "Path to file of symbol substitutions. No default value.")
		rollback    = flag.Int("rollback", 0,
			"The height to rollback to before importing a blockchain file or staring the synchronization. "+
				"Default value is 0 (no rollback).")
		profilerPort = flag.Int("profiler-port", 0,
			"Start HTTP profiler on given port (port must be between 1024 and 65535)")
		cpuProfileFile = flag.String("cpu-profile", "", "Write CPU profile to the specified file")
	)
	lp.Initialize()
	flag.Parse()
	if err := lp.Parse(); err != nil {
		return err
	}

	// Set up log
	lh := logging.DefaultHandler(lp)
	slog.SetDefault(slog.New(lh))

	// Get a channel that will be closed on shutdown signals (CTRL-C) or shutdown request
	interrupt := interruptListener()
	defer slog.Info("Shutdown complete")

	slog.Info("Waves Market Data (WMD)", "version", version)

	// Enable http profiling server if requested
	if *profilerPort != 0 {
		go func() {
			listenAddr := fmt.Sprintf(":%d", *profilerPort)
			slog.Info("Profile server listening", "address", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof", http.StatusSeeOther)
			h := http.NewServeMux()
			h.Handle("/", profileRedirect)
			s := &http.Server{
				Addr:              listenAddr,
				Handler:           h,
				ReadHeaderTimeout: defaultTimeout,
				ReadTimeout:       defaultTimeout,
			}
			if lErr := s.ListenAndServe(); lErr != nil {
				slog.Error("Failed to listen", logging.Error(lErr))
			}
		}()
	}

	// Write cpu profile if requested
	if *cpuProfileFile != "" {
		f, err := os.Create(*cpuProfileFile)
		if err != nil {
			slog.Error("Unable to create CPU profile", logging.Error(err))
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			slog.Error("Failed to start CPU profiling", logging.Error(err))
			return err
		}
		defer func() {
			pprof.StopCPUProfile()
			err := f.Close()
			if err != nil {
				slog.Error("Failed to close CPU profile file", logging.Error(err))
			}
		}()
	}

	if len(*scheme) != 1 {
		err := errors.Errorf("incorrect blockchain scheme '%s', expected one character", *scheme)
		slog.Error("Invalid configuration", logging.Error(err))
		return err
	}
	sch := (*scheme)[0]

	if *node == "" {
		err := errors.New("empty node address")
		slog.Error("Failed to parse node's API address", logging.Error(err))
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
		slog.Error("Invalid configuration", logging.Error(err))
		return err
	}

	storage := state.Storage{Path: *db, Scheme: sch}
	err := storage.Open()
	if err != nil {
		slog.Error("Failed to open the storage", logging.Error(err))
		return err
	}
	defer func() {
		slog.Info("Closing the storage...")
		err := storage.Close()
		if err != nil {
			slog.Error("Failed to close the storage", logging.Error(err))
		}
		slog.Info("Storage closed")
	}()

	if interruptRequested(interrupt) {
		return nil
	}

	if *rollback != 0 {
		slog.Info("Rollback was requested, rolling back...", "height", *rollback)
		rh, err := storage.SafeRollbackHeight(*rollback)
		if err != nil {
			slog.Error("Failed to find the correct height of rollback", logging.Error(err))
			return nil
		}
		slog.Info("Nearest correct height of rollback", "height", rh)
		err = storage.Rollback(rh)
		if err != nil {
			slog.Error("Failed to rollback", slog.Int("height", rh), logging.Error(err))
			return nil
		}
		slog.Info("Successfully rolled back", "height", rh)
	}

	if interruptRequested(interrupt) {
		return nil
	}

	matchers := make([]crypto.PublicKey, 0)
	for ms := range strings.SplitSeq(*matchersList, ",") {
		pk, err := crypto.NewPublicKeyFromBase58(strings.TrimSpace(ms))
		if err != nil {
			slog.Error("Failed to parse matcher's public key", slog.String("key", ms), logging.Error(err))
			return err
		}
		matchers = append(matchers, pk)
	}
	if len(matchers) == 0 {
		slog.Error("Empty matchers list")
		return err
	}

	oracleAddr, err := proto.NewAddressFromString(*oracle)
	if err != nil {
		slog.Error("Incorrect oracle's address", logging.Error(err))
		return err
	}

	symbols, err := data.NewSymbolsFromFile(*symbolsFile, oracleAddr, sch)
	if err != nil {
		slog.Error("Failed to load symbol substitutions", logging.Error(err))
		return nil
	}
	slog.Info("Imported symbol substitutions", "count", symbols.Count())

	h, err := storage.Height()
	if err != nil {
		slog.Warn("Failed to get current height", logging.Error(err))
	}
	slog.Info("Last stored height", "height", h)

	if interruptRequested(interrupt) {
		return nil
	}

	if *importFile != "" {
		if _, err := os.Stat(*importFile); errors.Is(err, fs.ErrNotExist) {
			slog.Error("Failed to import blockchain from file", logging.Error(err))
			return err
		}
		importer := internal.NewImporter(interrupt, sch, &storage, matchers)
		err := importer.Import(*importFile)
		if err != nil {
			slog.Error("Failed to import blockchain file", slog.String("file", *importFile), logging.Error(err))
			return err
		}
	}

	if interruptRequested(interrupt) {
		return nil
	}

	var apiDone <-chan struct{}
	if *address != "" {
		api := internal.NewDataFeedAPI(interrupt, lh, &storage, *address, symbols)
		apiDone = api.Done()
	}

	if interruptRequested(interrupt) {
		return nil
	}

	var synchronizerDone <-chan struct{}
	s, err := internal.NewSynchronizer(interrupt, &storage, sch, matchers, *node,
		time.Duration(*interval)*time.Second, *lag, symbols)
	if err != nil {
		slog.Error("Failed to start synchronization", logging.Error(err))
		return err
	}
	synchronizerDone = s.Done()

	if apiDone != nil {
		<-apiDone
		slog.Info("API shutdown complete")
	}
	if synchronizerDone != nil {
		<-synchronizerDone
		slog.Info("Synchronizer shutdown complete")
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
