package blockchaininfo

import (
	"context"
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
)

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

func (bu *BUpdatesExtensionState) publishUpdates(updates BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
	/* first publish block related info */
	blockInfo, err := BUpdatesInfoToProto(updates, scheme)
	if err != nil {
		return err
	}
	blockInfoProtobuf, err := blockInfo.MarshalVTStrict()
	if err != nil {
		return err
	}
	err = nc.Publish(BlockUpdates, blockInfoProtobuf)
	if err != nil {
		log.Printf("failed to publish message on topic %s", BlockUpdates)
		return err
	}
	fmt.Printf("Published on topic: %s\n", BlockUpdates)

	/* second publish contract data entries */
	dataEntries := L2ContractDataEntriesToProto(updates.AllDataEntries)
	dataEntriesProtobuf, err := dataEntries.MarshalVTStrict()
	if err != nil {
		return err
	}
	if dataEntries.DataEntries != nil {
		err = nc.Publish(ContractUpdates, dataEntriesProtobuf)
		if err != nil {
			log.Printf("failed to publish message on topic %s", ContractUpdates)
			return err
		}
		fmt.Printf("Published on topic: %s\n", ContractUpdates)
	}

	return nil
}

func (bu *BUpdatesExtensionState) RunBlockchainUpdatesPublisher(ctx context.Context, updatesChannel <-chan BUpdatesInfo, scheme proto.Scheme) {
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
				err := bu.publishUpdates(updates, nc, scheme)
				log.Printf("published")
				if err != nil {
					log.Printf("failed to publish updates")
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, updatesChannel)
}
