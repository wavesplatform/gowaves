package blockchaininfo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const StoreBlocksLimit = 2000
const PortDefault = 4222
const HostDefault = "127.0.0.1"
const ConnectionsTimeoutDefault = 5 * server.AUTH_TIMEOUT

const NATS_MAX_PAYLOAD_SIZE int32 = (1024 * 1024) * 64 // 64 MB

//const NATS_MAX_PAYLOAD_SIZE int32 = 5

const (
	START_PAGING = iota
	END_PAGING
	NO_PAGING
)

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

func splitIntoChunks(array []byte, maxChunkSize int) [][]byte {
	if maxChunkSize <= 0 {
		return nil
	}
	var chunkedArray [][]byte

	for i := 0; i < len(array); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(array) {
			end = len(array)
		}
		chunkedArray = append(chunkedArray, array[i:end])
	}

	return chunkedArray
}

func (bu *BUpdatesExtensionState) publishContractUpdates(contractUpdates L2ContractDataEntries, nc *nats.Conn, scheme proto.Scheme) error {
	dataEntriesProtobuf, err := L2ContractDataEntriesToProto(contractUpdates).MarshalVTStrict()
	if err != nil {
		return err
	}
	fmt.Println("Size in MB")

	fmt.Println(len(dataEntriesProtobuf) / 1024)

	if len(dataEntriesProtobuf) <= int(NATS_MAX_PAYLOAD_SIZE-1) {
		var msg []byte
		msg = append(msg, NO_PAGING)
		msg = append(msg, dataEntriesProtobuf...)
		err = nc.Publish(ContractUpdates, msg)
		if err != nil {
			log.Printf("failed to publish message on topic %s", ContractUpdates)
			return err
		}
		log.Printf("Published on topic: %s\n", ContractUpdates)
		return nil
	}

	chunkedPayload := splitIntoChunks(dataEntriesProtobuf, int(NATS_MAX_PAYLOAD_SIZE-1))

	for i, chunk := range chunkedPayload {
		var msg []byte

		if i == len(chunkedPayload)-1 {
			msg = append(msg, END_PAGING)
			msg = append(msg, chunk...)
			err = nc.Publish(ContractUpdates, msg)
			if err != nil {
				log.Printf("failed to publish message on topic %s", ContractUpdates)
				return err
			}
			log.Printf("Published on topic: %s\n", ContractUpdates)
			break
		}
		msg = append(msg, START_PAGING)
		msg = append(msg, chunk...)
		err = nc.Publish(ContractUpdates, msg)
		if err != nil {
			log.Printf("failed to publish message on topic %s", ContractUpdates)
			return err
		}
		log.Printf("Published on topic: %s\n", ContractUpdates)
	}

	return nil
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
	// temporary
	prettyJSON, err := json.MarshalIndent(updates.ContractUpdatesInfo, "", "    ")
	if err != nil {
		fmt.Println("Error converting to pretty JSON:", err)
		return err
	}
	heightStr := strconv.Itoa(int(updates.BlockUpdatesInfo.Height))
	// Write the pretty JSON to a file named "index.json"
	err = os.WriteFile("/media/alex/My_Book/dolgavin/waves/contract_data/"+heightStr+".json", prettyJSON, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	//
	err = nc.Publish(BlockUpdates, blockInfoProtobuf)
	if err != nil {
		log.Printf("failed to publish message on topic %s", BlockUpdates)
		return err
	}
	log.Printf("Published on topic: %s\n", BlockUpdates)

	/* second publish contract data entries */
	if updates.ContractUpdatesInfo.AllDataEntries != nil {
		//fmt.Println("Size in MB")
		//fmt.Println(len(dataEntriesProtobuf) / 1024)
		pblshErr := bu.publishContractUpdates(updates.ContractUpdatesInfo, nc, scheme)
		if pblshErr != nil {
			log.Printf("failed to publish message on topic %s", ContractUpdates)
			return pblshErr
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
		MaxPayload: NATS_MAX_PAYLOAD_SIZE,
		Host:       HostDefault,
		Port:       PortDefault,
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
