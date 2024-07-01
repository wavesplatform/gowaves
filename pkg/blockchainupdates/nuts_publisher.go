package blockchainupdates

import (
	"context"
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
)

// protoc --go_out=. --go_opt=paths=source_relative blockchainupdates.proto

type BUpdatesInfo struct {
	Height         uint64
	VRF            proto.B58Bytes
	BlockID        proto.BlockID
	BlockHeader    *proto.BlockHeader
	AllDataEntries []proto.DataEntry
}

type BUpdatesExtensionState struct {
	currentState  *BUpdatesInfo
	previousState *BUpdatesInfo // this information is what was just published
	Limit         uint64
}

func NewBUpdatesExtensionState(limit uint64) *BUpdatesExtensionState {
	return &BUpdatesExtensionState{Limit: limit}
}

func (bu *BUpdatesExtensionState) hasStateChanged() {

}

func (bu *BUpdatesExtensionState) RunBlockchainUpdatesPublisher(ctx context.Context, updatesChannel <-chan BUpdatesInfo) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: 4222,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("failed to create NATS server: %v", err)
	}
	go s.Start()
	if !s.ReadyForConnections(5 * server.AUTH_TIMEOUT) {
		log.Fatal("NATS Server not ready for connections")
	}
	log.Printf("NATS Server is running on port %d", 4222)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	func(ctx context.Context, updatesChannel <-chan BUpdatesInfo) {
		for {
			select {
			case updates, ok := <-updatesChannel:
				if !ok {
					log.Printf("the updates channel for publisher was closed")
					return
				}
				// update current state

				// compare the current state to the previous state

				// if there is any diff, send the update

				fmt.Println(updates.Height)
				var msg string
				msg = "hello"
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
}
