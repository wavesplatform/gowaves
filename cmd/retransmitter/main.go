package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/network/retransmit"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// filter only transactions and
func receiveFromRemoteCallbackfunc(b []byte, id string, resendTo chan peer.ProtoMessage, pool conn.Pool) {

	defer func() {
		pool.Put(b)
	}()

	zap.S().Debugf("receiveFromRemoteCallbackfunc, len bytes %d", len(b))

	if len(b) < 9 {
		return
	}

	switch b[8] {
	case proto.ContentIDTransaction:

		m := &proto.TransactionMessage{}
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
		}

	case proto.ContentIDPeers:
		fmt.Println("got proto.ContentIDPeers message", id)

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
		}
	default:
		fmt.Println("bytes id ", b[8])
		return
	}
}

func main() {

	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	ctx, cancel := context.WithCancel(context.Background())

	pool := bytespool.NewBytesPool(32, 2*1024*1024)

	r := retransmit.NewRetransmitter(ctx, peer.RunOutgoingPeer, peer.RunIncomingPeer, receiveFromRemoteCallbackfunc, pool)

	go r.Run()

	//r.AddAddress("34.253.153.4:6868")
	//r.AddAddress("52.214.55.18:6868")
	r.AddAddress("mainnet-aws-ca-1.wavesnodes.com:6868")

	httpServer := retransmit.NewHttpServer(r)

	router := mux.NewRouter()
	router.HandleFunc("/active", httpServer.ActiveConnections)
	router.HandleFunc("/known", httpServer.KnownPeers)
	http.Handle("/", router)

	srv := http.Server{
		Handler: router,
		Addr:    "127.0.0.1:8000",
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			zap.S().Error(err)
		}
	}()

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		_ = srv.Shutdown(ctx)
		cancel()
		zap.S().Infow("Caught signal, stopping", "signal", sig)
	}
}
