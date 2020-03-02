package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_changed"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/scoresender"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 1, Patch: 5}

var (
	logLevel          = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath         = flag.String("state-path", "", "Path to node's state directory")
	blockchainType    = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	peerAddresses     = flag.String("peers", "35.156.19.4:6868,52.50.69.247:6868,52.52.46.76:6868,52.57.147.71:6868,52.214.55.18:6868,54.176.190.226:6868", "Addresses of peers to connect to")
	declAddr          = flag.String("declared-address", "", "Address to listen on")
	apiAddr           = flag.String("api-address", "", "Address for REST API")
	apiKey            = flag.String("api-key", "", "Api key")
	grpcAddr          = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	enableGrpcApi     = flag.Bool("enable-grpc-api", true, "Enables/disables gRPC API")
	buildExtendedApi  = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be reimported in case it wasn't imported with similar flag set")
	serveExtendedApi  = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point")
	bindAddress       = flag.String("bind-address", "", "Bind address for incoming connections. If empty, will be same as declared address")
	connectPeers      = flag.String("connect-peers", "true", "Spawn outgoing connections")
	minerVoteFeatures = flag.String("vote", "", "Miner vote features")
	reward            = flag.String("reward", "", "Miner reward: for example 600000000")
	minerDelayParam   = flag.String("miner-delay", "4h", "Interval after last block then generation is allowed. example 1d4h30m")
	walletPath        = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default")
	walletPassword    = flag.String("wallet-password", "", "Pass password for wallet. Extremely insecure")
)

func main() {
	err := setMaxOpenFiles(1024)
	if err != nil {
		panic(err)
	}
	flag.Parse()

	util.SetupLogger(*logLevel)

	zap.S().Info("connectPeers ", *connectPeers)

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(), settings.FromJavaEnviron); err != nil {
		zap.S().Error(err)
		return
	}

	err = conf.Validate()
	if err != nil {
		zap.S().Error(err)
		return
	}

	cfg, err := settings.BlockchainSettingsByTypeName(*blockchainType)
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

	path := *statePath
	if path == "" {
		path, err = util.GetStatePath()
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

	minerDelaySecond, err := util.ParseDuration(*minerDelayParam)
	if err != nil {
		zap.S().Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	go ntptm.Run(ctx, 2*time.Minute)

	params := state.DefaultStateParams()
	params.StoreExtendedApiData = *buildExtendedApi
	params.ProvideExtendedApi = *serveExtendedApi
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

	features, err = miner.ValidateFeaturesWithLock(state, features)
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

	utx := utxpool.New(10000, utxpool.NewValidator(state, ntptm))

	parent := peer.NewParent()

	peerSpawnerImpl := peer_manager.NewPeerSpawner(pool, parent, conf.WavesNetwork, declAddr, "gowaves", uint64(rand.Int()), version)

	peerManager := peer_manager.NewPeerManager(peerSpawnerImpl, state)
	go peerManager.Run(ctx)

	scheduler := scheduler.NewScheduler(
		state,
		wal,
		cfg,
		ntptm,
		scheduler.NewMinerConsensus(peerManager, 1),
		proto.NewTimestampFromUSeconds(minerDelaySecond),
	)
	stateChanged := state_changed.NewStateChanged()
	blockApplier := node.NewBlocksApplier(state, ntptm)

	scheme, err := proto.NetworkSchemeByType(*blockchainType)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}
	services := services.Services{
		State:              state,
		Peers:              peerManager,
		Scheduler:          scheduler,
		BlocksApplier:      blockApplier,
		UtxPool:            utx,
		Scheme:             scheme,
		BlockAddedNotifier: stateChanged,
		Subscribe:          node.NewSubscribeService(),
		InvRequester:       ng.NewInvRequester(),
		LoggableRunner:     logRunner,
		Time:               ntptm,
		Wallet:             wal,
	}

	utxClean := utxpool.NewCleaner(services)

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)
	scoreSender := scoresender.New(peerManager, state, 5*time.Second, async)
	logRunner.Named("ScoreSender.Run", func() {
		scoreSender.Run(ctx)
	})

	mine := miner.NewMicroblockMiner(services, ngRuntime, cfg.AddressSchemeCharacter, features, reward)
	peerManager.SetConnectPeers(!(*connectPeers == "false"))
	go miner.Run(ctx, mine, scheduler)

	stateSync := node.NewStateSync(services, scoreSender, node.NewBlocksApplier(state, ntptm))

	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		scheduler.Reschedule()
	}))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		ngState.BlockApplied()
	}))
	stateChanged.AddHandler(utxClean)

	n := node.NewNode(services, declAddr, bindAddr, ngRuntime, stateSync)
	go n.Run(ctx, parent)

	go scheduler.Reschedule()

	if len(conf.Addresses) > 0 {
		adrs := strings.Split(conf.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	app, err := api.NewApp(*apiKey, scheduler, stateSync, services)
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
	<-time.After(2 * time.Second)
	n.Close()

	<-time.After(2 * time.Second)
}

func FromArgs() func(s *settings.NodeSettings) error {
	return func(s *settings.NodeSettings) error {
		s.DeclaredAddr = *declAddr
		s.HttpAddr = *apiAddr
		s.GrpcAddr = *grpcAddr
		networkStr, err := proto.NetworkStrByType(*blockchainType)
		if err != nil {
			return err
		}
		s.WavesNetwork = networkStr
		s.Addresses = *peerAddresses
		return nil
	}
}
