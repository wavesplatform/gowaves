package main

import (
	"context"
	"github.com/alecthomas/kong"
	"github.com/wavesplatform/gowaves/pkg/api"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"strings"
)

var version = proto.Version{0, 15, 1}

type Cli struct {
	Run struct {
		WavesNetwork string `kong:"wavesnetwork,short='n',help='Waves network.',required"`
		Addresses    string `kong:"address,short='a',help='Addresses connect to.'"`
		//Version      string `kong:"version,short='v',help='Version,(0.15.1).',required"`
	} `kong:"cmd,help='Run node'"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func noSkip(_ proto.Header) bool {
	return false
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var cli Cli
	kong.Parse(&cli)

	state, err := state.NewState("./", state.DefaultBlockStorageParams())
	if err != nil {
		zap.S().Error(err)
		return
	}

	switch cli.Run.WavesNetwork {
	case "wavesW", "wavesD", "wavesT":
	default:
		zap.S().Error("expected WavesNetwork to be wavesW, wavesD or wavesT, found %s", cli.Run.WavesNetwork)
		return
	}

	//pool := bytespool.NewBytesPool(64, 2*1024*2014)
	pool := bytespool.NewNoOpBytesPool(2 * 1024 * 2014)

	//sig, _ := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	//
	//zap.S().Info(state.Height())
	//zap.S().Info(state.BlockByHeight(1))
	//zap.S().Info(state.Block(sig))

	parent := peer.NewParent()

	peerSpawnerimpl := node.NewPeerSpawner(pool, noSkip, parent, cli.Run.WavesNetwork, proto.PeerInfo{}, "gowaves", 100500, version)

	peerManager := node.NewPeerManager(peerSpawnerimpl, state)

	n := node.NewNode(state, peerManager)

	go node.RunNode(ctx, n, parent)

	if len(cli.Run.Addresses) > 0 {
		adrs := strings.Split(cli.Run.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	webApi := api.NewNodeApi(state)
	go func() {
		err := api.Run(ctx, "0.0.0.0:8085", webApi)
		if err != nil {
			zap.S().Error("Failed to start API: %v", err)
		}
	}()

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		//if memprofile != "" {
		//	memProfile(memprofile)
		//}
		n.Close()

		zap.S().Infow("Caught signal, stopping", "signal", sig)
		//_ = srv.Shutdown(ctx)
		cancel()

		<-time.After(2 * time.Second)
	}

	//select {}

}
