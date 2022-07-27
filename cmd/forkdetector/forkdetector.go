package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/forkdetector/internal"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	version = "v0.0.0"
)

type configuration struct {
	logLevel       string
	logFile        string
	dbPath         string
	scheme         byte
	genesis        proto.BlockID
	apiBind        string
	netBind        string
	publicAddress  net.TCPAddr
	name           string
	nonce          uint64
	versions       []proto.Version
	seedPeers      []net.TCPAddr
	cpuProfileFile *os.File
	memProfileFile *os.File
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	cfg, err := parseConfiguration()
	if err != nil {
		fmt.Println("Invalid parameters:", err)
		flag.Usage()
		return err
	}
	setupLogger(cfg.logLevel, cfg.logFile)

	if cfg.cpuProfileFile != nil {
		defer func() {
			err := cfg.cpuProfileFile.Close()
			zap.S().Errorf("Failed to close CPU profile: %v", err)
		}()
		err := pprof.StartCPUProfile(cfg.cpuProfileFile)
		if err != nil {
			zap.S().Errorf("Could not start CPU profile: %v", err)
			return err
		}
		defer pprof.StopCPUProfile()
	}

	// Get a channel that will be closed on shutdown signals (CTRL-C) or shutdown request
	interrupt := interruptListener()
	defer zap.S().Info("Shutdown complete")

	zap.S().Infof("Waves Fork Detector %s", version)

	storage, err := internal.NewStorage(cfg.dbPath, cfg.genesis, cfg.scheme)
	if err != nil {
		zap.S().Errorf("Failed to open Storage: %v", err)
		return err
	}

	if interruptRequested(interrupt) {
		return nil
	}

	reg := internal.NewRegistry(cfg.scheme, &cfg.publicAddress, cfg.versions, storage)
	n := reg.AppendAddresses(cfg.seedPeers)
	if n > 0 {
		zap.S().Infof("%d seed peers added to storage", n)
	}

	drawer, err := internal.NewDrawer(storage)
	if err != nil {
		zap.S().Errorf("Failed to create restoer state: %v", err)
		return err
	}

	api, err := internal.NewAPI(interrupt, storage, reg, drawer, cfg.apiBind)
	if err != nil {
		zap.S().Errorf("Failed to create API server: %v", err)
		return err
	}
	apiDone := api.Start()

	if interruptRequested(interrupt) {
		return nil
	}

	distributor, err := internal.NewDistributor(interrupt, drawer)
	if err != nil {
		zap.S().Errorf("Failed to instantiate distributor: %v", err)
		return err
	}
	distributorDone := distributor.Start()

	h := internal.NewConnHandler(cfg.scheme, cfg.name, cfg.nonce, cfg.publicAddress, reg, distributor.NewConnectionsSink(), distributor.ClosedConnectionsSink(), distributor.ScoreSink(), distributor.IdsSink(), distributor.BlocksSink())
	opts := internal.NewOptions(h)
	opts.ReadDeadline = time.Minute
	opts.WriteDeadline = time.Minute

	dispatcher := internal.NewDispatcher(distributorDone, cfg.netBind, opts, reg)
	dispatcherDone := dispatcher.Start()

	<-interrupt

	<-apiDone
	zap.S().Debug("API shutdown complete")
	<-dispatcherDone
	zap.S().Debug("Dispatcher shutdown complete")
	<-distributorDone
	zap.S().Debugf("Distributor shutdown complete")

	err = storage.Close()
	if err != nil {
		zap.S().Errorf("Failed to close the storage: %v", err)
		return err
	}

	if cfg.memProfileFile != nil {
		defer func() {
			err := cfg.memProfileFile.Close()
			if err != nil {
				zap.S().Errorf("Failed to close memory profile: %v", err)
			}
		}()
		runtime.GC()
		err := pprof.WriteHeapProfile(cfg.memProfileFile)
		if err != nil {
			zap.S().Errorf("Failed to write memory profile: %v", err)
			return err
		}
	}

	return nil
}

func parseConfiguration() (*configuration, error) {
	var (
		logLevel        = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level is INFO.")
		logFile         = flag.String("log-file", "", "Path to log file. Log files are rotated at size 100MB. By default there is no log file.")
		db              = flag.String("db", "", "Path to database folder. No default value.")
		scheme          = flag.String("scheme", "W", "Blockchain scheme symbol. Defaults to \"W\" - MainNet scheme.")
		genesis         = flag.String("genesis", "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2", "Genesis block ID in BASE58 encoding. Default value is MainNet's genesis block id.")
		versions        = flag.String("versions", "1.0 0.17 0.16 0.15 0.14 0.13 0.10 0.9 0.8 0.7 0.6 0.3", "Space separated list of known node's versions. By default all MainNet versions are supported.")
		apiBindAddress  = flag.String("api-bind", ":8080", "Local network address to bind the HTTP API of the service on. Default value is \":8080\".")
		netBindAddress  = flag.String("net-bind", ":6868", "Local network address to bind the network server. Default value is \":6868 \".")
		netName         = flag.String("net-name", "Fork Detector", "Name of the node to identify on the network. Default value is \"Fork Detector\".")
		netNonce        = flag.Int("net-nonce", 0, "Nonce part of the node's identity on the network. Default value is 0, which means the nonce will be randomly generated.")
		declaredAddress = flag.String("declared-address", "", "The network address of the node publicly accessible for incoming connections. Empty default value (no publicly visible address).")
		seedPeers       = flag.String("seed-peers",
			"13.228.86.201:6868 13.229.0.149:6868 18.195.170.147:6868 34.253.153.4:6868 35.156.19.4:6868 52.50.69.247:6868 52.52.46.76:6868 52.57.147.71:6868 52.214.55.18:6868 54.176.190.226:6868",
			"Space separated list of public peers for initial connection. Defaults to MainNet's public peers.")
		cpuProfilePath = flag.String("cpu-profile", "", "Write CPU profile to the file.")
		memProfilePath = flag.String("mem-profile", "", "Write memory profile to the file.")
	)
	flag.Parse()
	if *db == "" {
		return nil, errors.New("no database path")
	}
	if len(*scheme) != 1 {
		return nil, errors.Errorf("invalid scheme '%s'", *scheme)
	}
	id, err := proto.NewBlockIDFromBase58(*genesis)
	if err != nil {
		return nil, errors.Wrap(err, "invalid genesis block id")
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
		nonce, err = generateNonce()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate nonce")
		}
	}
	addr := proto.ParseHandshakeTCPAddr(*declaredAddress)
	peers, err := splitPeers(*seedPeers)
	if err != nil {
		return nil, errors.Wrap(err, "invalid seed peers list")
	}
	var cpuProf *os.File
	if *cpuProfilePath != "" {
		cpuProf, err = os.Create(*cpuProfilePath)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create CPU profile")
		}
	}
	var memProf *os.File
	if *memProfilePath != "" {
		memProf, err = os.Create(*memProfilePath)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create memory profile")
		}
	}

	cfg := &configuration{
		dbPath:         *db,
		logLevel:       *logLevel,
		logFile:        *logFile,
		scheme:         (*scheme)[0],
		genesis:        id,
		versions:       vs,
		seedPeers:      peers,
		name:           *netName,
		nonce:          nonce,
		apiBind:        *apiBindAddress,
		netBind:        *netBindAddress,
		publicAddress:  net.TCPAddr(addr),
		cpuProfileFile: cpuProf,
		memProfileFile: memProf,
	}
	return cfg, nil
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
		r = append(r, v)
	}
	return r, nil
}

func setupLogger(level, file string) (*zap.Logger, *zap.SugaredLogger) {
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
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.Lock(os.Stdout), al)
	if file != "" {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename: file,
		})
		core = zapcore.NewTee(core, zapcore.NewCore(zapcore.NewConsoleEncoder(ec), w, al))
	}
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	return logger, logger.Sugar()
}

func generateNonce() (uint64, error) {
	nonce, err := rand.Int(rand.Reader, new(big.Int).SetUint64(math.MaxUint64))
	if err != nil {
		return 0, err
	}
	return nonce.Uint64(), nil
}
