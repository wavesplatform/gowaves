package main

import (
	"context"
	"crypto/rand"
	stderrs "errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/network"
	"github.com/wavesplatform/gowaves/pkg/node/peers"
	peersPersistentStorage "github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/versioning"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

const (
	mb                     = 1 << 20
	defaultTimeout         = 30 * time.Second
	shutdownTimeout        = 5 * time.Second
	fileDescriptorsReserve = 10
)

const profilerAddr = "localhost:6060"

const utxPoolMaxSizeBytes = 1024 * mb

var defaultPeers = map[string]string{
	"mainnet":  "34.253.153.4:6868,168.119.116.189:6868,135.181.87.72:6868,162.55.39.115:6868,168.119.155.201:6868",
	"testnet":  "159.69.126.149:6868,94.130.105.239:6868,159.69.126.153:6868,94.130.172.201:6868,35.157.247.122:6868",
	"stagenet": "88.99.185.128:6868,49.12.15.166:6868,95.216.205.3:6868,88.198.179.16:6868,52.58.254.101:6868",
}

type config struct {
	isParsed bool

	logLevel                   zapcore.Level
	logDevelopment             bool
	logNetwork                 bool
	logNetworkData             bool
	logFSM                     bool
	statePath                  string
	blockchainType             string
	peerAddresses              string
	declAddr                   string
	nodeName                   string
	cfgPath                    string
	apiAddr                    string
	apiKey                     string
	apiMaxConnections          int
	rateLimiterOptions         string
	grpcAddr                   string
	grpcAPIMaxConnections      int
	enableMetaMaskAPI          bool
	enableMetaMaskAPILog       bool
	enableGrpcAPI              bool
	blackListResidenceTime     time.Duration
	buildExtendedAPI           bool
	serveExtendedAPI           bool
	buildStateHashes           bool
	bindAddress                string
	disableOutgoingConnections bool
	minerVoteFeatures          string
	disableBloomFilter         bool
	reward                     int64
	obsolescencePeriod         time.Duration
	walletPath                 string
	walletPassword             string
	limitAllConnections        uint
	minPeersMining             int
	disableMiner               bool
	profiler                   bool
	prometheus                 string
	metricsID                  int
	metricsURL                 string
	dropPeers                  bool
	dbFileDescriptors          uint
	newConnectionsLimit        int
	disableNTP                 bool
	microblockInterval         time.Duration
	enableLightMode            bool
	generateInPast             bool
}

var errConfigNotParsed = stderrs.New("config is not parsed")

func (c *config) StatePath() (string, error) {
	if !c.isParsed {
		return "", errConfigNotParsed
	}
	if path := c.statePath; path != "" {
		return path, nil
	}
	path, err := common.GetStatePath()
	if err != nil {
		return "", errors.Wrap(err, "failed to get common state path")
	}
	return path, nil
}

func (c *config) logParameters() {
	zap.S().Debugf("log-level: %s", c.logLevel)
	zap.S().Debugf("log-dev: %t", c.logDevelopment)
	zap.S().Debugf("log-network: %t", c.logNetwork)
	zap.S().Debugf("log-fsm: %t", c.logFSM)
	zap.S().Debugf("state-path: %s", c.statePath)
	zap.S().Debugf("blockchain-type: %s", c.blockchainType)
	zap.S().Debugf("peers: %s", c.peerAddresses)
	zap.S().Debugf("declared-address: %s", c.declAddr)
	zap.S().Debugf("api-address: %s", c.apiAddr)
	zap.S().Debugf("api-key: %s", crypto.MustKeccak256([]byte(c.apiKey)).Hex())
	zap.S().Debugf("grpc-address: %s", c.grpcAddr)
	zap.S().Debugf("enable-grpc-api: %t", c.enableGrpcAPI)
	zap.S().Debugf("black-list-residence-time: %s", c.blackListResidenceTime)
	zap.S().Debugf("build-extended-api: %t", c.buildExtendedAPI)
	zap.S().Debugf("serve-extended-api: %t", c.serveExtendedAPI)
	zap.S().Debugf("build-state-hashes: %t", c.buildStateHashes)
	zap.S().Debugf("bind-address: %s", c.bindAddress)
	zap.S().Debugf("vote: %s", c.minerVoteFeatures)
	zap.S().Debugf("reward: %d", c.reward)
	zap.S().Debugf("obsolescence: %s", c.obsolescencePeriod)
	zap.S().Debugf("disable-miner %t", c.disableMiner)
	zap.S().Debugf("wallet-path: %s", c.walletPath)
	zap.S().Debugf("hashed wallet-password: %s", crypto.MustKeccak256([]byte(c.walletPassword)).Hex())
	zap.S().Debugf("limit-connections: %d", c.limitAllConnections)
	zap.S().Debugf("profiler: %t", c.profiler)
	zap.S().Debugf("disable-bloom: %t", c.disableBloomFilter)
	zap.S().Debugf("drop-peers: %t", c.dropPeers)
	zap.S().Debugf("db-file-descriptors: %v", c.dbFileDescriptors)
	zap.S().Debugf("new-connections-limit: %v", c.newConnectionsLimit)
	zap.S().Debugf("enable-metamask: %t", c.enableMetaMaskAPI)
	zap.S().Debugf("disable-ntp: %t", c.disableNTP)
	zap.S().Debugf("microblock-interval: %s", c.microblockInterval)
	zap.S().Debugf("enable-light-mode: %t", c.enableLightMode)
	zap.S().Debugf("generate-in-past: %t", c.generateInPast)
}

func (c *config) parse() {
	if c.isParsed { // no need to parse twice
		return
	}
	defer func() { c.isParsed = true }()
	const (
		defaultBlacklistResidenceDuration = 5 * time.Minute
		defaultObsolescenceDuration       = 4 * time.Hour
		defaultConnectionsLimit           = 60
		defaultNewConnectionLimit         = 10
		defaultMicroblockInterval         = 5 * time.Second
	)
	l := zap.LevelFlag("log-level", zapcore.InfoLevel,
		"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL.")
	flag.BoolVar(&c.logDevelopment, "log-dev", false,
		"Log with development setup for the logger. Switched off by default.")
	flag.BoolVar(&c.logNetwork, "log-network", false,
		"Log the operation of network stack. Turned off by default.")
	flag.BoolVar(&c.logNetworkData, "log-network-data", false,
		"Log network messages as Base64 strings. Turned off by default.")
	flag.BoolVar(&c.logFSM, "log-fsm", false,
		"Log the operation of FSM. Turned off by default.")
	flag.StringVar(&c.statePath, "state-path", "", "Path to node's state directory.")
	flag.StringVar(&c.blockchainType, "blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet.")
	flag.StringVar(&c.peerAddresses, "peers", "",
		"Forces the node to connect to the provided peers. Format: \"ip:port,...,ip:port\".")
	flag.StringVar(&c.declAddr, "declared-address", "", "Address to listen on.")
	flag.StringVar(&c.nodeName, "name", "gowaves", "Node name.")
	flag.StringVar(&c.cfgPath, "cfg-path", "",
		"Path to configuration JSON file, only for custom blockchain.")
	flag.StringVar(&c.apiAddr, "api-address", "", "Address for REST API.")
	flag.StringVar(&c.apiKey, "api-key", "", "Api key.")
	flag.IntVar(&c.apiMaxConnections, "api-max-connections", api.DefaultMaxConnections,
		"Max number of simultaneous connections for REST API.")
	flag.StringVar(&c.rateLimiterOptions, "rate-limiter-opts", "",
		"Rate limiter options in form of URL query options, e.g. \"cache=1024&rps=10&burst=5\", keys 'cache' - "+
			"rate limiter cache size in bytes, 'rps' - requests per second, 'burst' - available burst")
	flag.StringVar(&c.grpcAddr, "grpc-address", "127.0.0.1:7475", "Address for gRPC API.")
	flag.IntVar(&c.grpcAPIMaxConnections, "grpc-api-max-connections", server.DefaultMaxConnections,
		"Max number of simultaneous connections for gRPC API.")
	flag.BoolVar(&c.enableMetaMaskAPI, "enable-metamask", true, "Enables/disables metamask API.")
	flag.BoolVar(&c.enableMetaMaskAPILog, "enable-metamask-log", false,
		"Enables/disables metamask API logging.")
	flag.BoolVar(&c.enableGrpcAPI, "enable-grpc-api", false, "Enables/disables gRPC API.")
	flag.DurationVar(&c.blackListResidenceTime, "blacklist-residence-time", defaultBlacklistResidenceDuration,
		"Period of time for which the information about external peer stays in the blacklist. "+
			"Default value is 5 min. To disable blacklisting pass zero value.")
	flag.BoolVar(&c.buildExtendedAPI, "build-extended-api", false,
		"Builds extended API. "+
			"Note that state must be re-imported in case it wasn't imported with similar flag set.")
	flag.BoolVar(&c.serveExtendedAPI, "serve-extended-api", false,
		"Serves extended API requests since the very beginning. "+
			"The default behavior is to import until first block close to current time, "+
			"and start serving at this point.")
	flag.BoolVar(&c.buildStateHashes, "build-state-hashes", false,
		"Calculate and store state hashes for each block height.")
	flag.StringVar(&c.bindAddress, "bind-address", "",
		"Bind address for incoming connections. If empty, will be same as declared address")
	flag.BoolVar(&c.disableOutgoingConnections, "no-connections", false,
		"Disable outgoing network connections to known peers."+
			"This flag DOES NOT disable outgoing connections to peers from the 'peers' option.")
	flag.StringVar(&c.minerVoteFeatures, "vote", "", "Miner vote features.")
	flag.BoolVar(&c.disableBloomFilter, "disable-bloom", false,
		"Disable bloom filter. Less memory usage, but decrease performance.")
	flag.Int64Var(&c.reward, "reward", 0, "Miner reward: for example 600000000.")
	flag.DurationVar(&c.obsolescencePeriod, "obsolescence", defaultObsolescenceDuration,
		"Blockchain obsolescence period. Disable mining if last block older then given value.")
	flag.StringVar(&c.walletPath, "wallet-path", "", "Path to wallet, or ~/.waves by default.")
	flag.StringVar(&c.walletPassword, "wallet-password", "", "Pass password for wallet.")
	flag.UintVar(&c.limitAllConnections, "limit-connections", defaultConnectionsLimit,
		"Total limit of network connections, both inbound and outbound. Divided in half to limit each direction.")
	flag.IntVar(&c.minPeersMining, "min-peers-mining", 1,
		"Minimum connected peers for allow mining.")
	flag.BoolVar(&c.disableMiner, "disable-miner", false, "Disable miner.")
	flag.BoolVar(&c.profiler, "profiler", false,
		fmt.Sprintf("Start built-in profiler on 'http://%s/debug/pprof/'.", profilerAddr))
	flag.StringVar(&c.prometheus, "prometheus", "",
		"Provide collected metrics by prometheus client.")
	flag.IntVar(&c.metricsID, "metrics-id", -1,
		"ID of the node on the metrics collection system.")
	flag.StringVar(&c.metricsURL, "metrics-url", "",
		"URL of InfluxDB or Telegraf in form of 'http://username:password@host:port/db'.")
	flag.BoolVar(&c.dropPeers, "drop-peers", false,
		"Drop peers storage before node start.")
	flag.UintVar(&c.dbFileDescriptors, "db-file-descriptors", uint(state.DefaultOpenFilesCacheCapacity), // #nosec:G115
		"Maximum allowed file descriptors count that will be used by state database.")
	flag.IntVar(&c.newConnectionsLimit, "new-connections-limit", defaultNewConnectionLimit,
		"Number of new outbound connections established simultaneously, defaults to 10. Should be positive. "+
			"Big numbers can badly affect file descriptors consumption.")
	flag.BoolVar(&c.disableNTP, "disable-ntp", false,
		"Disable NTP synchronization. Useful when running the node in a docker container.")
	flag.DurationVar(&c.microblockInterval, "microblock-interval", defaultMicroblockInterval,
		"Interval between microblocks.")
	flag.BoolVar(&c.enableLightMode, "enable-light-mode", false,
		"Start node in light mode")
	flag.BoolVar(&c.generateInPast, "generate-in-past", false,
		"Enable block generation with timestamp in the past")
	flag.Parse()
	c.logLevel = *l
}

func loggerSetup(nc *config) func() {
	logger := logging.SetupLogger(nc.logLevel,
		logging.DevelopmentFlag(nc.logDevelopment),
		logging.NetworkFilter(nc.logNetwork),
		logging.NetworkDataFilter(nc.logNetworkData),
		logging.FSMFilter(nc.logFSM),
	)
	return func() {
		if err := logger.Sync(); err != nil && stderrs.Is(err, os.ErrInvalid) {
			panic(fmt.Sprintf("Failed to close logging subsystem: %v\n", err))
		}
	}
}

type Scheduler interface {
	Mine() chan scheduler.Emit
	types.Scheduler
	Emits() []scheduler.Emit
}

func main() {
	os.Exit(realMain()) // for more info see https://github.com/golang/go/issues/42078
}

func realMain() int {
	nc := new(config)
	nc.parse()
	syncFn := loggerSetup(nc)
	defer syncFn()
	err := run(nc)
	if err != nil {
		zap.S().Errorf("Failed to run: %v", err)
		return 1
	}
	return 0
}

func run(nc *config) (retErr error) {
	errg, ctx := errgroup.WithContext(context.Background())
	defer func() {
		if wErr := errg.Wait(); !errors.Is(wErr, context.Canceled) {
			retErr = stderrs.Join(retErr, wErr)
		}
	}()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if nc.profiler {
		errg.Go(func() error {
			<-runProfiler(ctx)
			return nil
		})
	}
	if nc.prometheus != "" {
		errg.Go(func() error {
			<-runPrometheusMetricsServer(ctx, nc.prometheus)
			return nil
		})
	}

	zap.S().Infof("Gowaves Node version: %s", versioning.Version)

	nc.logParameters() // print all parsed parameters

	if err := raiseToMaxFDs(nc); err != nil { // raiseToMaxFDs raises the limit of file descriptors
		return errors.Wrap(err, "failed to raise file descriptors limit")
	}

	if nc.metricsURL != "" && nc.metricsID != -1 {
		err := metrics.Start(ctx, nc.metricsID, nc.metricsURL)
		if err != nil {
			zap.S().Warnf("Metrics reporting failed to start: %v", err)
			zap.S().Warn("Proceeding without reporting any metrics")
		} else {
			zap.S().Info("Metrics reporting activated")
		}
	}

	nodeCloser, err := runNode(ctx, nc)
	if err != nil {
		return errors.Wrap(err, "failed to run node")
	}

	<-ctx.Done()
	zap.S().Info("User termination in progress...")
	defer func() { <-time.After(1 * time.Second) }() // give some time to close internal node processes
	if clErr := nodeCloser.Close(); clErr != nil {
		return errors.Wrap(clErr, "failed to close node")
	}
	return nil
}

func runNode(ctx context.Context, nc *config) (_ io.Closer, retErr error) {
	cfg, err := blockchainSettings(nc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blockchain settings")
	}

	conf, err := nodeSettings(nc, cfg.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node settings")
	}

	wal, err := embeddedWallet(nc, cfg.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get embedded wallet")
	}

	path, err := nc.StatePath()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state path")
	}

	ntpTime, err := getNtp(ctx, nc.disableNTP)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get NTP time")
	}

	params, err := stateParams(nc, ntpTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create state parameters")
	}

	st, err := state.NewState(path, true, params, cfg, nc.enableLightMode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize node's state")
	}
	defer func() { retErr = closeIfErrorf(st, retErr, "failed to close state") }()

	features, err := minerFeatures(st, nc.minerVoteFeatures)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse and validate miner features")
	}

	// Check if we need to start serving extended API right now.
	if eapiErr := node.MaybeEnableExtendedApi(st, ntpTime); eapiErr != nil {
		return nil, errors.Wrap(eapiErr, "failed to enable extended API")
	}

	parent := peer.NewParent(nc.enableLightMode)
	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	peerManager, err := createPeerManager(nc, conf, parent, declAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create peer manager")
	}
	defer func() { retErr = closeIfErrorf(peerManager, retErr, "failed to close peer manager") }()
	go peerManager.Run(ctx)

	minerScheduler, err := newMinerScheduler(nc, st, wal, cfg, ntpTime, peerManager)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize miner scheduler")
	}

	svs, err := createServices(nc, st, wal, cfg, ntpTime, peerManager, parent, minerScheduler)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create services")
	}

	app, err := api.NewApp(nc.apiKey, minerScheduler, svs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize application")
	}

	if pErr := spawnPeersByAddresses(ctx, conf.Addresses, peerManager); pErr != nil {
		return nil, errors.Wrap(pErr, "failed to spawn peers by addresses")
	}

	if apiErr := runAPIs(ctx, nc, conf, app, svs); apiErr != nil {
		return nil, errors.Wrap(apiErr, "failed to run APIs")
	}

	return startNode(ctx, nc, svs, features, minerScheduler, parent, declAddr), nil
}

func startNode(
	ctx context.Context,
	nc *config,
	svs services.Services,
	features miner.Features,
	minerScheduler Scheduler,
	parent peer.Parent,
	declAddr proto.TCPAddr,
) *node.Node {
	bindAddr := proto.NewTCPAddrFromString(nc.bindAddress)

	mine := miner.NewMicroblockMiner(svs, features, nc.reward)
	go miner.Run(ctx, mine, minerScheduler, svs.InternalChannel)

	ntw, networkInfoCh := network.NewNetwork(svs, parent, nc.obsolescencePeriod)
	go ntw.Run(ctx)

	n := node.NewNode(svs, declAddr, bindAddr, nc.microblockInterval, nc.enableLightMode)
	go n.Run(ctx, parent, svs.InternalChannel, networkInfoCh, ntw.SyncPeer())

	go minerScheduler.Reschedule() // Reschedule mining after node start

	return n
}

func raiseToMaxFDs(nc *config) error {
	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		return errors.Wrap(err, "failed to get max FDs")
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		return errors.Wrap(err, "failed to raise max FDs")
	}
	if m := maxFDs - uint64(nc.limitAllConnections) - fileDescriptorsReserve; uint64(nc.dbFileDescriptors) > m {
		return errors.Errorf("invalid 'db-file-descriptors' flag value (%d), value shall be less or equal to %d",
			nc.dbFileDescriptors, m,
		)
	}
	return nil
}

func blockchainSettings(nc *config) (_ *settings.BlockchainSettings, retErr error) {
	if nc.cfgPath == "" {
		cfg, err := settings.BlockchainSettingsByTypeName(nc.blockchainType)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get blockchain settings")
		}
		return cfg, nil
	}
	f, err := os.Open(nc.cfgPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open configuration file")
	}
	defer func() {
		if clErr := f.Close(); clErr != nil {
			retErr = stderrs.Join(retErr, errors.Wrap(clErr, "failed to close configuration file"))
		}
	}()
	cfg, err := settings.ReadBlockchainSettings(io.LimitReader(f, mb))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configuration file")
	}
	return cfg, nil
}

func stateParams(nc *config, ntpTime types.Time) (state.StateParams, error) {
	dbFileDescriptors := nc.dbFileDescriptors
	if dbFileDescriptors > math.MaxInt {
		return state.StateParams{}, errors.Errorf("too big 'db-file-descriptors' flag value (%d)",
			nc.dbFileDescriptors,
		)
	}
	params := state.DefaultStateParams()
	params.DbParams.OpenFilesCacheCapacity = int(dbFileDescriptors)
	params.StoreExtendedApiData = nc.buildExtendedAPI
	params.ProvideExtendedApi = nc.serveExtendedAPI
	params.BuildStateHashes = nc.buildStateHashes
	params.Time = ntpTime
	params.DbParams.DisableBloomFilter = nc.disableBloomFilter
	return params, nil
}

func runProfiler(ctx context.Context) <-chan struct{} {
	pprofMux := http.NewServeMux()
	// taken from "net/http/pprof" init()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s := &http.Server{
		Addr:              profilerAddr,
		Handler:           pprofMux,
		ReadHeaderTimeout: defaultTimeout,
		ReadTimeout:       defaultTimeout,
	}
	s.RegisterOnShutdown(func() {
		zap.S().Info("Profiler is shutting down...")
	})
	go func() {
		zap.S().Infof("Starting built-in profiler on 'http://%s/debug/pprof/'", profilerAddr)
		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.S().Errorf("Failed to start profiler: %v", err)
		}
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			zap.S().Errorf("Failed to shutdown profiler: %v", err)
		}
	}()
	return done
}

func runPrometheusMetricsServer(ctx context.Context, prometheusAddr string) <-chan struct{} {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	s := &http.Server{
		Addr:              prometheusAddr,
		Handler:           h,
		ReadHeaderTimeout: defaultTimeout,
		ReadTimeout:       defaultTimeout,
	}
	s.RegisterOnShutdown(func() {
		zap.S().Info("Prometheus metrics server is shutting down...")
	})
	go func() {
		zap.S().Infof("Starting prometheus metrics server on '%s'", prometheusAddr)
		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.S().Errorf("Failed to start prometheus metrics server: %v", err)
		}
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			zap.S().Errorf("Failed to shutdown prometheus: %v", err)
		}
	}()
	return done
}

func runGRPCServer(ctx context.Context, addr string, nc *config, svs services.Services) error {
	srv, srvErr := server.NewServer(svs)
	if srvErr != nil {
		return errors.Wrap(srvErr, "failed to create gRPC server")
	}
	go func() {
		if runErr := srv.Run(ctx, addr, grpcAPIRunOptsFromCLIFlags(nc)); runErr != nil {
			zap.S().Errorf("grpcServer.Run(): %v", runErr)
		}
	}()
	return nil
}

func nodeSettings(nc *config, scheme proto.Scheme) (*settings.NodeSettings, error) {
	conf := &settings.NodeSettings{}
	err := settings.ApplySettings(conf, FromArgs(scheme, nc), settings.FromJavaEnviron)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply node settings")
	}

	err = conf.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate node settings")
	}
	return conf, nil
}

func embeddedWallet(nc *config, scheme proto.Scheme) (types.EmbeddedWallet, error) {
	wal := wallet.NewEmbeddedWallet(wallet.NewLoader(nc.walletPath), wallet.NewWallet(), scheme)
	if nc.walletPassword != "" {
		if err := wal.Load([]byte(nc.walletPassword)); err != nil {
			return nil, errors.Wrap(err, "failed to load wallet")
		}
	}
	return wal, nil
}

func spawnPeersByAddresses(ctx context.Context, addressesByComma string, pm *peers.PeerManagerImpl) error {
	if addressesByComma == "" { // That means that we don't have any peers to connect to
		return nil
	}
	addresses := strings.Split(addressesByComma, ",")
	for _, addr := range addresses {
		peerInfos, err := proto.NewPeerInfosFromString(addr)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve TCP addresses from string %q", addr)
		}
		for _, pi := range peerInfos {
			tcpAddr := proto.NewTCPAddr(pi.Addr, int(pi.Port))
			if tcpAddr.Empty() {
				return errors.Errorf("failed to create TCP address from IP %q and port %d",
					fmt.Stringer(pi.Addr), pi.Port,
				)
			}
			if pErr := pm.AddAddress(ctx, tcpAddr); pErr != nil {
				// That means that we have problems with peers storage
				return errors.Wrapf(pErr, "failed to add address %q into known peers storage", tcpAddr.String())
			}
		}
	}
	return nil
}

func newMinerScheduler(
	nc *config,
	st state.State,
	wal types.EmbeddedWallet,
	cfg *settings.BlockchainSettings,
	ntpTime types.Time,
	peerManager peers.PeerManager,
) (Scheduler, error) {
	if nc.disableMiner {
		return scheduler.DisabledScheduler{}, nil
	}
	consensus := scheduler.NewMinerConsensus(peerManager, nc.minPeersMining)
	ms, err := scheduler.NewScheduler(st, wal, cfg, ntpTime, consensus, nc.obsolescencePeriod, nc.generateInPast)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize miner scheduler")
	}
	return ms, nil
}

func minerFeatures(st state.State, minerVoteFeaturesByComma string) (miner.Features, error) {
	features, err := miner.ParseVoteFeatures(minerVoteFeaturesByComma)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse '-vote'")
	}

	features, err = miner.ValidateFeatures(st, features)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate features")
	}
	return features, nil
}

func closeIfErrorf(closer io.Closer, retErr error, format string, args ...interface{}) error {
	if retErr != nil {
		if clErr := closer.Close(); clErr != nil {
			return stderrs.Join(retErr, errors.Wrapf(clErr, format, args...))
		}
	}
	return retErr
}

func createPeerManager(
	nc *config,
	conf *settings.NodeSettings,
	parent peer.Parent,
	declAddr proto.TCPAddr,
) (*peers.PeerManagerImpl, error) {
	nodeNonce, err := rand.Int(rand.Reader, new(big.Int).SetUint64(math.MaxInt32))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node's nonce")
	}

	peerSpawnerImpl := peers.NewPeerSpawner(
		parent,
		conf.WavesNetwork,
		declAddr,
		nc.nodeName,
		nodeNonce.Uint64(),
		proto.ProtocolVersion(),
	)
	peerStorage, err := peersPersistentStorage.NewCBORStorage(nc.statePath, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open or create peers storage")
	}
	if nc.dropPeers {
		if err := peerStorage.DropStorage(); err != nil {
			return nil, errors.Wrap(err, "failed to drop peers storage (drop peers storage manually)")
		}
		zap.S().Info("Successfully dropped peers storage")
	}
	return peers.NewPeerManager(
		peerSpawnerImpl,
		peerStorage,
		int(nc.limitAllConnections/2),
		proto.ProtocolVersion(),
		conf.WavesNetwork,
		!nc.disableOutgoingConnections,
		nc.newConnectionsLimit,
		nc.blackListResidenceTime,
	), nil
}

func createServices(
	nc *config,
	st state.State,
	wal types.EmbeddedWallet,
	cfg *settings.BlockchainSettings,
	ntpTime types.Time,
	peerManager peers.PeerManager,
	parent peer.Parent,
	scheduler Scheduler,
) (services.Services, error) {
	utxValidator, err := utxpool.NewValidator(st, ntpTime, nc.obsolescencePeriod)
	if err != nil {
		return services.Services{}, errors.Wrap(err, "failed to initialize UTX")
	}
	return services.Services{
		State:           st,
		Peers:           peerManager,
		Scheduler:       scheduler,
		BlocksApplier:   blocks_applier.NewBlocksApplier(),
		UtxPool:         utxpool.New(utxPoolMaxSizeBytes, utxValidator, cfg),
		Scheme:          cfg.AddressSchemeCharacter,
		Time:            ntpTime,
		Wallet:          wal,
		MicroBlockCache: microblock_cache.NewMicroBlockCache(),
		InternalChannel: messages.NewInternalChannel(),
		MinPeersMining:  nc.minPeersMining,
		SkipMessageList: parent.SkipMessageList,
	}, nil
}

func runAPIs(
	ctx context.Context,
	nc *config,
	conf *settings.NodeSettings,
	app *api.App,
	svs services.Services,
) error {
	if nc.enableGrpcAPI {
		if sErr := runGRPCServer(ctx, conf.GrpcAddr, nc, svs); sErr != nil {
			return errors.Wrap(sErr, "failed to run gRPC server")
		}
	}

	webAPI := api.NewNodeAPI(app, svs.State)
	go func() {
		zap.S().Infof("Starting node HTTP API on '%v'", conf.HttpAddr)
		if runErr := api.Run(ctx, conf.HttpAddr, webAPI, apiRunOptsFromCLIFlags(nc)); runErr != nil {
			zap.S().Errorf("Failed to start API: %v", runErr)
		}
	}()
	return nil
}

func FromArgs(scheme proto.Scheme, c *config) func(s *settings.NodeSettings) error {
	return func(s *settings.NodeSettings) error {
		s.DeclaredAddr = c.declAddr
		s.HttpAddr = c.apiAddr
		s.GrpcAddr = c.grpcAddr
		s.WavesNetwork = proto.NetworkStrFromScheme(scheme)
		s.Addresses = c.peerAddresses
		if c.peerAddresses == "" && !c.disableOutgoingConnections {
			s.Addresses = defaultPeers[c.blockchainType]
		}
		return nil
	}
}

func apiRunOptsFromCLIFlags(c *config) *api.RunOptions {
	// TODO: add more run flags to CLI flags
	opts := api.DefaultRunOptions()
	opts.MaxConnections = c.apiMaxConnections
	if c.enableMetaMaskAPI {
		if c.buildExtendedAPI {
			opts.EnableMetaMaskAPI = c.enableMetaMaskAPI
			opts.EnableMetaMaskAPILog = c.enableMetaMaskAPILog
		} else {
			zap.S().Warn("'enable-metamask' flag requires activated 'build-extended-api' flag")
		}
	}
	if c.rateLimiterOptions != "" {
		rlo, err := api.NewRateLimiterOptionsFromString(c.rateLimiterOptions)
		if err == nil {
			opts.RateLimiterOpts = rlo
		} else {
			zap.S().Errorf("Invalid rate limiter options '%s': %v", c.rateLimiterOptions, err)
		}
	}
	return opts
}

func grpcAPIRunOptsFromCLIFlags(c *config) *server.RunOptions {
	opts := server.DefaultRunOptions()
	opts.MaxConnections = c.grpcAPIMaxConnections
	return opts
}

func getNtp(ctx context.Context, disable bool) (types.Time, error) {
	if disable {
		return ntptime.Stub{}, nil
	}
	tm, err := ntptime.TryNew("pool.ntp.org", 10)
	if err != nil {
		return nil, err
	}
	go tm.Run(ctx, 2*time.Minute)
	return tm, nil
}
