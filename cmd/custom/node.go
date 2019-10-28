package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/api"
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
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 1, Patch: 2}

type Cli struct {
	Run struct {
		WavesNetwork string `kong:"wavesnetwork,short='n',help='Waves network.',required"`
		Addresses    string `kong:"address,short='a',help='Addresses connect to.'"`
		DeclAddr     string `kong:"decladdr,short='d',help='Address listen on.'"`
		Http         string `kong:"http addr,help='Http addr bind on.'"`
		GenesisPath  string `kong:"genesis,short='g',help='Path to genesis json file.'"`
		Seed         string `kong:"seed,help='Seed for miner.'"`
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

	custom := &settings.BlockchainSettings{
		Type: 'E',
		FunctionalitySettings: settings.FunctionalitySettings{
			FeaturesVotingPeriod:   10000,
			MaxTxTimeBackOffset:    120 * 60000,
			MaxTxTimeForwardOffset: 90 * 60000,

			AddressSchemeCharacter: 'E',

			AverageBlockDelaySeconds: 60,
			MaxBaseTarget:            200,
		},
		GenesisGetter: settings.FromPath(cli.Run.GenesisPath),
	}

	state, err := state.NewState("./", state.DefaultStateParams(), custom)
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
	if len(cli.Run.Seed) > 0 {
		keyPairs = append(keyPairs, proto.NewKeyPair([]byte(cli.Run.Seed)))
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

	Mainer := miner.NewMicroblockMiner(services, ngRuntime, 'E')

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

func FromArgs(c *Cli) func(s *settings.NodeSettings) {
	return func(s *settings.NodeSettings) {
		s.DeclaredAddr = c.Run.DeclAddr
		s.HttpAddr = c.Run.Http
		s.WavesNetwork = c.Run.WavesNetwork
		s.Addresses = c.Run.Addresses
	}
}
