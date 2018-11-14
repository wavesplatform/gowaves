package main

import (
	"context"
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		node     = flag.String("node", "http://127.0.0.1:6869", "URL of node API. Default value http://127.0.0.1:6869.")
		logLevel = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	)
	flag.Parse()

	logger, log := setupLogger(*logLevel)
	defer logger.Sync()

	_, err := url.Parse(*node)
	if err != nil {
		log.Errorf("Failed to parse node API address: %s", err.Error())
		shutdown()
	}

	appCtx, cancel := context.WithCancel(context.Background())

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
