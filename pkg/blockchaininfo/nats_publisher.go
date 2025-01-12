package blockchaininfo

import (
	"context"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const StoreBlocksLimit = 200
const PortDefault = 4222
const HostDefault = "127.0.0.1"
const ConnectionsTimeoutDefault = 5 * server.AUTH_TIMEOUT

const NatsMaxPayloadSize int32 = 1024 * 1024 // 1 MB

const PublisherWaitingTime = 100 * time.Millisecond

const (
	StartPaging = iota
	EndPaging
	NoPaging
)

const L2RequestsTopic = "l2_requests_topic"
const (
	RequestRestartSubTopic = "restart"
)

func ConcatenateContractTopics(contractAddress string) string {
	return ContractUpdates + contractAddress
}

type BUpdatesExtensionState struct {
	currentState      *BUpdatesInfo
	previousState     *BUpdatesInfo // this information is what was just published
	Limit             uint64
	scheme            proto.Scheme
	l2ContractAddress string
}

func NewBUpdatesExtensionState(limit uint64, scheme proto.Scheme, l2ContractAddress string) *BUpdatesExtensionState {
	return &BUpdatesExtensionState{Limit: limit, scheme: scheme, l2ContractAddress: l2ContractAddress}
}

func (bu *BUpdatesExtensionState) hasStateChanged() (bool, BUpdatesInfo, error) {
	statesAreEqual, changes, err := statesEqual(*bu, bu.scheme)
	if err != nil {
		return false, BUpdatesInfo{}, err
	}
	if statesAreEqual {
		return false, BUpdatesInfo{}, nil
	}
	return true, changes, nil
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

func (bu *BUpdatesExtensionState) publishContractUpdates(contractUpdates L2ContractDataEntries, nc *nats.Conn) error {
	dataEntriesProtobuf, err := L2ContractDataEntriesToProto(contractUpdates).MarshalVTStrict()
	if err != nil {
		return err
	}

	if len(dataEntriesProtobuf) <= int(NatsMaxPayloadSize-1) {
		var msg []byte
		msg = append(msg, NoPaging)
		msg = append(msg, dataEntriesProtobuf...)
		err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
		if err != nil {
			log.Printf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return err
		}
		log.Printf("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
		return nil
	}

	chunkedPayload := splitIntoChunks(dataEntriesProtobuf, int(NatsMaxPayloadSize-1)/2)

	for i, chunk := range chunkedPayload {
		var msg []byte

		if i == len(chunkedPayload)-1 {
			msg = append(msg, EndPaging)
			msg = append(msg, chunk...)
			err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
			if err != nil {
				log.Printf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
				return err
			}
			log.Printf("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
			break
		}
		msg = append(msg, StartPaging)
		msg = append(msg, chunk...)
		err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
		if err != nil {
			log.Printf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return err
		}
		log.Printf("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
		time.Sleep(PublisherWaitingTime)
	}

	return nil
}

func (bu *BUpdatesExtensionState) publishBlockUpdates(updates BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
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
	return nil
}

func (bu *BUpdatesExtensionState) publishUpdates(updates BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
	/* first publish block data */
	err := bu.publishBlockUpdates(updates, nc, scheme)
	if err != nil {
		log.Printf("failed to publish message on topic %s", BlockUpdates)
		return err
	}

	/* second publish contract data entries */
	if updates.ContractUpdatesInfo.AllDataEntries != nil {
		pblshErr := bu.publishContractUpdates(updates.ContractUpdatesInfo, nc)
		if pblshErr != nil {
			log.Printf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return pblshErr
		}
		log.Printf("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
	}

	return nil
}

func handleBlockchainUpdate(updates BUpdatesInfo, bu *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn) {
	// update current state
	bu.currentState = &updates
	if bu.previousState == nil {
		// publish initial updates
		filteredDataEntries, err := filterDataEntries(updates.BlockUpdatesInfo.Height-bu.Limit,
			updates.ContractUpdatesInfo.AllDataEntries)
		if err != nil {
			return
		}
		updates.ContractUpdatesInfo.AllDataEntries = filteredDataEntries
		pblshErr := bu.publishUpdates(updates, nc, scheme)
		if pblshErr != nil {
			log.Printf("failed to publish updates, %v", pblshErr)
			return
		}
		bu.previousState = &updates
		return
	}
	// compare the current state to the previous state
	stateChanged, changes, cmprErr := bu.hasStateChanged()
	if cmprErr != nil {
		log.Printf("failed to compare current and previous states, %v", cmprErr)
		return
	}
	// if there is any diff, send the update
	if stateChanged {
		pblshErr := bu.publishUpdates(changes, nc, scheme)
		log.Printf("published changes")
		if pblshErr != nil {
			log.Printf("failed to publish changes, %v", pblshErr)
		}
		bu.previousState = &updates
	}
}

func runPublisher(ctx context.Context, updatesChannel <-chan BUpdatesInfo,
	bu *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn) {
	for {
		select {
		case updates, ok := <-updatesChannel:
			if !ok {
				log.Printf("the updates channel for publisher was closed")
				return
			}
			handleBlockchainUpdate(updates, bu, scheme, nc)
		case <-ctx.Done():
			return
		}
	}
}

func runReceiver(requestsChannel chan<- L2Requests, nc *nats.Conn) error {
	_, subErr := nc.Subscribe(L2RequestsTopic, func(request *nats.Msg) {
		signal := string(request.Data)
		switch signal {
		case RequestRestartSubTopic:
			l2Request := L2Requests{Restart: true}
			requestsChannel <- l2Request

			notNilResponse := "ok"
			err := request.Respond([]byte(notNilResponse))
			if err != nil {
				return
			}
		default:
			zap.S().Errorf("nats receiver received an unknown signal, %s", signal)
		}
	})
	if subErr != nil {
		return subErr
	}
	return nil
}

func (bu *BUpdatesExtensionState) RunBlockchainUpdatesPublisher(ctx context.Context,
	updatesChannel <-chan BUpdatesInfo, scheme proto.Scheme, l2Requests chan<- L2Requests) error {
	opts := &server.Options{
		MaxPayload: NatsMaxPayloadSize,
		Host:       HostDefault,
		Port:       PortDefault,
		NoSigs:     true,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		return errors.Wrap(err, "failed to create NATS server")
	}
	go s.Start()
	defer func() {
		s.Shutdown()
		s.WaitForShutdown()
	}()
	if !s.ReadyForConnections(ConnectionsTimeoutDefault) {
		return errors.New("NATS server is not ready for connections")
	}

	zap.S().Infof("NATS Server is running on port %d", PortDefault)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to NATS server")
	}
	defer nc.Close()

	receiverErr := runReceiver(l2Requests, nc)
	if receiverErr != nil {
		return receiverErr
	}
	runPublisher(ctx, updatesChannel, bu, scheme, nc)
	return nil
}
