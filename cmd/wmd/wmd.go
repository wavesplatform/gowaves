package main

import (
	"context"
	"flag"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal"
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
		logLevel   = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		importFile = flag.String("import-file", "", "Path to binary blockchain file to import.")
		node       = flag.String("node", "http://127.0.0.1:6869", "URL of node API. Default value http://127.0.0.1:6869.")
		address    = flag.String("address", ":6990", "Local network address to bind HTTP API of the service.")
		db         = flag.String("db", "", "Path to data base.")
	)
	flag.Parse()

	logger, log := setupLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("Failed to close logging subsystem: %s", err.Error())
		}
	}()

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
	storage := internal.Storage{Path: *db}
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

	h, err := storage.GetLastHeight()
	if err != nil {
		log.Warnf("Failed to get current height: %s", err.Error())
	}
	log.Infof("Last stored height: %d", h)

	err = importBlockchainIfNeeded(appCtx, log, *importFile, &storage)
	if err != nil {
		log.Errorf("Initial blockchain import failed: %s", err.Error())
	} else {
		os.Exit(0)
	}

	df := internal.DataFeedAPI{Storage: &storage}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/datafeed", df.Routes())
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

func importBlockchainIfNeeded(ctx context.Context, log *zap.SugaredLogger, n string, storage *internal.Storage) error {
	if n != "" {
		if _, err := os.Stat(n); os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to import blockchain file '%s'", n)
		}
		i := internal.NewImporter(ctx, log, storage)
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
