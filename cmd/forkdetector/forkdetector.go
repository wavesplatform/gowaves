package main

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/forkdetector/internal"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
	name         string
	nonce        uint64
	versions     []proto.Version
	seedPeers    []net.TCPAddr
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
		fmt.Println("Invalid parameters:", err)
		flag.Usage()
		return err
	}
	setupLogger(cfg.logLevel)

	// Get a channel that will be closed on shutdown signals (CTRL-C) or shutdown request
	interrupt := interruptListener()
	defer zap.S().Info("Shutdown complete")

	zap.S().Infof("Waves Fork Detector %s", version)

	storage, err := internal.NewStorage(cfg.dbPath, cfg.genesis)
	if err != nil {
		zap.S().Errorf("Failed to open Storage: %v", err)
		return err
	}

	if interruptRequested(interrupt) {
		return nil
	}

	registry := internal.NewPublicAddressRegistry(storage, 10*time.Minute, 24*time.Hour, cfg.versions)
	na, err := registry.RegisterNewAddresses(cfg.seedPeers)
	if err != nil {
		zap.S().Errorf("Failed to initialize seed peers: %v", err)
		return err
	}
	zap.S().Infof("%d new seed addresses were registered", na)

	api, err := internal.NewAPI(interrupt, storage, cfg.apiBind)
	if err != nil {
		zap.S().Errorf("Failed to create API server: %v", err)
		return err
	}
	apiDone := api.Start()

	if interruptRequested(interrupt) {
		return nil
	}

	server, err := internal.NewServer(interrupt, cfg.netBind)
	if err != nil {
		zap.S().Errorf("Failed to start network server: %v", err)
		return err
	}

	dispatcher, err := internal.NewDispatcher(interrupt, registry, server.GetConnections(), cfg.announcement, cfg.name, cfg.nonce, cfg.scheme, cfg.versions)
	if err != nil {
		zap.S().Errorf("Failed to initialize dispatcher: %v", err)
		return err
	}
	dispatcherDone := dispatcher.Start()

	if interruptRequested(interrupt) {
		return nil
	}

	serverDone := server.Start()

	if interruptRequested(interrupt) {
		return nil
	}

	<-interrupt

	<-apiDone
	zap.S().Debug("API shutdown complete")
	<-dispatcherDone
	zap.S().Debug("Dispatcher shutdown complete")
	<-serverDone
	zap.S().Debug("Network server shutdown complete")

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
		netNonce        = flag.Int("net-nonce", 0, "Nonce part of the node's identity on the network. Default value is 0, which means the nonce will be randomly generated.")
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
	nonce := uint64(*netNonce)
	if nonce == 0 {
		nonce = generateNonce()
	}
	var addr string
	if *declaredAddress == "" {
		addr = ""
	} else {
		addr, err = validateNetworkAddress(*declaredAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid declared address")
		}
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
		name:         *netName,
		nonce:        nonce,
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

func splitPeers(s string) ([]net.TCPAddr, error) {
	sp := strings.Fields(s)
	r := make([]net.TCPAddr, 0)
	for _, ma := range sp {
		h, p, err := net.SplitHostPort(ma)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid address '%s'", ma)
		}
		pn, err := strconv.Atoi(p)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid address '%s'", ma)
		}
		a := net.TCPAddr{IP: net.ParseIP(h), Port: pn}
		r = append(r, a)
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
	zap.ReplaceGlobals(logger)
	return logger, logger.Sugar()
}

func generateNonce() uint64 {
	return uint64(rand.Int63n(1000000))
}
