package main

import (
	"context"
	"github.com/alecthomas/kong"

	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"strings"
)

type Cli struct {
	Run struct {
		WavesNetwork string `kong:"wavesnetwork,short='n',help='Waves network.',required"`
		Addresses    string `kong:"address,short='a',help='Addresses connect to.'"`
		Version      string `kong:"version,short='v',help='Version,(0.15.1).',required"`
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
	ctx := context.Background()

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

	pool := bytespool.NewBytesPool(64, 2*1024*2014)

	//sig, _ := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")
	//
	//zap.S().Info(state.Height())
	//zap.S().Info(state.BlockByHeight(1))
	//zap.S().Info(state.Block(sig))

	parent := peer.NewParent()

	peerSpawnerimpl := node.NewPeerSpawner(pool, noSkip, parent, cli.Run.WavesNetwork, proto.PeerInfo{}, "gowaves", 100500)

	peerManager := node.NewPeerManager(peerSpawnerimpl)

	n := node.NewNode(state, peerManager)

	go node.RunNode(ctx, n, parent)

	if len(cli.Run.Addresses) > 0 {
		adrs := strings.Split(cli.Run.Addresses, ",")
		for _, addr := range adrs {
			peerManager.AddAddress(ctx, addr)
		}
	}

	select {}

}
