package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/afero"
	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/httpserver"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

func cpuProfile(filename string) func() {
	cleanFilename := filepath.Clean(filename)
	f, err := os.Create(cleanFilename)
	if err != nil {
		zap.S().Fatal(err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		zap.S().Fatal(err)
	}
	return pprof.StopCPUProfile
}

func memProfile(filename string) {
	cleanFilename := filepath.Clean(filename)
	f, err := os.Create(cleanFilename)
	if err != nil {
		zap.S().Fatal(err)
	}
	if err := pprof.WriteHeapProfile(f); err != nil {
		zap.S().Fatal(err)
	}
	_ = f.Close()
}

func skipUselessMessages(header proto.Header) bool {
	switch header.ContentID {
	case proto.ContentIDTransaction, proto.ContentIDPeers, proto.ContentIDGetPeers:
		return false
	default:
		return true
	}
}

var defaultPeers = map[string]string{
	"wavesW": "35.156.19.4:6868,52.50.69.247:6868,52.52.46.76:6868,52.57.147.71:6868,52.214.55.18:6868,54.176.190.226:6868",
	"wavesT": "52.51.92.182:6863,52.231.205.53:6863,52.30.47.67:6863,52.28.66.217:6863",
	"wavesS": "217.100.219.251:6861",
}

var schemes = map[string]byte{
	"wavesW": proto.MainNetScheme,
	"wavesT": proto.TestNetScheme,
	"wavesS": proto.StageNetScheme,
}

func main() {
	// delay before exit
	defer func() {
		<-time.After(1 * time.Second)
	}()

	var err error

	cfg := zap.NewDevelopmentConfig()
	logger, err := cfg.Build()
	if err != nil {
		fmt.Println(err)
		return
	}
	zap.ReplaceGlobals(logger)

	var bind string
	var decl string
	var addresses string
	var wavesNetwork string
	var cpuprofile string
	var memprofile string
	flag.StringVarP(&bind, "bind", "b", "", "Local address listen on")
	flag.StringVarP(&decl, "decl", "d", "", "Declared Address")
	flag.StringVarP(&addresses, "addresses", "a", "", "Addresses connect to")
	flag.StringVarP(&wavesNetwork, "wavesnetwork", "n", "", "Required, waves network, should be wavesW or wavesT or wavesD")
	flag.StringVarP(&cpuprofile, "cpuprofile", "", "", "write cpu profile to file")
	flag.StringVarP(&memprofile, "memprofile", "", "", "write memory profile to this file")
	flag.Parse()

	if cpuprofile != "" {
		defer cpuProfile(cpuprofile)()
	}

	switch wavesNetwork {
	case "wavesW", "wavesT", "wavesS":
	default:
		zap.S().Errorf("expected waves network to be wavesW or wavesT or wavesD, found %s", wavesNetwork)
		return
	}

	declAddr := proto.TCPAddr{}
	if decl != "" {
		declAddr = proto.NewTCPAddrFromString(decl)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fs := afero.NewOsFs()

	storage, err := utils.NewFileBasedStorage(fs, "known_peers.json")
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	knownPeers, err := utils.NewKnownPeers(storage)
	if err != nil {
		zap.S().Error(err)
		cancel()
		return
	}

	parent := peer.NewParent()
	spawner := retransmit.NewPeerSpawner(skipUselessMessages, parent, wavesNetwork, declAddr)
	scheme := schemes[wavesNetwork]
	behaviour := retransmit.NewBehaviour(knownPeers, spawner, scheme)
	r := retransmit.NewRetransmitter(behaviour, parent)
	r.Run(ctx)

	if addresses == "" {
		addresses = defaultPeers[wavesNetwork]
	}

	for _, a := range strings.Split(addresses, ",") {
		a = strings.Trim(a, " ")
		if a != "" {
			r.Address(ctx, a)
		}
	}

	if bind != "" {
		err = r.ServeIncomingConnections(ctx, bind)
		if err != nil {
			zap.S().Error(err)
			cancel()
			return
		}
	}

	srv := httpserver.NewHttpServer(behaviour)

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			zap.S().Error(err)
		}
	}()

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	sig := <-gracefulStop
	if memprofile != "" {
		memProfile(memprofile)
	}
	zap.S().Infof("Caught signal '%s', stopping...", sig)
	_ = srv.Shutdown(ctx)
	cancel()
}
