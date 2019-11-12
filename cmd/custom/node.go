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
	"github.com/wavesplatform/gowaves/pkg/miner"
	scheduler2 "github.com/wavesplatform/gowaves/pkg/miner/scheduler"
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

var version = proto.Version{Major: 1, Minor: 1, Patch: 2}

var (
	logLevel      = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath     = flag.String("state-path", "", "Path to node's state directory")
	peerAddresses = flag.String("peers", "", "Addresses of peers to connect to")
	declAddr      = flag.String("declared-address", "", "Address to listen on")
	apiAddr       = flag.String("api-address", "", "Address for REST API")
	grpcAddr      = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	genesisPath   = flag.String("genesis-path", "", "Path to genesis json file")
	seed          = flag.String("seed", "", "Seed for miner")
)

func init() {
	util.SetupLogger(*logLevel)
}

func main() {
	flag.Parse()
	if *genesisPath == "" {
		zap.S().Error("Please provide path to genesis JSON file")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	zap.S().Info(os.Args)
	zap.S().Info(os.Environ())
	zap.S().Info(os.LookupEnv("WAVES_OPTS"))

	conf := &settings.NodeSettings{}
	if err := settings.ApplySettings(conf, FromArgs(), settings.FromJavaEnviron); err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	zap.S().Info("conf", conf)

	err := conf.Validate()
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	custom := settings.DefaultSettingsForCustomBlockchain(settings.FromPath(*genesisPath))

	path := *statePath
	if path == "" {
		path, err = util.GetStatePath()
		if err != nil {
			zap.S().Error(err)
			cancel()
			return
		}
	}
	state, err := state.NewState(path, state.DefaultStateParams(), custom)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	mb := 1024 * 1014
	btsPool := bytespool.NewBytesPool(64, mb+(mb/2))

	parent := peer.NewParent()

	peerSpawnerImpl := peer_manager.NewPeerSpawner(btsPool, parent, conf.WavesNetwork, declAddr, "gowaves", uint64(rand.Int()), version)

	peerManager := peer_manager.NewPeerManager(peerSpawnerImpl, state)
	go peerManager.Run(ctx)

	var keyPairs []proto.KeyPair
	if *seed != "" {
		keyPairs = append(keyPairs, proto.NewKeyPair([]byte(*seed)))
	}

	scheduler := scheduler2.NewScheduler(state, keyPairs, custom)

	utx := utxpool.New(10000)

	stateChanged := state_changed.NewStateChanged()
	blockApplier := node.NewBlockApplier(state, stateChanged, scheduler)

	services := services.Services{
		State:        state,
		Peers:        peerManager,
		Scheduler:    scheduler,
		BlockApplier: blockApplier,
		UtxPool:      utx,
		Scheme:       custom.FunctionalitySettings.AddressSchemeCharacter,
	}

	utxClean := utxpool.NewCleaner(services)
	go utxClean.Run(ctx)

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)

	Mainer := miner.NewMicroblockMiner(services, ngRuntime, proto.CustomNetScheme)

	stateChanged.AddHandler(state_changed.NewScoreSender(peerManager, state))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		scheduler.Reschedule()
	}))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		ngState.BlockApplied()
	}))
	stateChanged.AddHandler(utxClean)

	go miner.Run(ctx, Mainer, scheduler)
	go scheduler.Reschedule()

	n := node.NewNode(services, declAddr, ngRuntime, Mainer)

	go node.RunNode(ctx, n, parent)

	if len(conf.Addresses) > 0 {
		adrs := strings.Split(conf.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	// TODO hardcore
	app, err := api.NewApp("integration-test-rest-api", state, peerManager, scheduler, utx)
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
	n.Close()

	zap.S().Infow("Caught signal, stopping", "signal", sig)
	cancel()

	<-time.After(2 * time.Second)
}

func FromArgs() func(s *settings.NodeSettings) error {
	return func(s *settings.NodeSettings) error {
		s.DeclaredAddr = *declAddr
		s.HttpAddr = *apiAddr
		s.GrpcAddr = *grpcAddr
		networkStr, err := proto.NetworkStrByType("custom")
		if err != nil {
			return err
		}
		s.WavesNetwork = networkStr
		s.Addresses = *peerAddresses
		return nil
	}
}
