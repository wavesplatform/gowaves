package blockchainupdates

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func runBlockchainUpdatesPublisher(updatesChannel chan interface{}) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	func(ctx context.Context, updatesChannel chan interface{}) {
		for {
			select {
			case <-updatesChannel:
				var msg string
				// Publish blockchain updates
				topic := block_updates
				err := nc.Publish(topic, []byte(msg))
				if err != nil {
					log.Printf("failed to publish message %s on topic %s", msg, topic)
				}
				fmt.Printf("Published: %s\n", msg)

			case <-ctx.Done():
				return
			}
		}
	}(ctx, updatesChannel)
	<-ctx.Done()

}
