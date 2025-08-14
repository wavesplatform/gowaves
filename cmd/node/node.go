package main

import (
	"context"
	"crypto/rand"
	stderrs "errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
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
	"golang.org/x/sync/errgroup"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/blockchaininfo"
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

const (
	netNamespace     = "NET"
	netDataNamespace = "NET.DATA"
	fsmNamespace     = "FSM"
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

	lp                            logging.Parameters
	logNetwork                    bool
	logNetworkData                bool
	logFSM                        bool
	statePath                     string
	blockchainType                string
	peerAddresses                 string
	declAddr                      string
	nodeName                      string
	cfgPath                       string
	apiAddr                       string
	apiKey                        string
	apiMaxConnections             int
	rateLimiterOptions            string
	grpcAddr                      string
	grpcAPIMaxConnections         int
	enableMetaMaskAPI             bool
	enableMetaMaskAPILog          bool
	enableGrpcAPI                 bool
	blackListResidenceTime        time.Duration
	buildExtendedAPI              bool
	serveExtendedAPI              bool
	buildStateHashes              bool
	bindAddress                   string
	disableOutgoingConnections    bool
	minerVoteFeatures             string
	disableBloomFilter            bool
	reward                        int64
	obsolescencePeriod            time.Duration
	walletPath                    string
	walletPassword                string
	limitAllConnections           uint
	minPeersMining                int
	disableMiner                  bool
	profiler                      bool
	prometheus                    string
	metricsID                     int
	metricsURL                    string
	dropPeers                     bool
	dbFileDescriptors             uint
	newConnectionsLimit           int
	disableNTP                    bool
	microblockInterval            time.Duration
	enableLightMode               bool
	generateInPast                bool
	enableBlockchainUpdatesPlugin bool
	blockchainUpdatesL2Address    string
	h                             slog.Handler
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

func (c *config) String() string {
	return fmt.Sprintf("{Logger: %s, log-network: %t, log-fsm: %t, state-path: %s, blockchain-type: %s, "+
		"peers: %s, declared-address: %s, api-address: %s, api-key: %s, grpc-address: %s, "+
		"enable-grpc-api: %t, black-list-residence-time: %s, build-extended-api: %t, serve-extended-api: %t, "+
		"build-state-hashes: %t, bind-address: %s, vote: %s, reward: %d, obsolescence: %s, disable-miner: %t, "+
		"wallet-path: %s, hashed wallet-password: %s, limit-connections: %d, profiler: %t, "+
		"disable-bloom: %t, drop-peers: %t, db-file-descriptors: %d, new-connections-limit: %d, "+
		"enable-metamask: %t, disable-ntp: %t, microblock-interval: %s, enable-light-mode: %t, generate-in-past: %t, "+
		"enable-blockchain-updates-plugin: %t, l2-contract-address: %s}",
		c.lp.String(), c.logNetwork, c.logFSM, c.statePath, c.blockchainType,
		c.peerAddresses, c.declAddr, c.apiAddr, crypto.MustKeccak256([]byte(c.apiKey)).Hex(), c.grpcAddr,
		c.enableGrpcAPI, c.blackListResidenceTime, c.buildExtendedAPI, c.serveExtendedAPI,
		c.buildStateHashes, c.bindAddress, c.minerVoteFeatures, c.reward, c.obsolescencePeriod, c.disableMiner,
		c.walletPath, crypto.MustKeccak256([]byte(c.walletPassword)).Hex(), c.limitAllConnections, c.profiler,
		c.disableBloomFilter, c.dropPeers, c.dbFileDescriptors, c.newConnectionsLimit,
		c.enableMetaMaskAPI, c.disableNTP, c.microblockInterval, c.enableLightMode, c.generateInPast,
		c.enableBlockchainUpdatesPlugin, c.blockchainUpdatesL2Address)
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
	c.lp = logging.Parameters{}
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
	flag.BoolVar(&c.enableBlockchainUpdatesPlugin, "enable-blockchain-info", false,
		"Turn on blockchain updates plugin")
	flag.StringVar(&c.blockchainUpdatesL2Address, "l2-contract-address", "",
		"Specify the smart contract address from which the updates will be pulled")
	flag.BoolVar(&c.generateInPast, "generate-in-past", false,
		"Enable block generation with timestamp in the past")
	c.lp.Initialize()
	flag.Parse()
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
	if err := nc.lp.Parse(); err != nil {
		slog.Error("Failed to parse application parameters", logging.Error(err))
		return 1
	}
	nc.h = logging.DefaultHandler(nc.lp)
	slog.SetDefault(slog.New(nc.h))
	err := run(nc)
	if err != nil {
		slog.Error("Failed to run node", logging.Error(err))
		return 1
	}
	return 0
}

func run(nc *config) (retErr error) {
	eg, ctx := errgroup.WithContext(context.Background())
	defer func() {
		if wErr := eg.Wait(); !errors.Is(wErr, context.Canceled) {
			retErr = stderrs.Join(retErr, wErr)
		}
	}()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if nc.profiler {
		eg.Go(func() error {
			<-runProfiler(ctx)
			return nil
		})
	}
	if nc.prometheus != "" {
		eg.Go(func() error {
			<-runPrometheusMetricsServer(ctx, nc.prometheus)
			return nil
		})
	}

	slog.Info("Gowaves Node", "version", versioning.Version)
	slog.Debug("Starting with parameters", "parameters", nc.String())

	if err := raiseToMaxFDs(nc); err != nil { // raiseToMaxFDs raises the limit of file descriptors
		return errors.Wrap(err, "failed to raise file descriptors limit")
	}

	if nc.metricsURL != "" && nc.metricsID != -1 {
		err := metrics.Start(ctx, nc.metricsID, nc.metricsURL)
		if err != nil {
			slog.Warn("Metrics reporting failed to start", logging.Error(err))
			slog.Warn("Proceeding without reporting any metrics")
		} else {
			slog.Info("Metrics reporting activated")
		}
	}

	nodeCloser, err := runNode(ctx, nc)
	if err != nil {
		return errors.Wrap(err, "failed to run node")
	}

	<-ctx.Done()
	slog.Info("User termination in progress...")
	defer func() { <-time.After(1 * time.Second) }() // give some time to close internal node processes
	if clErr := nodeCloser.Close(); clErr != nil {
		return errors.Wrap(clErr, "failed to close node")
	}
	return nil
}

func initBlockchainUpdatesPlugin(ctx context.Context,
	l2addressContract string,
	enableBlockchainUpdatesPlugin bool,
	updatesChannel chan<- proto.BUpdatesInfo,
) (*proto.BlockchainUpdatesPluginInfo, error) {
	l2address, cnvrtErr := proto.NewAddressFromString(l2addressContract)
	if cnvrtErr != nil {
		return nil, errors.Wrapf(cnvrtErr, "failed to convert L2 contract address %q", l2addressContract)
	}
	bUpdatesPluginInfo := proto.NewBlockchainUpdatesPluginInfo(ctx, l2address, updatesChannel,
		enableBlockchainUpdatesPlugin)
	return bUpdatesPluginInfo, nil
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

	ntpTime, err := GetNtp(ctx, nc.disableNTP)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get NTP time")
	}

	params, err := stateParams(nc, ntpTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create state parameters")
	}

	updatesChannel := make(chan proto.BUpdatesInfo, blockchaininfo.UpdatesBufferedChannelSize)
	var bUpdatesPluginInfo *proto.BlockchainUpdatesPluginInfo
	if nc.enableBlockchainUpdatesPlugin {
		var initErr error
		bUpdatesPluginInfo, initErr = initBlockchainUpdatesPlugin(ctx, nc.blockchainUpdatesL2Address,
			nc.enableBlockchainUpdatesPlugin, updatesChannel)
		if initErr != nil {
			return nil, errors.Wrap(initErr, "failed to initialize blockchain updates plugin")
		}
	}
	st, err := state.NewState(path, true, params, cfg, nc.enableLightMode, bUpdatesPluginInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize node's state")
	}
	defer func() { retErr = closeIfErrorf(st, retErr, "failed to close state") }()
	makeExtensionReadyFunc := func() {
		bUpdatesPluginInfo.MakeExtensionReady()
	}

	if nc.enableBlockchainUpdatesPlugin {
		bUpdatesExtension, bUErr := initializeBlockchainUpdatesExtension(cfg, nc.blockchainUpdatesL2Address,
			st, makeExtensionReadyFunc, nc.obsolescencePeriod, ntpTime)
		if bUErr != nil {
			return nil, errors.Wrap(bUErr, "failed to run blockchain updates plugin")
		}
		go func() {
			publshrErr := bUpdatesExtension.RunBlockchainUpdatesPublisher(ctx,
				cfg.AddressSchemeCharacter, updatesChannel)
			if publshrErr != nil {
				slog.Error("Failed to run blockchain updates publisher", logging.Error(publshrErr))
			}
		}()
		slog.Info("The blockchain info extension started pulling info from smart contract address",
			"address", nc.blockchainUpdatesL2Address)
	}

	features, err := minerFeatures(st, nc.minerVoteFeatures)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse and validate miner features")
	}

	// Check if we need to start serving extended API right now.
	if eaErr := node.MaybeEnableExtendedApi(st, ntpTime); eaErr != nil {
		return nil, errors.Wrap(eaErr, "failed to enable extended API")
	}

	parent := peer.NewParent(nc.enableLightMode)
	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	nl := buildLogger(nc.h, netNamespace, nc.logNetwork)
	ndl := buildLogger(nc.h, netDataNamespace, nc.logNetworkData)

	peerManager, err := createPeerManager(nc, conf, parent, declAddr, nl, ndl)
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

	if pErr := spawnPeersByAddresses(conf.Addresses, peerManager); pErr != nil {
		return nil, errors.Wrap(pErr, "failed to spawn peers by addresses")
	}

	if apiErr := runAPIs(ctx, nc, conf, app, svs); apiErr != nil {
		return nil, errors.Wrap(apiErr, "failed to run APIs")
	}

	return startNode(ctx, nc, svs, features, minerScheduler, parent, declAddr, nl), nil
}

func startNode(
	ctx context.Context,
	nc *config,
	svs services.Services,
	features miner.Features,
	minerScheduler Scheduler,
	parent peer.Parent,
	declAddr proto.TCPAddr,
	nl *slog.Logger,
) *node.Node {
	bindAddr := proto.NewTCPAddrFromString(nc.bindAddress)

	mine := miner.NewMicroblockMiner(svs, features, nc.reward)
	go miner.Run(ctx, mine, minerScheduler, svs.InternalChannel)

	fl := buildLogger(nc.h, fsmNamespace, nc.logFSM)

	ntw, networkInfoCh := network.NewNetwork(svs, parent, nc.obsolescencePeriod, nl)
	go ntw.Run(ctx)

	n := node.NewNode(svs, declAddr, bindAddr, nc.microblockInterval, nc.enableLightMode, nl, fl)
	go n.Run(ctx, parent, svs.InternalChannel, networkInfoCh, ntw.SyncPeer())
	return n
}

func buildLogger(h slog.Handler, namespace string, enabled bool) *slog.Logger {
	if !enabled {
		return slog.New(slog.DiscardHandler)
	}
	return slog.New(h).With(slog.String(logging.NamespaceKey, namespace))
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
		slog.Info("Profiler is shutting down...")
	})
	go func() {
		slog.Info("Starting built-in profiler",
			"URL", fmt.Sprintf("http://%s/debug/pprof/", profilerAddr))
		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start profiler", logging.Error(err))
		}
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown profiler", logging.Error(err))
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
		slog.Info("Prometheus metrics server is shutting down...")
	})
	go func() {
		slog.Info("Starting prometheus metrics server", "address", prometheusAddr)
		err := s.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start prometheus metrics server", logging.Error(err))
		}
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown prometheus", logging.Error(err))
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
			slog.Error("Failed to run gRPC server", logging.Error(runErr))
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

func spawnPeersByAddresses(addressesByComma string, pm *peers.PeerManagerImpl) error {
	if addressesByComma == "" { // That means that we don't have any peers to connect to
		return nil
	}
	addresses := strings.SplitSeq(addressesByComma, ",")
	for addr := range addresses {
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
			if pErr := pm.AddAddress(tcpAddr); pErr != nil {
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

func closeIfErrorf(closer io.Closer, retErr error, format string, args ...any) error {
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
	logger, dl *slog.Logger,
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
		logger,
		dl,
	)
	peerStorage, err := peersPersistentStorage.NewCBORStorage(nc.statePath, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open or create peers storage")
	}
	if nc.dropPeers {
		if err := peerStorage.DropStorage(); err != nil {
			return nil, errors.Wrap(err, "failed to drop peers storage (drop peers storage manually)")
		}
		slog.Info("Successfully dropped peers storage")
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
		logger,
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
	utxValidator, err := utxpool.NewValidator(ntpTime, nc.obsolescencePeriod)
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
		slog.Info("Starting node HTTP API", "address", conf.HttpAddr)
		if runErr := api.Run(ctx, conf.HttpAddr, webAPI, apiRunOptsFromCLIFlags(nc)); runErr != nil {
			slog.Error("Failed to start API", logging.Error(runErr))

		}
	}()
	return nil
}

func initializeBlockchainUpdatesExtension(
	cfg *settings.BlockchainSettings,
	l2ContractAddress string,
	state state.State,
	makeExtensionReady func(),
	obsolescencePeriod time.Duration,
	ntpTime types.Time,
) (*blockchaininfo.BlockchainUpdatesExtension, error) {
	bUpdatesExtensionState, err := blockchaininfo.NewBUpdatesExtensionState(
		blockchaininfo.StoreBlocksLimit,
		cfg.AddressSchemeCharacter,
		l2ContractAddress,
		state,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize blockchain updates extension state")
	}
	l2address, cnvrtErr := proto.NewAddressFromString(l2ContractAddress)
	if cnvrtErr != nil {
		return nil, errors.Wrapf(cnvrtErr, "failed to convert L2 contract address %q", l2ContractAddress)
	}
	return blockchaininfo.NewBlockchainUpdatesExtension(l2address,
		bUpdatesExtensionState, makeExtensionReady, obsolescencePeriod, ntpTime), nil
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
			slog.Warn("'enable-metamask' flag requires activated 'build-extended-api' flag")
		}
	}
	if c.rateLimiterOptions != "" {
		rlo, err := api.NewRateLimiterOptionsFromString(c.rateLimiterOptions)
		if err == nil {
			opts.RateLimiterOpts = rlo
		} else {
			slog.Error("Invalid rate limiter options", slog.Any("options", c.rateLimiterOptions),
				logging.Error(err))
		}
	}
	return opts
}

func grpcAPIRunOptsFromCLIFlags(c *config) *server.RunOptions {
	opts := server.DefaultRunOptions()
	opts.MaxConnections = c.grpcAPIMaxConnections
	return opts
}

func GetNtp(ctx context.Context, disable bool) (types.Time, error) {
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
