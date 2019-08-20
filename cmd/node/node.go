package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/miner"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/ng"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	"github.com/wavesplatform/gowaves/pkg/node/state_changed"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 0, Minor: 16, Patch: 1}

type Cli struct {
	Run struct {
		WavesNetwork string `kong:"wavesnetwork,short='n',help='Waves network.',required"`
		Addresses    string `kong:"address,short='a',help='Addresses connect to.'"`
		DeclAddr     string `kong:"decladdr,short='d',help='Address listen on.'"`
		HttpAddr     string `kong:"httpaddr,short='w',help='Http addr bind on.'"`
	} `kong:"cmd,help='Run node'"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	zap.S().Info(os.Args)
	zap.S().Info(os.Environ())
	zap.S().Info(os.LookupEnv("WAVES_OPTS"))

	var cli Cli
	kong.Parse(&cli)

	conf := &settings.NodeSettings{}
	settings.ApplySettings(conf,
		FromArgs(&cli),
		settings.FromJavaEnviron)

	zap.S().Info("conf", conf)

	err := conf.Validate()
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	state, err := state.NewState("./", state.DefaultStateParams(), settings.MainNetSettings)
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

	scheduler := scheduler.NewScheduler(state, nil, nil)
	stateChanged := state_changed.NewStateChanged()
	blockApplier := node.NewBlockApplier(state, stateChanged, scheduler)

	services := services.Services{
		State:        state,
		Peers:        peerManager,
		Scheduler:    scheduler,
		BlockApplier: blockApplier,
		UtxPool:      utx,
		Scheme:       'W',
	}

	mine := miner.NoOpMiner()

	ngState := ng.NewState(services)
	ngRuntime := ng.NewRuntime(services, ngState)

	stateChanged.AddHandler(state_changed.NewScoreSender(peerManager, state))
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
	n.Close()

	zap.S().Infow("Caught signal, stopping", "signal", sig)
	cancel()

	<-time.After(2 * time.Second)

}

func FromArgs(c *Cli) func(s *settings.NodeSettings) {
	return func(s *settings.NodeSettings) {
		s.DeclaredAddr = c.Run.DeclAddr
		s.HttpAddr = c.Run.HttpAddr
		s.WavesNetwork = c.Run.WavesNetwork
		s.Addresses = c.Run.Addresses
	}
}
