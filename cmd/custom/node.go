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
	"github.com/wavesplatform/gowaves/pkg/libs/microblock_cache"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/libs/runner"
	"github.com/wavesplatform/gowaves/pkg/miner"
	scheduler2 "github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/node/blocks_applier"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/node/peer_manager"
	peersPersistentStorage "github.com/wavesplatform/gowaves/pkg/node/peer_manager/storage"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/ng"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/util/fdlimit"
	"github.com/wavesplatform/gowaves/pkg/wallet"
	"go.uber.org/zap"
)

var version = proto.Version{Major: 1, Minor: 3, Patch: 0}

var (
	logLevel          = flag.String("log-level", "INFO", "Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
	statePath         = flag.String("state-path", "", "Path to node's state directory")
	peerAddresses     = flag.String("peers", "", "Addresses of peers to connect to")
	declAddr          = flag.String("declared-address", "", "Address to listen on")
	apiAddr           = flag.String("api-address", "", "Address for REST API")
	grpcAddr          = flag.String("grpc-address", "127.0.0.1:7475", "Address for gRPC API")
	cfgPath           = flag.String("cfg-path", "", "Path to configuration JSON file. No default value.")
	enableGrpcApi     = flag.Bool("enable-grpc-api", false, "Enables/disables gRPC API")
	buildExtendedApi  = flag.Bool("build-extended-api", false, "Builds extended API. Note that state must be re-imported in case it wasn't imported with similar flag set")
	serveExtendedApi  = flag.Bool("serve-extended-api", false, "Serves extended API requests since the very beginning. The default behavior is to import until first block close to current time, and start serving at this point")
	buildStateHashes  = flag.Bool("build-state-hashes", false, "Calculate and store state hashes for each block height.")
	minerVoteFeatures = flag.String("vote", "", "Miner vote features")
	reward            = flag.String("reward", "", "Miner reward: for example 600000000")
	outdateS          = flag.String("outdate", "4h", "Interval between last applied block and current time. If greater than no mining, no transaction accepted. Example 1d4h30m")
	walletPath        = flag.String("wallet-path", "", "Path to wallet, or ~/.waves by default")
	walletPassword    = flag.String("wallet-password", "", "Pass password for wallet. Extremely insecure")
	limitConnectionsS = flag.String("limit-connections", "30", "N incoming and outgoing connections")
	minPeersMining    = flag.Int("min-peers-mining", 1, "Minimum connected peers for allow mining")
	dropPeers         = flag.Bool("drop-peers", false, "Drop peers storage before node start.")
)

func init() {
	common.SetupLogger(*logLevel)
}

func main() {
	flag.Parse()

	if *cfgPath == "" {
		zap.S().Fatalf("Please provide path to blockchain config JSON file")
	}

	maxFDs, err := fdlimit.MaxFDs()
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}
	_, err = fdlimit.RaiseMaxFDs(maxFDs)
	if err != nil {
		zap.S().Fatalf("Initialization error: %v", err)
	}

	common.SetupLogger(*logLevel)

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

	ntpTime, err := ntptime.TryNew("pool.ntp.org", 10)
	if err != nil {
		zap.S().Error(err)
		return
	}

	outdateSeconds, err := common.ParseDuration(*outdateS)
	if err != nil {
		zap.S().Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	go ntpTime.Run(ctx, 2*time.Minute)

	params := state.DefaultStateParams()
	params.StoreExtendedApiData = *buildExtendedApi
	params.ProvideExtendedApi = *serveExtendedApi
	params.BuildStateHashes = *buildStateHashes
	params.Time = ntpTime
	nodeState, err := state.NewState(path, params, custom)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	// Check if we need to start serving extended API right now.
	if err := node.MaybeEnableExtendedApi(nodeState, ntpTime); err != nil {
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

	features, err = miner.ValidateFeatures(nodeState, features)
	if err != nil {
		cancel()
		zap.S().Error(err)
		return
	}

	declAddr := proto.NewTCPAddrFromString(conf.DeclaredAddr)

	parent := peer.NewParent()
	utx := utxpool.New(10000, utxpool.NewValidator(nodeState, ntpTime, outdateSeconds*1000), custom)
	peerSpawnerImpl := peer_manager.NewPeerSpawner(parent, conf.WavesNetwork, declAddr, "gowaves", uint64(rand.Int()), version)

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

	peerManager := peer_manager.NewPeerManager(peerSpawnerImpl, peerStorage, int(limitConnections), version, conf.WavesNetwork, true, 10)
	go peerManager.Run(ctx)

	scheduler := scheduler2.NewScheduler(
		nodeState,
		wal,
		custom,
		ntpTime,
		scheduler2.NewMinerConsensus(peerManager, *minPeersMining),
		proto.NewTimestampFromUSeconds(outdateSeconds),
	)

	blockApplier := blocks_applier.NewBlocksApplier()

	async := runner.NewAsync()
	logRunner := runner.NewLogRunner(async)

	InternalCh := messages.NewInternalChannel()

	var nodeServices = services.Services{
		State:           nodeState,
		Peers:           peerManager,
		Scheduler:       scheduler,
		BlocksApplier:   blockApplier,
		UtxPool:         utx,
		Scheme:          custom.AddressSchemeCharacter,
		InvRequester:    ng.NewInvRequester(),
		LoggableRunner:  logRunner,
		MicroBlockCache: microblock_cache.NewMicroblockCache(),
		InternalChannel: InternalCh,
		Time:            ntpTime,
	}
	Miner := miner.NewMicroblockMiner(nodeServices, features, reward, proto.NewTimestampFromUSeconds(outdateSeconds))
	go miner.Run(ctx, Miner, scheduler, InternalCh)

	n := node.NewNode(nodeServices, declAddr, declAddr, proto.NewTimestampFromUSeconds(outdateSeconds))

	go n.Run(ctx, parent, InternalCh)

	if len(conf.Addresses) > 0 {
		addresses := strings.Split(conf.Addresses, ",")
		for _, addr := range addresses {
			tcpAddr := proto.NewTCPAddrFromString(addr)
			if tcpAddr.Empty() {
				// nickeskov: that means that configuration parameter is invalid
				zap.S().Errorf("Failed to parse TCPAddr from string %q", tcpAddr.String())
				cancel()
				return
			}
			if err := peerManager.AddAddress(ctx, tcpAddr); err != nil {
				// nickeskov: than means that we have problems with peers storage
				zap.S().Errorf("Failed to add addres into know peers storage: %v", err)
				cancel()
				return
			}
		}
	}

	// TODO hardcore
	app, err := api.NewApp("integration-test-rest-api", scheduler, nodeServices)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	webApi := api.NewNodeApi(app, nodeState, n)
	go func() {
		zap.S().Info("===== ", conf.HttpAddr)
		err := api.Run(ctx, conf.HttpAddr, webApi)
		if err != nil {
			zap.S().Errorf("Failed to start API: %v", err)
		}
	}()

	if *enableGrpcApi {
		grpcServer, err := server.NewServer(nodeServices)
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
		return nil
	}
}
