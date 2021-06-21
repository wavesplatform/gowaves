package metamask

import (
	"context"
	"github.com/semrush/zenrpc/v2"
	"log"
	"net/http"
	"os"
)

func RunMetaMaskService(ctx context.Context, address string) error {
	rpc := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true})
	rpc.Register("", MetaMask{}) // public
	rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))

	http.Handle("/", rpc)

	server := &http.Server{Addr: address, Handler: nil}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("failed to start metamask service")
		}
	}()
	server.Shutdown(ctx)
	return nil
}
