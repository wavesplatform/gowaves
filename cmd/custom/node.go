package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/miner"
	scheduler2 "github.com/wavesplatform/gowaves/pkg/miner/scheduler"
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
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 1, Patch: 2}

var (
	logLevel          = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath         = flag.String("state-path", "", "Path to node's state directory")
	peerAddresses     = flag.String("peers", "", "Addresses of peers to connect to")
	declAddr          = flag.String("declared-address", "", "Address to listen on")
	apiAddr           = flag.String("api-address", "", "Address for REST API")
	grpcAddr          = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	cfgPath           = flag.String("cfg-path", "", "Path to configuration JSON file. No default value.")
	enableGrpcApi     = flag.Bool("enable-grpc-api", true, "Enables/disables gRPC API")
	buildExtendedApi  = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be reimported in case it wasn't imported with similar flag set")
	serveExtendedApi  = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point")
	minerVoteFeatures = flag.String("vote", "", "Miner vote features")
	reward            = flag.String("reward", "", "Miner reward: for example 600000000")
	minerDelayParam   = flag.String("miner-delay", "4h", "Interval after last block then generation is allowed. example 1d4h30m")
	walletPath        = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default")
	walletPassword    = flag.String("wallet-password", "", "Pass password for wallet. Extremely insecure")
	limitConnectionsS = flag.String("limit-connections", "30", "N incoming and outgoing connections")
)

func init() {
	common.SetupLogger(*logLevel)
}

func main() {
	flag.Parse()
	if *cfgPath == "" {
		zap.S().Error("Please provide path to blockchain config JSON file")
		return
	}
	zap.S().Info(os.Args)
	zap.S().Info(os.Environ())
	zap.S().Info(os.LookupEnv("WAVES_OPTS"))

	f, err := os.Open(*cfgPath)
	if err != nil {
		zap.S().Fatalf("Failed to open configuration file: %v", err)
	}
	defer func() { _ = f.Close() }()
	custom, err := settings.ReadBlockchainSettings(f)
	if err != nil {
		zap.S().Fatalf("Failed to read configuration file: %v", err)
	}

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(custom.AddressSchemeCharacter), settings.FromJavaEnviron); err != nil {
		zap.S().Error(err)
		return
	}

	reward, err := miner.ParseReward(*reward)
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Info("conf", conf)

	err = conf.Validate()
	if err != nil {
		zap.S().Error(err)
		return
	}

	wal := wallet.NewEmbeddedWallet(wallet.NewLoader(*walletPath), wallet.NewWallet(), custom.AddressSchemeCharacter)
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

	ntptm, err := ntptime.TryNew("pool.ntp.org", 10)
	if err != nil {
		zap.S().Error(err)
		return
	}

	minerDelaySecond, err := common.ParseDuration(*minerDelayParam)
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
	state, err := state.NewState(path, params, custom)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	// Check if we need to start serving extended API right now.
	if err := node.MaybeEnableExtendedApi(state, ntptm); err != nil {
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

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	mb := 1024 * 1014
	btsPool := bytespool.NewBytesPool(64, mb+(mb/2))

	parent := peer.NewParent()

	peerSpawnerImpl := peer_manager.NewPeerSpawner(btsPool, parent, conf.WavesNetwork, declAddr, "gowaves", uint64(rand.Int()), version)

	peerManager := peer_manager.NewPeerManager(peerSpawnerImpl, state, int(limitConnections))
	go peerManager.Run(ctx)

	scheduler := scheduler2.NewScheduler(
		state,
		wal,
		custom,
		ntptm,
		scheduler2.NewMinerConsensus(peerManager, 1),
		proto.NewTimestampFromUSeconds(minerDelaySecond),
	)

	utx := utxpool.New(10000, utxpool.NewValidator(state, ntptm), custom)

	stateChanged := state_changed.NewStateChanged()
	blockApplier := node.NewBlocksApplier(state, ntptm)

	services := services.Services{
		State:              state,
		Peers:              peerManager,
		Scheduler:          scheduler,
		BlocksApplier:      blockApplier,
		UtxPool:            utx,
		Scheme:             custom.AddressSchemeCharacter,
		BlockAddedNotifier: stateChanged,
		Subscribe:          node.NewSubscribeService(),
		InvRequester:       ng.NewInvRequester(),
	}

	utxClean := utxpool.NewCleaner(services)

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)

	Miner := miner.NewMicroblockMiner(services, ngRuntime, proto.CustomNetScheme, features, reward)

	async := runner.NewAsync()
	scoreSender := scoresender.New(peerManager, state, 4*time.Second, async)

	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		scheduler.Reschedule()
	}))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		ngState.BlockApplied()
	}))
	stateChanged.AddHandler(utxClean)

	async.Go(func() {
		scoreSender.Run(ctx)
	})
	stateSync := node.NewStateSync(services, scoreSender, blockApplier)

	go miner.Run(ctx, Miner, scheduler)
	go scheduler.Reschedule()

	n := node.NewNode(services, declAddr, declAddr, ngRuntime, stateSync)

	go n.Run(ctx, parent)

	if len(conf.Addresses) > 0 {
		adrs := strings.Split(conf.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	// TODO hardcore
	app, err := api.NewApp("integration-test-rest-api", scheduler, stateSync, services)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	webApi := api.NewNodeApi(app, state, n)
	go func() {
		zap.S().Info("===== ", conf.HttpAddr)
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
	n.Close()

	zap.S().Infow("Caught signal, stopping", "signal", sig)
	cancel()

	<-time.After(2 * time.Second)
}

func FromArgs(scheme proto.Scheme) func(s *settings.NodeSettings) error {
	return func(s *settings.NodeSettings) error {
		s.DeclaredAddr = *declAddr
		s.HttpAddr = *apiAddr
		s.GrpcAddr = *grpcAddr
		s.WavesNetwork = proto.NetworkStrFromScheme(scheme)
		s.Addresses = *peerAddresses
		return nil
	}
}
