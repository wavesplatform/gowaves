package blockchaininfo

import (
	"context"
	"log"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const StoreBlocksLimit = 2000
const PortDefault = 4222
const HostDefault = "127.0.0.1"
const ConnectionsTimeoutDefault = 5 * server.AUTH_TIMEOUT

type BUpdatesExtensionState struct {
	currentState  *BUpdatesInfo
	previousState *BUpdatesInfo // this information is what was just published
	Limit         uint64
	scheme        proto.Scheme
}

func NewBUpdatesExtensionState(limit uint64, scheme proto.Scheme) *BUpdatesExtensionState {
	return &BUpdatesExtensionState{Limit: limit, scheme: scheme}
}

func (bu *BUpdatesExtensionState) hasStateChanged() (bool, error) {
	statesAreEqual, err := statesEqual(*bu, bu.scheme)
	if err != nil {
		return false, err
	}
	if statesAreEqual {
		return false, nil
	}
	return true, nil
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
	log.Printf("Published on topic: %s\n", BlockUpdates)

	/* second publish contract data entries */
	dataEntries := L2ContractDataEntriesToProto(updates.ContractUpdatesInfo.AllDataEntries)
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
		log.Printf("Published on topic: %s\n", ContractUpdates)
	}

	return nil
}

func handleBlockchainUpdates(updates BUpdatesInfo, ok bool,
	bu *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn) {
	if !ok {
		log.Printf("the updates channel for publisher was closed")
		return
	}
	// update current state
	bu.currentState = &updates
	// compare the current state to the previous state
	stateChanged, cmprErr := bu.hasStateChanged()
	if cmprErr != nil {
		log.Printf("failed to compare current and previous states, %v", cmprErr)
		return
	}
	// if there is any diff, send the update
	if stateChanged {
		pblshErr := bu.publishUpdates(updates, nc, scheme)
		log.Printf("published")
		if pblshErr != nil {
			log.Printf("failed to publish updates, %v", pblshErr)
		}
		bu.previousState = &updates
	}
}

func runPublisher(ctx context.Context, updatesChannel <-chan BUpdatesInfo,
	bu *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn) {
	func(ctx context.Context, updatesChannel <-chan BUpdatesInfo) {
		for {
			select {
			case updates, ok := <-updatesChannel:
				handleBlockchainUpdates(updates, ok, bu, scheme, nc)

			case <-ctx.Done():
				return
			}
		}
	}(ctx, updatesChannel)
}

func (bu *BUpdatesExtensionState) RunBlockchainUpdatesPublisher(ctx context.Context,
	updatesChannel <-chan BUpdatesInfo, scheme proto.Scheme) {
	opts := &server.Options{
		Host: HostDefault,
		Port: PortDefault,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("failed to create NATS server: %v", err)
	}
	go s.Start()
	if !s.ReadyForConnections(ConnectionsTimeoutDefault) {
		log.Fatal("NATS Server not ready for connections")
	}
	log.Printf("NATS Server is running on port %d", PortDefault)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	runPublisher(ctx, updatesChannel, bu, scheme, nc)
}
