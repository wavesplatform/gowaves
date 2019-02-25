package main

import (
	"flag"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/forkdetector/internal"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net"
	"os"
	"strings"
)

var (
	version = "v0.0.0"
)

type configuration struct {
	logLevel     string
	dbPath       string
	scheme       byte
	genesis      crypto.Signature
	apiBind      string
	netBind      string
	announcement string
	nodeName     string
	versions     []proto.Version
	seedPeers    []string
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
	os.Exit(1)
}

func run() error {
	cfg, err := parseConfiguration()
	if err != nil {
		flag.Usage()
		return err
	}
	logger, log := setupLogger(cfg.logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil && err == os.ErrInvalid {
			log.Fatalf("Failed to close logging subsystem: %s", err.Error())
		}
	}()

	// Get a channel that will be closed on shutdown signals (CTRL-C) or shutdown request
	interrupt := interruptListener(log)
	defer log.Info("Shutdown complete")

	log.Infof("Waves Fork Detector %s", version)

	storage, err := internal.NewStorage(cfg.dbPath, log, cfg.genesis)
	if err != nil {
		log.Errorf("Failed to open Storage: %v", err)
		return err
	}

	apiDone := internal.StartForkDetectorAPI(interrupt, logger, cfg.apiBind)
	if interruptRequested(interrupt) {
		return nil
	}

	dispatcher := internal.NewDispatcher(interrupt, log, storage, cfg.announcement, cfg.nodeName, cfg.scheme)
	dispatcherDone, err := dispatcher.Start(cfg.netBind, cfg.seedPeers)
	if err != nil {
		log.Errorf("Failed to start peers dispatcher: %v", err)
		return err
	}

	<-apiDone
	log.Info("API shutdown complete")
	<-dispatcherDone
	log.Info("Peers dispatcher shutdown complete")

	<-interrupt
	return nil
}

func parseConfiguration() (*configuration, error) {
	var (
		logLevel        = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level is INFO.")
		db              = flag.String("db", "", "Path to database folder. No default value.")
		scheme          = flag.String("scheme", "W", "Blockchain scheme symbol. Defaults to \"W\" - MainNet scheme.")
		genesis         = flag.String("genesis", "5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa", "Genesis block signature in BASE58 encoding. Default value is MainNet's genesis block signature.")
		versions        = flag.String("versions", "0.16 0.15 0.14 0.13 0.10 0.9 0.8 0.7 0.6 0.3", "Space separated list of known node's versions. By default all MainNet versions are supported.")
		apiBindAddress  = flag.String("api-bind", ":8080", "Local network address to bind the HTTP API of the service on. Default value is \":8080\".")
		netBindAddress  = flag.String("net-bind", ":6868", "Local network address to bind the network server. Default value is \":6868 \".")
		netName         = flag.String("net-name", "Fork Detector", "Name of the node to identify on the network. Default value is \"Fork Detector\".")
		declaredAddress = flag.String("declared-address", "", "The network address of the node publicly accessible for incoming connections. Empty default value (no publicly visible address).")
		seedPeers       = flag.String("seed-peers",
			"13.228.86.201:6868 13.229.0.149:6868 18.195.170.147:6868 34.253.153.4:6868 35.156.19.4:6868 52.50.69.247:6868 52.52.46.76:6868 52.57.147.71:6868 52.214.55.18:6868 54.176.190.226:6868",
			"Space separated list of public peers for initial connection. Defaults to MainNet's public peers.")
	)
	flag.Parse()
	if *db == "" {
		return nil, errors.New("no database path")
	}
	if len(*scheme) != 1 {
		return nil, errors.Errorf("invalid scheme '%s'", *scheme)
	}
	sig, err := crypto.NewSignatureFromBase58(*genesis)
	if err != nil {
		return nil, errors.Wrap(err, "invalid genesis block signature")
	}
	vs, err := splitVersions(*versions)
	if err != nil {
		return nil, errors.Wrap(err, "invalid versions")
	}
	if *netBindAddress == "" {
		return nil, errors.Errorf("invalid bind address for network server '%s'", *netBindAddress)
	}
	if l := len(*netName); l <= 0 || l > 255 {
		return nil, errors.Errorf("invalid network name '%s'", *netName)
	}
	addr, err := validateNetworkAddress(*declaredAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid declared address")
	}
	peers, err := splitPeers(*seedPeers)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid seed peers list")
	}
	cfg := &configuration{
		dbPath:       *db,
		logLevel:     *logLevel,
		scheme:       (byte)((*scheme)[0]),
		genesis:      sig,
		versions:     vs,
		seedPeers:    peers,
		nodeName:     *netName,
		apiBind:      *apiBindAddress,
		netBind:      *netBindAddress,
		announcement: addr,
	}
	return cfg, nil
}

func validateNetworkAddress(s string) (string, error) {
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return "", errors.Wrap(err, "invalid network address")
	}
	if h == "" {
		return "", errors.New("no host")
	}
	return net.JoinHostPort(h, p), nil
}

func splitPeers(s string) ([]string, error) {
	r := strings.Fields(s)
	for _, a := range r {
		_, err := validateNetworkAddress(a)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid address '%s'", a)
		}
	}
	return r, nil
}

func splitVersions(s string) ([]proto.Version, error) {
	fields := strings.Fields(s)
	r := make([]proto.Version, 0, len(fields))
	for _, f := range fields {
		v, err := proto.NewVersionFromString(f)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid version '%s'", f)
		}
		r = append(r, *v)
	}
	return r, nil
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
