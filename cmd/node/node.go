package main

import (
	"context"
	"flag"
	"math/rand"
	"net/http"
	_ "net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 2, Patch: 3}

var (
	logLevel                   = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath                  = flag.String("state-path", "", "Path to node's state directory")
	blockchainType             = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	peerAddresses              = flag.String("peers", "", "Addresses of peers to connect to")
	declAddr                   = flag.String("declared-address", "", "Address to listen on")
	apiAddr                    = flag.String("api-address", "", "Address for REST API")
	apiKey                     = flag.String("api-key", "", "Api key")
	grpcAddr                   = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	enableGrpcApi              = flag.Bool("enable-grpc-api", true, "Enables/disables gRPC API")
	buildExtendedApi           = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be reimported in case it wasn't imported with similar flag set")
	serveExtendedApi           = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point")
	buildStateHashes           = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
	bindAddress                = flag.String("bind-address", "", "Bind address for incoming connections. If empty, will be same as declared address")
	disableOutgoingConnections = flag.Bool("no-connections", false, "Disable outgoing network connections to peers. Default value is false.")
	minerVoteFeatures          = flag.String("vote", "", "Miner vote features")
	reward                     = flag.String("reward", "", "Miner reward: for example 600000000")
	minerDelayParam            = flag.String("outdate", "4h", "Interval after last block then generation is allowed. example 1d4h30m")
	walletPath                 = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default")
	walletPassword             = flag.String("wallet-password", "", "Pass password for wallet. Extremely insecure")
	limitConnectionsS          = flag.String("limit-connections", "30", "N incoming and outgoing connections")
	minPeersMining             = flag.Int("min-peers-mining", 1, "Minimum connected peers for allow mining")
	profiler                   = flag.Bool("profiler", false, "Start built-in profiler on 'http://localhost:6060/debug/pprof/'")
)

var defaultPeers = map[string]string{
	"mainnet": "35.156.19.4:6868,52.50.69.247:6868,52.52.46.76:6868,52.57.147.71:6868,52.214.55.18:6868,54.176.190.226:6868",
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
	zap.S().Debugf("miner-delay: %s", *minerDelayParam)
	zap.S().Debugf("wallet-path: %s", *walletPath)
	zap.S().Debugf("wallet-password: %s", *walletPassword)
	zap.S().Debugf("limit-connections: %s", *limitConnectionsS)
	zap.S().Debugf("profiler: %v", *profiler)
}

func main() {
	err := setMaxOpenFiles(1024)
	if err != nil {
		panic(err)
	}
	flag.Parse()

	common.SetupLogger(*logLevel)

	if *profiler {
		zap.S().Infof("Starting built-in profiler on 'http://localhost:6060/debug/pprof/'")
		go func() {
			zap.S().Warn(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	debugCommandLineParameters()

	cfg, err := settings.BlockchainSettingsByTypeName(*blockchainType)
	if err != nil {
		zap.S().Error(err)
		return
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

	wal := wallet.NewEmbeddedWallet(wallet.NewLoader(*walletPath), wallet.NewWallet(), cfg.AddressSchemeCharacter)
	if *walletPassword != "" {
		err := wal.Load([]byte(*walletPassword))
		if err != nil {
			zap.S().Error(err)
			return
		}
	}

	limitConnections, err := strconv.ParseUint(*limitConnectionsS, 10, 64)
	if err != nil {
		zap.S().Error(err)
		return
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

	ntptm, err := ntptime.TryNew("pool.ntp.org", 10)
	if err != nil {
		zap.S().Error(err)
		return
	}

	outdatePeriod, err := common.ParseDuration(*minerDelayParam)
	if err != nil {
		zap.S().Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	go ntptm.Run(ctx, 2*time.Minute)

	params := state.DefaultStateParams()
	params.StoreExtendedApiData = *buildExtendedApi
	params.ProvideExtendedApi = *serveExtendedApi
	params.BuildStateHashes = *buildStateHashes
	params.Time = ntptm
	state, err := state.NewState(path, params, cfg)
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

	features, err = miner.ValidateFeatures(state, features)
	if err != nil {
		cancel()
		zap.S().Error(err)
		return
	}

	// Check if we need to start serving extended API right now.
	if err := node.MaybeEnableExtendedApi(state, ntptm); err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	async := runner.NewAsync()
	logRunner := runner.NewLogRunner(async)

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)
	bindAddr := proto.NewTCPAddrFromString(*bindAddress)

	mb := 1024 * 1014
	pool := bytespool.NewBytesPool(64, mb+(mb/2))

	utx := utxpool.New(uint64(1024*mb), utxpool.NewValidator(state, ntptm), cfg)

	parent := peer.NewParent()

	peerSpawnerImpl := peer_manager.NewPeerSpawner(pool, parent, conf.WavesNetwork, declAddr, "gowaves", uint64(rand.Int()), version)

	peerStorage, err := peer_manager.NewJsonFileStorage(path)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	peerManager := peer_manager.NewPeerManager(
		peerSpawnerImpl,
		peerStorage,
		int(limitConnections),
	)
	go peerManager.Run(ctx)

	scheduler := scheduler.NewScheduler(
		state,
		wal,
		cfg,
		ntptm,
		scheduler.NewMinerConsensus(peerManager, *minPeersMining),
		proto.NewTimestampFromUSeconds(outdatePeriod),
	)
	blockApplier := blocks_applier.NewBlocksApplier()

	services := services.Services{
		State:           state,
		Peers:           peerManager,
		Scheduler:       scheduler,
		BlocksApplier:   blockApplier,
		UtxPool:         utx,
		Scheme:          cfg.AddressSchemeCharacter,
		LoggableRunner:  logRunner,
		Time:            ntptm,
		Wallet:          wal,
		MicroBlockCache: microblock_cache.NewMicroblockCache(),
		InternalChannel: messages.NewInternalChannel(),
	}

	mine := miner.NewMicroblockMiner(services, features, reward)
	peerManager.SetConnectPeers(!*disableOutgoingConnections)
	go miner.Run(ctx, mine, scheduler, services.InternalChannel)

	n := node.NewNode(services, declAddr, bindAddr, proto.NewTimestampFromUSeconds(outdatePeriod))
	go n.Run(ctx, parent, services.InternalChannel)

	go scheduler.Reschedule()

	if len(conf.Addresses) > 0 {
		adrs := strings.Split(conf.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	app, err := api.NewApp(*apiKey, scheduler, services)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	webApi := api.NewNodeApi(app, state, n)
	go func() {
		err := api.Run(ctx, conf.HttpAddr, webApi)
		if err != nil {
			zap.S().Errorf("Failed to start API: %v", err)
		}
	}()

	if *enableGrpcApi {
		grpcServer, err := server.NewServer(services)
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

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	sig := <-gracefulStop
	zap.S().Infow("Caught signal, stopping", "signal", sig)
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
