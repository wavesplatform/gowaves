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

var version = proto.Version{Major: 1, Minor: 1, Patch: 2}

var (
	logLevel       = flag.String("logLevel", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath      = flag.String("statePath", "", "Path to node's state directory")
	blockchainType = flag.String("blockchainType", "mainnet", "Blockchain type: mainnet/testnet/stagenet")
	peerAddresses  = flag.String("peers", "35.156.19.4:6868,52.50.69.247:6868,52.52.46.76:6868,52.57.147.71:6868,52.214.55.18:6868,54.176.190.226:6868", "Addresses of peers to connect to")
	declAddr       = flag.String("declAddr", "", "Address to listen on")
	apiAddr        = flag.String("apiAddr", "", "Address for API")
)

func main() {
	err := setMaxOpenFiles(1024)
	if err != nil {
		panic(err)
	}
	flag.Parse()

	util.SetupLogger(*logLevel)

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

	scheduler := scheduler.NewScheduler(state, nil, nil)
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
	}

	mine := miner.NoOpMiner()

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)

	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		scheduler.Reschedule()
	}))
	stateChanged.AddHandler(state_changed.NewFuncHandler(func() {
		ngState.BlockApplied()
	}))

	n := node.NewNode(services, declAddr, ngRuntime, mine)
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
		err := api.Run(ctx, conf.HttpAddr, webApi)
		if err != nil {
			zap.S().Error("Failed to start API: %v", err)
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
		networkStr, err := proto.NetworkStrByType(*blockchainType)
		if err != nil {
			return err
		}
		s.WavesNetwork = networkStr
		s.Addresses = *peerAddresses
		return nil
	}
}
