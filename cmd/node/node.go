package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mr-tron/base58"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/metamask"
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
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 3, Patch: 0}

const (
	maxTransactionTimeForwardOffset = 300 // seconds
	mb                              = 1 << (10 * 2)
)

var (
	logLevel                              = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath                             = flag.String("state-path", "", "Path to node's state directory")
	blockchainType                        = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	peerAddresses                         = flag.String("peers", "", "Addresses of peers to connect to")
	declAddr                              = flag.String("declared-address", "", "Address to listen on")
	nodeName                              = flag.String("name", "gowaves", "Node name.")
	cfgPath                               = flag.String("cfg-path", "", "Path to configuration JSON file, only for custom blockchain.")
	apiAddr                               = flag.String("api-address", "", "Address for REST API")
	apiKey                                = flag.String("api-key", "", "Api key")
	grpcAddr                              = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	enableMetaMaskService                 = flag.Bool("enable-metamask", true, "Enables/disables metamask service")
	metaMaskServiceAddr                   = flag.String("metamask-address", "127.0.0.1:8545", "Address for ethereum compatible RPC API for MetaMask.")
	enableGrpcApi                         = flag.Bool("enable-grpc-api", false, "Enables/disables gRPC API")
	buildExtendedApi                      = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be re-imported in case it wasn't imported with similar flag set")
	serveExtendedApi                      = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point")
	buildStateHashes                      = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
	bindAddress                           = flag.String("bind-address", "", "Bind address for incoming connections. If empty, will be same as declared address")
	disableOutgoingConnections            = flag.Bool("no-connections", false, "Disable outgoing network connections to peers. Default value is false.")
	minerVoteFeatures                     = flag.String("vote", "", "Miner vote features")
	bloomFilter                           = flag.Bool("bloom", true, "Enable/Disable bloom filter. Less memory usage, but decrease performance. Usage: -bloom=false")
	reward                                = flag.String("reward", "", "Miner reward: for example 600000000")
	outdatePeriod                         = flag.String("outdate", "4h", "Interval after last block then generation is allowed. Example 1d4h30m")
	walletPath                            = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default.")
	walletPassword                        = flag.String("wallet-password", "", "Pass password for wallet.")
	limitAllConnections                   = flag.Uint("limit-connections", 60, "Total limit of network connections, both inbound and outbound. Divided in half to limit each direction. Default value is 60.")
	minPeersMining                        = flag.Int("min-peers-mining", 1, "Minimum connected peers for allow mining.")
	disableMiner                          = flag.Bool("disable-miner", false, "Disable miner. Enabled by default.")
	profiler                              = flag.Bool("profiler", false, "Start built-in profiler on 'http://localhost:6060/debug/pprof/'")
	prometheus                            = flag.String("prometheus", "", "Provide collected metrics by prometheus client.")
	integrationGenesisSignature           = flag.String("integration.genesis.signature", "", "Integration. Genesis signature.")
	integrationGenesisTimestamp           = flag.Int("integration.genesis.timestamp", 0, "??")
	integrationGenesisBlockTimestamp      = flag.Int("integration.genesis.block-timestamp", 0, "??")
	integrationAccountSeed                = flag.String("integration.account-seed", "", "??")
	integrationAddressSchemeCharacter     = flag.String("integration.address-scheme-character", "", "??")
	integrationMinAssetInfoUpdateInterval = flag.Int("integration.min-asset-info-update-interval", 100000, "Minimum asset info update interval for integration tests.")
	metricsID                             = flag.Int("metrics-id", -1, "ID of the node on the metrics collection system")
	metricsURL                            = flag.String("metrics-url", "", "URL of InfluxDB or Telegraf in form of 'http://username:password@host:port/db'")
	dropPeers                             = flag.Bool("drop-peers", false, "Drop peers storage before node start.")
	dbFileDescriptors                     = flag.Int("db-file-descriptors", state.DefaultOpenFilesCacheCapacity, fmt.Sprintf("Maximum allowed file descriptors count that will be used by state database. Default value is %d.", state.DefaultOpenFilesCacheCapacity))
	newConnectionsLimit                   = flag.Int("new-connections-limit", 10, "Number of new outbound connections established simultaneously, defaults to 10. Should be positive. Big numbers can badly affect file descriptors consumption.")
)

var defaultPeers = map[string]string{
	"mainnet":  "34.253.153.4:6868,168.119.116.189:6868,135.181.87.72:6868,35.158.218.156:6868,52.48.34.89:6868",
	"testnet":  "159.69.126.149:6868,94.130.105.239:6868,159.69.126.153:6868,94.130.172.201:6868",
	"stagenet": "88.99.185.128:6868,49.12.15.166:6868,95.216.205.3:6868,88.198.179.16:6868",
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
	zap.S().Debugf("enable-grpc-api: %v", *enableGrpcApi)
	zap.S().Debugf("build-extended-api: %v", *buildExtendedApi)
	zap.S().Debugf("serve-extended-api: %v", *serveExtendedApi)
	zap.S().Debugf("build-state-hashes: %v", *buildStateHashes)
	zap.S().Debugf("bind-address: %s", *bindAddress)
	zap.S().Debugf("vote: %s", *minerVoteFeatures)
	zap.S().Debugf("reward: %s", *reward)
	zap.S().Debugf("miner-delay: %s", *outdatePeriod)
	zap.S().Debugf("disable-miner %v", *disableMiner)
	zap.S().Debugf("wallet-path: %s", *walletPath)
	zap.S().Debugf("hashed wallet-password: %s", crypto.MustFastHash([]byte(*walletPassword)))
	zap.S().Debugf("limit-connections: %d", *limitAllConnections)
	zap.S().Debugf("profiler: %v", *profiler)
	zap.S().Debugf("bloom: %v", *bloomFilter)
	zap.S().Debugf("drop-peers: %v", *dropPeers)
	zap.S().Debugf("db-file-descriptors: %v", *dbFileDescriptors)
	zap.S().Debugf("new-connections-limit: %v", *newConnectionsLimit)
	zap.S().Debugf("enable-metamask: %v", *enableMetaMaskService)
	zap.S().Debugf("metamask-address: %v", *metaMaskServiceAddr)
}

func main() {
	flag.Parse()

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

	common.SetupLogger(*logLevel)

	if *profiler {
		zap.S().Infof("Starting built-in profiler on 'http://localhost:6060/debug/pprof/'")
		go func() {
			zap.S().Warn(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())

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
	if *blockchainType == "integration" {
		cfg = settings.GetIntegrationSetting()
		cfg = applyIntegrationSettings(cfg)
		zap.S().Debugf("cfg: %+v", cfg)
	} else {
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
				zap.S().Error(err)
				return
			}
		}
	}

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(cfg.AddressSchemeCharacter), settings.FromJavaEnviron); err != nil {
		zap.S().Error(err)
		return
	}

	err = conf.Validate()
	if err != nil {
		zap.S().Error(err)
		return
	}

	var wal types.EmbeddedWallet = wallet.NewEmbeddedWallet(wallet.NewLoader(*walletPath), wallet.NewWallet(), cfg.AddressSchemeCharacter)
	if *blockchainType == "integration" {
		decoded, err := base58.Decode(*integrationAccountSeed)
		if err != nil {
			zap.S().Error(err)
			return
		}
		wal = wallet.Stub{
			S: [][]byte{decoded},
		}
	}
	if *walletPassword != "" {
		err := wal.Load([]byte(*walletPassword))
		if err != nil {
			zap.S().Error(err)
			return
		}
	}

	path := *statePath
	if path == "" {
		path, err = common.GetStatePath()
		if err != nil {
			zap.S().Error(err)
			return
		}
	}

	reward, err := miner.ParseReward(*reward)
	if err != nil {
		zap.S().Error(err)
		return
	}

	ntpTime, err := getNtp(ctx)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	outdatePeriodSeconds, err := common.ParseDuration(*outdatePeriod)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	params := state.DefaultStateParams()
	params.StorageParams.DbParams.OpenFilesCacheCapacity = *dbFileDescriptors
	params.StoreExtendedApiData = *buildExtendedApi
	params.ProvideExtendedApi = *serveExtendedApi
	params.BuildStateHashes = *buildStateHashes
	params.Time = ntpTime
	if !*bloomFilter {
		params.DbParams.BloomFilterParams.Disable = true
	}
	st, err := state.NewState(path, params, cfg)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	features, err := miner.ParseVoteFeatures(*minerVoteFeatures)
	if err != nil {
		cancel()
		zap.S().Error(err)
		return
	}

	features, err = miner.ValidateFeatures(st, features)
	if err != nil {
		cancel()
		zap.S().Error(err)
		return
	}

	// Check if we need to start serving extended API right now.
	if err := node.MaybeEnableExtendedApi(st, ntpTime); err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	async := runner.NewAsync()
	logRunner := runner.NewLogRunner(async)

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)
	bindAddr := proto.NewTCPAddrFromString(*bindAddress)

	utx := utxpool.New(uint64(1024*mb), utxpool.NewValidator(st, ntpTime, outdatePeriodSeconds*1000), cfg)
	parent := peer.NewParent()
	peerSpawnerImpl := peer_manager.NewPeerSpawner(parent, conf.WavesNetwork, declAddr, *nodeName, uint64(rand.Int()), version)
	peerStorage, err := peersPersistentStorage.NewCBORStorage(*statePath, time.Now())
	if err != nil {
		zap.S().Errorf("Failed to open or create peers storage: %v", err)
		cancel()
		return
	}
	if *dropPeers {
		if err := peerStorage.DropStorage(); err != nil {
			zap.S().Errorf(
				"Failed to drop peers storage. Drop peers storage manually. Err: %v",
				err,
			)
			cancel()
			return
		}
		zap.S().Info("Successfully dropped peers storage")
	}

	peerManager := peer_manager.NewPeerManager(
		peerSpawnerImpl,
		peerStorage,
		int(*limitAllConnections/2),
		version,
		conf.WavesNetwork,
		!*disableOutgoingConnections,
		*newConnectionsLimit,
	)
	go peerManager.Run(ctx)

	var minerScheduler Scheduler = scheduler.NewScheduler(
		st,
		wal,
		cfg,
		ntpTime,
		scheduler.NewMinerConsensus(peerManager, *minPeersMining),
		proto.NewTimestampFromUSeconds(outdatePeriodSeconds),
	)
	if *disableMiner {
		minerScheduler = scheduler.DisabledScheduler{}
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
	}

	mine := miner.NewMicroblockMiner(svs, features, reward, maxTransactionTimeForwardOffset)
	go miner.Run(ctx, mine, minerScheduler, svs.InternalChannel)

	n := node.NewNode(svs, declAddr, bindAddr, proto.NewTimestampFromUSeconds(outdatePeriodSeconds))
	go n.Run(ctx, parent, svs.InternalChannel)

	go minerScheduler.Reschedule()

	if len(conf.Addresses) > 0 {
		addresses := strings.Split(conf.Addresses, ",")
		for _, addr := range addresses {
			tcpAddr := proto.NewTCPAddrFromString(addr)
			if tcpAddr.Empty() {
				// That means that configuration parameter is invalid
				zap.S().Errorf("Failed to parse TCPAddr from string %q", tcpAddr.String())
				cancel()
				return
			}
			if err := peerManager.AddAddress(ctx, tcpAddr); err != nil {
				// That means that we have problems with peers storage
				zap.S().Errorf("Failed to add addres into know peers storage: %v", err)
				cancel()
				return
			}
		}
	}

	app, err := api.NewApp(*apiKey, minerScheduler, svs)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	webApi := api.NewNodeApi(app, st, n)
	go func() {
		zap.S().Infof("Starting node HTTP API on '%v'", conf.HttpAddr)
		err := api.Run(ctx, conf.HttpAddr, webApi)
		if err != nil {
			zap.S().Errorf("Failed to start API: %v", err)
		}
	}()

	go func() {
		if *prometheus != "" {
			h := http.NewServeMux()
			h.Handle("/metrics", promhttp.Handler())
			s := &http.Server{Addr: *prometheus, Handler: h}
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
			err := grpcServer.Run(ctx, conf.GrpcAddr)
			if err != nil {
				zap.S().Errorf("grpcServer.Run(): %v", err)
			}
		}()
	}

	if *enableMetaMaskService {
		if *buildExtendedApi {
			service := metamask.NewRPCService(&svs)
			go func() {
				zap.S().Infof("Starting metamask service on %s...", *metaMaskServiceAddr)
				// TODO(nickeskov): add parameter for `enableRpcServiceLog`
				err := metamask.RunMetaMaskService(ctx, *metaMaskServiceAddr, service, true)
				if err != nil {
					zap.S().Errorf("metamask service: %v", err)
				}
			}()
		} else {
			zap.S().Warn("'enable-grpc-api' flag requires activated 'build-extended-api' flag")
		}
	}

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	sig := <-gracefulStop
	zap.S().Infof("Caught signal '%s', stopping...", sig)
	cancel()
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

func applyIntegrationSettings(blockchainSettings *settings.BlockchainSettings) *settings.BlockchainSettings {
	blockchainSettings.Genesis.BlockSignature = crypto.MustSignatureFromBase58(*integrationGenesisSignature)
	blockchainSettings.Genesis.Timestamp = uint64(*integrationGenesisBlockTimestamp)

	zap.S().Debugf("applyIntegrationSettings: *integrationGenesisBlockTimestamp = %d", *integrationGenesisBlockTimestamp)

	for _, t := range blockchainSettings.Genesis.Transactions {
		t.(*proto.Genesis).Timestamp = uint64(*integrationGenesisTimestamp)
	}
	blockchainSettings.AddressSchemeCharacter = (*integrationAddressSchemeCharacter)[0]
	blockchainSettings.AverageBlockDelaySeconds = blockchainSettings.AverageBlockDelaySeconds / 2
	blockchainSettings.MinUpdateAssetInfoInterval = uint64(*integrationMinAssetInfoUpdateInterval)

	// scala value is 50_000
	blockchainSettings.Genesis.BaseTarget = 500_000

	return blockchainSettings
}

func getNtp(ctx context.Context) (types.Time, error) {
	if *blockchainType == "integration" {
		return ntptime.Stub{}, nil
	}
	tm, err := ntptime.TryNew("pool.ntp.org", 10)
	if err != nil {
		return nil, err
	}
	go tm.Run(ctx, 2*time.Minute)
	return tm, nil
}
