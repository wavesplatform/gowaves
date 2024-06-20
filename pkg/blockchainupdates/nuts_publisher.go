package blockchainupdates

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type BUpdatesInfo struct {
	Height      uint64
	VRF         proto.B58Bytes
	BlockID     proto.BlockID
	BlockHeader *proto.BlockHeader
}

type BUpdatesExtensionState struct {
	currentState  *BUpdatesInfo
	previousState *BUpdatesInfo // this information is what was just published
}

func NewBUpdatesExtensionState() *BUpdatesExtensionState {
	return &BUpdatesExtensionState{}
}

func (bu *BUpdatesExtensionState) hasStateChanged() {

}

func (bu *BUpdatesExtensionState) RunBlockchainUpdatesPublisher(updatesChannel <-chan BUpdatesInfo) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	func(ctx context.Context, updatesChannel <-chan BUpdatesInfo) {
		for {
			select {
			case updates := <-updatesChannel:
				// update current state

				// compare the current state to the previous state

				// if there is any diff, send the update

				fmt.Println(updates.Height)
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
