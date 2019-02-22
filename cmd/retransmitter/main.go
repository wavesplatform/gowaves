package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/afero"
	flag "github.com/spf13/pflag"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/httpserver"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

// filter transactions, ContentIDPeers, ContentIDGetPeers
func receiveFromRemoteCallbackFunc(b []byte, id string, resendTo chan peer.ProtoMessage, pool conn.Pool) {
	defer func() {
		pool.Put(b)
	}()

	if len(b) < 9 {
		return
	}

	switch b[8] {
	case proto.ContentIDTransaction:
		m := &proto.TransactionMessage{}
		err := m.UnmarshalBinary(b)
		if err != nil {
			zap.S().Error(err, id, b)
			return
		}

		zap.S().Debugf("transaction from %s", id)

		mess := peer.ProtoMessage{
			ID:      id,
			Message: m,
		}

		select {
		case resendTo <- mess:
		default:
			zap.S().Warnf("failed to resend to parent, channel is full", id)
		}

	case proto.ContentIDPeers:
		m := &proto.PeersMessage{}
		err := m.UnmarshalBinary(b)
		if err != nil {
			fmt.Println(err)
			return
		}

		mess := peer.ProtoMessage{
			ID:      id,
			Message: m,
		}

		select {
		case resendTo <- mess:
		default:
			zap.S().Warnf("failed to resend to parent, channel is full", id)
		}

	case proto.ContentIDGetPeers:
		fmt.Println("retransmitter got proto.ContentIDGetPeers message from ", id)
		m := &proto.GetPeersMessage{}
		err := m.UnmarshalBinary(b)
		if err != nil {
			fmt.Println(err)
			return
		}

		mess := peer.ProtoMessage{
			ID:      id,
			Message: m,
		}

		select {
		case resendTo <- mess:
		default:
			zap.S().Warnf("failed to resend to parent, channel is full", id)
		}
	default:
		return
	}
}

func cpuProfile(filename string) func() {
	f, err := os.Create(filename)
	if err != nil {
		zap.S().Fatal(err)
	}
	pprof.StartCPUProfile(f)
	return pprof.StopCPUProfile
}

func memProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		zap.S().Fatal(err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
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
	case "wavesW", "wavesT", "wavesD":
	default:
		zap.S().Errorf("expected waves network to be wavesW or wavesT or wavesD, found %s", wavesNetwork)
		return
	}

	declAddr := proto.PeerInfo{}
	if decl != "" {
		declAddr, err = proto.NewPeerInfoFromString(decl)
		if err != nil {
			fmt.Println(err)
			return
		}
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

	pool := bytespool.NewBytesPool(96, 2*1024*1024)

	parent := peer.NewParent()

	spawner := retransmit.NewPeerSpawner(pool, receiveFromRemoteCallbackFunc, parent, wavesNetwork, declAddr)

	behaviour := retransmit.NewBehaviour(knownPeers, spawner)

	r := retransmit.NewRetransmitter(behaviour, parent)

	go r.Run(ctx)

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

	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
				allocations, puts, gets := pool.Stat()
				zap.S().Info("allocations: ", allocations, " puts: ", puts, " gets: ", gets)
			case <-ctx.Done():
				return
			}
		}
	}()

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		if memprofile != "" {
			memProfile(memprofile)
		}
		zap.S().Infow("Caught signal, stopping", "signal", sig)
		_ = srv.Shutdown(ctx)
		cancel()
	}
}
