package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/grpc/server"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_changed"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 1, Patch: 5}

var (
	logLevel       = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath      = flag.String("state-path", "", "Path to node's state directory")
	blockchainType = flag.String("blockchain-type", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	peerAddresses  = flag.String("peers", "35.156.19.4:6868,52.50.69.247:6868,52.52.46.76:6868,52.57.147.71:6868,52.214.55.18:6868,54.176.190.226:6868", "Addresses of peers to connect to")
	declAddr       = flag.String("declared-address", "", "Address to listen on")
	apiAddr        = flag.String("api-address", "", "Address for REST API")
	grpcAddr       = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	seed           = flag.String("seed", "", "Seed for miner")
)

func main() {
	err := setMaxOpenFiles(1024)
	if err != nil {
		panic(err)
	}
	flag.Parse()

	util.SetupLogger(*logLevel)

	ctx, cancel := context.WithCancel(context.Background())

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(), settings.FromJavaEnviron); err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	zap.S().Info("conf", conf)

	err = conf.Validate()
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	cfg, err := settings.NetworkSettingsByType(*blockchainType)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	path := *statePath
	if path == "" {
		path, err = util.GetStatePath()
		if err != nil {
			zap.S().Error(err)
			cancel()
			return
		}
	}
	state, err := state.NewState(path, state.DefaultStateParams(), cfg)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	mb := 1024 * 1014
	pool := bytespool.NewBytesPool(64, mb+(mb/2))

	utx := utxpool.New(10000)

	parent := peer.NewParent()

	peerSpawnerImpl := peer_manager.NewPeerSpawner(pool, parent, conf.WavesNetwork, declAddr, "gowaves", 100500, version)

	peerManager := peer_manager.NewPeerManager(peerSpawnerImpl, state)
	go peerManager.Run(ctx)

	var keyPairs []proto.KeyPair
	if *seed != "" {
		keyPairs = append(keyPairs, proto.MustKeyPair([]byte(*seed)))
	}

	scheduler := scheduler.NewScheduler(state, keyPairs, cfg)
	stateChanged := state_changed.NewStateChanged()
	blockApplier := node.NewBlockApplier(state, stateChanged, scheduler)

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
		BlockApplier:       blockApplier,
		UtxPool:            utx,
		Scheme:             scheme,
		BlockAddedNotifier: stateChanged,
		Subscribe:          node.NewSubscribeService(),
		InvRequester:       ng.NewInvRequester(),
	}

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)

	mine := miner.NewMicroblockMiner(services, ngRuntime, cfg.AddressSchemeCharacter)
	go miner.Run(ctx, mine, scheduler)

	stateSync := node.NewStateSync(services, mine)

	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		scheduler.Reschedule()
	}))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		ngState.BlockApplied()
	}))

	n := node.NewNode(services, declAddr, ngRuntime, mine, stateSync)
	go node.RunNode(ctx, n, parent)

	go scheduler.Reschedule()

	if len(conf.Addresses) > 0 {
		adrs := strings.Split(conf.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	// TODO hardcore
	app, err := api.NewApp("integration-test-rest-api", state, peerManager, scheduler, utx, stateSync)
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

	grpcServer := server.NewServer(state)
	go func() {
		err := grpcServer.Run(ctx, conf.GrpcAddr)
		if err != nil {
			zap.S().Errorf("grpcServer.Run(): %v", err)
		}
	}()

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
