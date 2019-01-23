package main

import (
	"context"
	"flag"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		logLevel    = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		importFile  = flag.String("import-file", "", "Path to binary blockchain file to import before starting synchronization.")
		node        = flag.String("node", "http://127.0.0.1:6869", "URL of node API. Default value http://127.0.0.1:6869.")
		address     = flag.String("address", ":6990", "Local network address to bind HTTP API of the service.")
		db          = flag.String("db", "", "Path to data base.")
		matcher     = flag.String("matcher", "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "Matcher's public key in form of Base58 string.")
		scheme      = flag.String("scheme", "W", "Blockchain scheme symbol. Defaults to 'W'.")
		symbolsFile = flag.String("symbols", "", "Path to file of symbols substitutions.")
		rollback    = flag.Int("rollback", 0, "The height to rollback to before importing file or staring synchronization.")
	)
	flag.Parse()

	logger, log := setupLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("Failed to close logging subsystem: %s", err.Error())
		}
	}()

	if len(*scheme) != 1 {
		log.Fatalf("Incorrect blockchain scheme symbol '%s', expected one character.", *scheme)
		shutdown()
	}

	_, err := url.Parse(*node)
	if err != nil {
		log.Errorf("Failed to parse node API address: %s", err.Error())
		shutdown()
	}

	appCtx, cancel := context.WithCancel(context.Background())

	if *db == "" {
		log.Error("No data base path specified")
		shutdown()
	}
	sch := (byte)((*scheme)[0])
	storage := state.Storage{Path: *db, Scheme: sch}
	err = storage.Open()
	if err != nil {
		log.Errorf("Failed to open storage: %s", err.Error())
		shutdown()
	}
	defer func() {
		err := storage.Close()
		if err != nil {
			log.Errorf("Failed to close Storage: %s", err.Error())
		}
	}()

	if *rollback != 0 {
		rh, err := storage.SafeRollbackHeight(*rollback)
		if err != nil {
			log.Errorf("Failed to find the correct height of rollback: %s", err.Error())
			shutdown()
		}
		log.Infof("Nearest correct height of rollback: %d", rh)
		err = storage.Rollback(rh)
		if err != nil {
			log.Errorf("Failed to rollback to height %d: %s", rh, err.Error())
			shutdown()
		}
		log.Infof("Successfully rolled back to height %d", rh)
	}

	matcherPK, err := crypto.NewPublicKeyFromBase58(*matcher)
	if err != nil {
		log.Errorf("Incorrect matcher's address: %s", err.Error())
		shutdown()
	}

	symbols, err := data.ImportSymbols(*symbolsFile)
	if err != nil {
		log.Errorf("Failed to load symbols substitutions: %s", err.Error())
		shutdown()
	}
	log.Debugf("Imported %d of symbols substitution", symbols.Count())

	h, err := storage.Height()
	if err != nil {
		log.Warnf("Failed to get current height: %s", err.Error())
	}
	log.Infof("Last stored height: %d", h)

	err = importBlockchainIfNeeded(appCtx, log, *importFile, sch, &storage, matcherPK)
	if err != nil {
		log.Errorf("Initial blockchain import failed: %s", err.Error())
	}

	df := internal.NewDataFeedAPI(log, &storage, symbols)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(internal.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Mount("/api", df.Routes())
	err = http.ListenAndServe(*address, r)
	if err != nil {
		log.Fatalf("Failed to bind API: %s", err.Error())
		shutdown()
	}

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Debugf("Caught signal '%s', stopping", sig)
		cancel()
		shutdown()
	}

}

func importBlockchainIfNeeded(ctx context.Context, log *zap.SugaredLogger, n string, scheme byte, storage *state.Storage, matcher crypto.PublicKey) error {
	if n != "" {
		if _, err := os.Stat(n); os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to import blockchain file '%s'", n)
		}
		i := internal.NewImporter(ctx, log, scheme, storage, matcher)
		err := i.Import(n)
		if err != nil {
			return errors.Wrapf(err, "failed to import blockchain file '%s'", n)
		}
	}
	return nil
}

func setupLogger(level string) (*zap.Logger, *zap.SugaredLogger) {
	al := zap.NewAtomicLevel()
	switch strings.ToUpper(level) {
	case "DEBUG":
		al.SetLevel(zap.DebugLevel)
	case "INFO":
		al.SetLevel(zap.InfoLevel)
	case "ERROR":
		al.SetLevel(zap.ErrorLevel)
	case "WARN":
		al.SetLevel(zap.WarnLevel)
	case "FATAL":
		al.SetLevel(zap.FatalLevel)
	default:
		al.SetLevel(zap.InfoLevel)
	}
	ec := zap.NewDevelopmentEncoderConfig()
	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.Lock(os.Stdout), al))
	return logger, logger.Sugar()
}

func shutdown() {
	os.Exit(0)
}
