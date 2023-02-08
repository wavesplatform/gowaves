package main

import (
	"context"
	"crypto/rand"
	"flag"
	"math"
	"math/big"
	"net/http"
	_ "net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/metrics"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	peersPersistentStorage "github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
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
	"go.uber.org/zap"
)

const (
	maxTransactionTimeForwardOffset = 300 // seconds
	mb                              = 1 << (10 * 2)
	defaultTimeout                  = 30 * time.Second
)

var (
	logLevel                   = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL.")
	statePath                  = flag.String("state-path", "", "Path to node's state directory.")
	blockchainType             = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet.")
	peerAddresses              = flag.String("peers", "", "Addresses of peers to connect to.")
	declAddr                   = flag.String("declared-address", "", "Address to listen on.")
	nodeName                   = flag.String("name", "gowaves", "Node name.")
	cfgPath                    = flag.String("cfg-path", "", "Path to configuration JSON file, only for custom blockchain.")
	apiAddr                    = flag.String("api-address", "", "Address for REST API.")
	apiKey                     = flag.String("api-key", "", "Api key.")
	apiMaxConnections          = flag.Int("api-max-connections", api.DefaultMaxConnections, "Max number of simultaneous connections for REST API.")
	grpcAddr                   = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API.")
	grpcApiMaxConnections      = flag.Int("grpc-api-max-connections", server.DefaultMaxConnections, "Max number of simultaneous connections for gRPC API.")
	enableMetaMaskAPI          = flag.Bool("enable-metamask", true, "Enables/disables metamask API.")
	enableMetaMaskAPILog       = flag.Bool("enable-metamask-log", false, "Enables/disables metamask API logging.")
	enableGrpcApi              = flag.Bool("enable-grpc-api", false, "Enables/disables gRPC API.")
	blackListResidenceTime     = flag.Duration("blacklist-residence-time", 5*time.Minute, "Period of time for which the information about external peer stays in the blacklist. Default value is 5 min. To disable blacklisting pass zero value.")
	buildExtendedApi           = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be re-imported in case it wasn't imported with similar flag set.")
	serveExtendedApi           = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point.")
	buildStateHashes           = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
	bindAddress                = flag.String("bind-address", "", "Bind address for incoming connections. If empty, will be same as declared address")
	disableOutgoingConnections = flag.Bool("no-connections", false, "Disable outgoing network connections to peers.")
	minerVoteFeatures          = flag.String("vote", "", "Miner vote features.")
	disableBloomFilter         = flag.Bool("disable-bloom", false, "Disable bloom filter. Less memory usage, but decrease performance.")
	reward                     = flag.String("reward", "", "Miner reward: for example 600000000.")
	obsolescencePeriod         = flag.Duration("obsolescence", 4*time.Hour, "Blockchain obsolescence period. Disable mining if last block older then given value.")
	walletPath                 = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default.")
	walletPassword             = flag.String("wallet-password", "", "Pass password for wallet.")
	limitAllConnections        = flag.Uint("limit-connections", 60, "Total limit of network connections, both inbound and outbound. Divided in half to limit each direction.")
	minPeersMining             = flag.Int("min-peers-mining", 1, "Minimum connected peers for allow mining.")
	disableMiner               = flag.Bool("disable-miner", false, "Disable miner.")
	profiler                   = flag.Bool("profiler", false, "Start built-in profiler on 'http://localhost:6060/debug/pprof/'.")
	prometheus                 = flag.String("prometheus", "", "Provide collected metrics by prometheus client.")
	metricsID                  = flag.Int("metrics-id", -1, "ID of the node on the metrics collection system.")
	metricsURL                 = flag.String("metrics-url", "", "URL of InfluxDB or Telegraf in form of 'http://username:password@host:port/db'.")
	dropPeers                  = flag.Bool("drop-peers", false, "Drop peers storage before node start.")
	dbFileDescriptors          = flag.Int("db-file-descriptors", state.DefaultOpenFilesCacheCapacity, "Maximum allowed file descriptors count that will be used by state database.")
	newConnectionsLimit        = flag.Int("new-connections-limit", 10, "Number of new outbound connections established simultaneously, defaults to 10. Should be positive. Big numbers can badly affect file descriptors consumption.")
	disableNTP                 = flag.Bool("disable-ntp", false, "Disable NTP synchronization. Useful when running the node in a docker container.")
	microblockInterval         = flag.Duration("microblock-interval", 5*time.Second, "Interval between microblocks.")
)

var defaultPeers = map[string]string{
	"mainnet":  "34.253.153.4:6868,168.119.116.189:6868,135.181.87.72:6868,35.158.18.65:6868,52.51.9.86:6868",
	"testnet":  "159.69.126.149:6868,94.130.105.239:6868,159.69.126.153:6868,94.130.172.201:6868,35.157.247.122:6868",
	"stagenet": "88.99.185.128:6868,49.12.15.166:6868,95.216.205.3:6868,88.198.179.16:6868,52.58.254.101:6868",
}

type Scheduler interface {
	Mine() chan scheduler.Emit
	types.Scheduler
	Emits() []scheduler.Emit
}

func debugCommandLineParameters() {
	zap.S().Debugf("log-level: %s", *logLevel)
	zap.S().Debugf("state-path: %s", *statePath)
	zap.S().Debugf("blockchain-type: %s", *blockchainType)
	zap.S().Debugf("peers: %s", *peerAddresses)
	zap.S().Debugf("declared-address: %s", *declAddr)
	zap.S().Debugf("api-address: %s", *apiAddr)
	zap.S().Debugf("api-key: %s", *apiKey)
	zap.S().Debugf("grpc-address: %s", *grpcAddr)
	zap.S().Debugf("enable-grpc-api: %t", *enableGrpcApi)
	zap.S().Debugf("black-list-residence-time: %s", *blackListResidenceTime)
	zap.S().Debugf("build-extended-api: %t", *buildExtendedApi)
	zap.S().Debugf("serve-extended-api: %t", *serveExtendedApi)
	zap.S().Debugf("build-state-hashes: %t", *buildStateHashes)
	zap.S().Debugf("bind-address: %s", *bindAddress)
	zap.S().Debugf("vote: %s", *minerVoteFeatures)
	zap.S().Debugf("reward: %s", *reward)
	zap.S().Debugf("obsolescence: %s", *obsolescencePeriod)
	zap.S().Debugf("disable-miner %t", *disableMiner)
	zap.S().Debugf("wallet-path: %s", *walletPath)
	zap.S().Debugf("hashed wallet-password: %s", crypto.MustFastHash([]byte(*walletPassword)))
	zap.S().Debugf("limit-connections: %d", *limitAllConnections)
	zap.S().Debugf("profiler: %t", *profiler)
	zap.S().Debugf("disable-bloom: %t", *disableBloomFilter)
	zap.S().Debugf("drop-peers: %t", *dropPeers)
	zap.S().Debugf("db-file-descriptors: %v", *dbFileDescriptors)
	zap.S().Debugf("new-connections-limit: %v", *newConnectionsLimit)
	zap.S().Debugf("enable-metamask: %t", *enableMetaMaskAPI)
	zap.S().Debugf("disable-ntp: %t", *disableNTP)
	zap.S().Debugf("microblock-interval: %s", *microblockInterval)
}

func main() {
	flag.Parse()
	common.SetupLogger(*logLevel)

	zap.S().Infof("Gowaves Node version: %s", versioning.Version)

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		zap.S().Fatalf("Initialization failure: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization failure: %v", err)
	}
	if maxAvailableFileDescriptors := int(maxFDs) - int(*limitAllConnections) - 10; *dbFileDescriptors > maxAvailableFileDescriptors {
		zap.S().Fatalf("Invalid 'db-file-descriptors' flag value (%d). Value shall be less or equal to %d.", *dbFileDescriptors, maxAvailableFileDescriptors)
	}

	if *profiler {
		zap.S().Infof("Starting built-in profiler on 'http://localhost:6060/debug/pprof/'")
		go func() {
			pprofMux := http.NewServeMux()
			// taken from "net/http/pprof" init()
			pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
			pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			s := &http.Server{Addr: "localhost:6060", Handler: pprofMux, ReadHeaderTimeout: defaultTimeout, ReadTimeout: defaultTimeout}
			zap.S().Warn(s.ListenAndServe())
		}()
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	if *metricsURL != "" && *metricsID != -1 {
		err := metrics.Start(ctx, *metricsID, *metricsURL)
		if err != nil {
			zap.S().Warnf("Metrics reporting failed to start: %v", err)
			zap.S().Warn("Proceeding without reporting any metrics")
		} else {
			zap.S().Info("Metrics reporting activated")
		}
	}

	debugCommandLineParameters()

	var cfg *settings.BlockchainSettings
	if *cfgPath != "" {
		f, err := os.Open(*cfgPath)
		if err != nil {
			zap.S().Fatalf("Failed to open configuration file: %v", err)
		}
		defer func() { _ = f.Close() }()
		cfg, err = settings.ReadBlockchainSettings(f)
		if err != nil {
			zap.S().Fatalf("Failed to read configuration file: %v", err)
		}
	} else {
		cfg, err = settings.BlockchainSettingsByTypeName(*blockchainType)
		if err != nil {
			zap.S().Errorf("Failed to get blockchain settings: %v", err)
			return
		}
	}

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(cfg.AddressSchemeCharacter), settings.FromJavaEnviron); err != nil {
		zap.S().Errorf("Failed to apply node settings: %v", err)
		return
	}

	err = conf.Validate()
	if err != nil {
		zap.S().Errorf("Failed to validate node settings: %v", err)
		return
	}

	var wal types.EmbeddedWallet = wallet.NewEmbeddedWallet(wallet.NewLoader(*walletPath), wallet.NewWallet(), cfg.AddressSchemeCharacter)
	if *walletPassword != "" {
		err := wal.Load([]byte(*walletPassword))
		if err != nil {
			zap.S().Errorf("Failed to load wallet: %v", err)
			return
		}
	}

	path := *statePath
	if path == "" {
		path, err = common.GetStatePath()
		if err != nil {
			zap.S().Errorf("Failed to get state path: %v", err)
			return
		}
	}

	reward, err := miner.ParseReward(*reward)
	if err != nil {
		zap.S().Errorf("Failed to parse '-reward': %v", err)
		return
	}

	ntpTime, err := getNtp(ctx, *disableNTP)
	if err != nil {
		zap.S().Errorf("Failed to get NTP time: %v", err)
		return
	}

	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = *dbFileDescriptors
	params.StoreExtendedApiData = *buildExtendedApi
	params.ProvideExtendedApi = *serveExtendedApi
	params.BuildStateHashes = *buildStateHashes
	params.Time = ntpTime
	params.DbParams.BloomFilterParams.Disable = *disableBloomFilter

	st, err := state.NewState(path, true, params, cfg)
	if err != nil {
		zap.S().Error("Failed to initialize node's state: %v", err)
		return
	}

	features, err := miner.ParseVoteFeatures(*minerVoteFeatures)
	if err != nil {
		zap.S().Errorf("Failed to parse '-vote': %v", err)
		return
	}

	features, err = miner.ValidateFeatures(st, features)
	if err != nil {
		zap.S().Errorf("Failed to validate features: %v", err)
		return
	}

	// Check if we need to start serving extended API right now.
	if err := node.MaybeEnableExtendedApi(st, ntpTime); err != nil {
		zap.S().Errorf("Failed to enable extended API: %v", err)
		return
	}

	async := runner.NewAsync()
	logRunner := runner.NewLogRunner(async)

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)
	bindAddr := proto.NewTCPAddrFromString(*bindAddress)

	utxValidator, err := utxpool.NewValidator(st, ntpTime, *obsolescencePeriod)
	if err != nil {
		zap.S().Errorf("Failed to initialize UTX: %v", err)
		return
	}
	utx := utxpool.New(uint64(1024*mb), utxValidator, cfg)
	parent := peer.NewParent()

	nodeNonce, err := rand.Int(rand.Reader, new(big.Int).SetUint64(math.MaxInt32))
	if err != nil {
		zap.S().Errorf("Failed to get node's nonce: %v", err)
		return
	}
	peerSpawnerImpl := peer_manager.NewPeerSpawner(parent, conf.WavesNetwork, declAddr, *nodeName, nodeNonce.Uint64(), proto.ProtocolVersion)
	peerStorage, err := peersPersistentStorage.NewCBORStorage(*statePath, time.Now())
	if err != nil {
		zap.S().Errorf("Failed to open or create peers storage: %v", err)
		return
	}
	if *dropPeers {
		if err := peerStorage.DropStorage(); err != nil {
			zap.S().Errorf("Failed to drop peers storage. Drop peers storage manually. Err: %v", err)
			return
		}
		zap.S().Info("Successfully dropped peers storage")
	}

	peerManager := peer_manager.NewPeerManager(
		peerSpawnerImpl,
		peerStorage,
		int(*limitAllConnections/2),
		proto.ProtocolVersion,
		conf.WavesNetwork,
		!*disableOutgoingConnections,
		*newConnectionsLimit,
		*blackListResidenceTime,
	)
	go peerManager.Run(ctx)

	var minerScheduler Scheduler
	if *disableMiner {
		minerScheduler = scheduler.DisabledScheduler{}
	} else {
		minerScheduler, err = scheduler.NewScheduler(
			st,
			wal,
			cfg,
			ntpTime,
			scheduler.NewMinerConsensus(peerManager, *minPeersMining),
			*obsolescencePeriod,
		)
		if err != nil {
			zap.S().Errorf("Failed to initialize miner scheduler: %v", err)
			return
		}
	}
	blockApplier := blocks_applier.NewBlocksApplier()

	svs := services.Services{
		State:           st,
		Peers:           peerManager,
		Scheduler:       minerScheduler,
		BlocksApplier:   blockApplier,
		UtxPool:         utx,
		Scheme:          cfg.AddressSchemeCharacter,
		LoggableRunner:  logRunner,
		Time:            ntpTime,
		Wallet:          wal,
		MicroBlockCache: microblock_cache.NewMicroblockCache(),
		InternalChannel: messages.NewInternalChannel(),
		MinPeersMining:  *minPeersMining,
		SkipMessageList: parent.SkipMessageList,
	}

	mine := miner.NewMicroblockMiner(svs, features, reward, maxTransactionTimeForwardOffset)
	go miner.Run(ctx, mine, minerScheduler, svs.InternalChannel)

	n := node.NewNode(svs, declAddr, bindAddr, *microblockInterval)
	go n.Run(ctx, parent, svs.InternalChannel)

	go minerScheduler.Reschedule()

	if len(conf.Addresses) > 0 {
		addresses := strings.Split(conf.Addresses, ",")
		for _, addr := range addresses {
			tcpAddr := proto.NewTCPAddrFromString(addr)
			if tcpAddr.Empty() {
				// That means that configuration parameter is invalid
				zap.S().Errorf("Failed to parse TCPAddr from string %q", tcpAddr.String())
				return
			}
			if err := peerManager.AddAddress(ctx, tcpAddr); err != nil {
				// That means that we have problems with peers storage
				zap.S().Errorf("Failed to add addres into know peers storage: %v", err)
				return
			}
		}
	}

	app, err := api.NewApp(*apiKey, minerScheduler, svs)
	if err != nil {
		zap.S().Errorf("Failed to initialize application: %v", err)
		return
	}

	webApi := api.NewNodeApi(app, st, n)
	go func() {
		zap.S().Infof("Starting node HTTP API on '%v'", conf.HttpAddr)
		err := api.Run(ctx, conf.HttpAddr, webApi, apiRunOptsFromCLIFlags())
		if err != nil {
			zap.S().Errorf("Failed to start API: %v", err)
		}
	}()

	go func() {
		if *prometheus != "" {
			h := http.NewServeMux()
			h.Handle("/metrics", promhttp.Handler())
			s := &http.Server{Addr: *prometheus, Handler: h, ReadHeaderTimeout: defaultTimeout, ReadTimeout: defaultTimeout}
			zap.S().Infof("Starting node metrics endpoint on '%v'", *prometheus)
			_ = s.ListenAndServe()
		}
	}()

	if *enableGrpcApi {
		grpcServer, err := server.NewServer(svs)
		if err != nil {
			zap.S().Errorf("Failed to create gRPC server: %v", err)
		}
		go func() {
			err := grpcServer.Run(ctx, conf.GrpcAddr, grpcApiRunOptsFromCLIFlags())
			if err != nil {
				zap.S().Errorf("grpcServer.Run(): %v", err)
			}
		}()
	}

	<-ctx.Done()
	zap.S().Info("User termination in progress...")
	n.Close()
	<-time.After(1 * time.Second)
}

func FromArgs(scheme proto.Scheme) func(s *settings.NodeSettings) error {
	return func(s *settings.NodeSettings) error {
		s.DeclaredAddr = *declAddr
		s.HttpAddr = *apiAddr
		s.GrpcAddr = *grpcAddr
		s.WavesNetwork = proto.NetworkStrFromScheme(scheme)
		s.Addresses = *peerAddresses
		if *peerAddresses == "" {
			s.Addresses = defaultPeers[*blockchainType]
		}
		return nil
	}
}

func apiRunOptsFromCLIFlags() *api.RunOptions {
	// TODO: add more run flags to CLI flags
	opts := api.DefaultRunOptions()
	opts.MaxConnections = *apiMaxConnections
	if *enableMetaMaskAPI {
		if *buildExtendedApi {
			opts.EnableMetaMaskAPI = *enableMetaMaskAPI
			opts.EnableMetaMaskAPILog = *enableMetaMaskAPILog
		} else {
			zap.S().Warn("'enable-metamask' flag requires activated 'build-extended-api' flag")
		}
	}
	return opts
}

func grpcApiRunOptsFromCLIFlags() *server.RunOptions {
	opts := server.DefaultRunOptions()
	opts.MaxConnections = *grpcApiMaxConnections
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
