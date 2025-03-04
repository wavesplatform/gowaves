package blockchaininfo

import (
	"context"
	"github.com/wavesplatform/gowaves/pkg/state"
	"time"

	"go.uber.org/zap"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const StoreBlocksLimit = 200
const ConnectionsTimeoutDefault = 5 * server.AUTH_TIMEOUT

const portDefault = 4222
const hostDefault = "127.0.0.1"
const natsMaxPayloadSize int32 = 1024 * 1024 // 1 MB
const publisherWaitingTime = 100 * time.Millisecond

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
	currentState      *proto.BUpdatesInfo
	previousState     *proto.BUpdatesInfo // this information is what was just published
	Limit             uint64
	scheme            proto.Scheme
	l2ContractAddress string
	historyJournal    *HistoryJournal
	stateCache        *StateCache
	st                state.State
}

func NewBUpdatesExtensionState(limit uint64, scheme proto.Scheme, l2ContractAddress string, state state.State) (*BUpdatesExtensionState, error) {
	stateCache := NewStateCache()
	currentHeight, err := state.Height()
	if err != nil {
		return nil, err
	}
	l2address, cnvrtErr := proto.NewAddressFromString(l2ContractAddress)
	if cnvrtErr != nil {
		return nil, errors.Wrapf(cnvrtErr, "failed to convert L2 contract address %s", l2ContractAddress)
	}
	for i := 0; i < HistoryJournalLengthMax; i++ {
		dataEntriesAtHeight, err := state.RetrieveEntriesAtHeight(l2address, currentHeight)
		if err != nil {
			return nil, err
		}
		filteredDataEntries, err := filterDataEntries(currentHeight-limit, dataEntriesAtHeight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize state cache, failed to filter data entries")
		}
		stateCache.AddCacheRecord(currentHeight, filteredDataEntries)
		currentHeight--
	}

	return &BUpdatesExtensionState{Limit: limit, scheme: scheme,
		l2ContractAddress: l2ContractAddress, historyJournal: NewHistoryJournal(), stateCache: stateCache, st: state}, nil
}

func (bu *BUpdatesExtensionState) hasStateChanged() (bool, proto.BUpdatesInfo, error) {
	statesAreEqual, changes, err := bu.statesEqual(bu.scheme)
	if err != nil {
		return false, proto.BUpdatesInfo{}, err
	}
	if statesAreEqual {
		return false, proto.BUpdatesInfo{}, nil
	}
	return true, changes, nil
}

func (bu *BUpdatesExtensionState) statesEqual(scheme proto.Scheme) (bool, proto.BUpdatesInfo, error) {
	return CompareBUpdatesInfo(*bu.currentState, *bu.previousState, scheme)
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

func (bu *BUpdatesExtensionState) publishContractUpdates(contractUpdates proto.L2ContractDataEntries, nc *nats.Conn) error {
	dataEntriesProtobuf, err := L2ContractDataEntriesToProto(contractUpdates).MarshalVTStrict()
	if err != nil {
		return err
	}

	if len(dataEntriesProtobuf) <= int(natsMaxPayloadSize-1) {
		var msg []byte
		msg = append(msg, NoPaging)
		msg = append(msg, dataEntriesProtobuf...)
		err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
		if err != nil {
			zap.S().Errorf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return err
		}
		zap.S().Infof("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
		return nil
	}

	chunkedPayload := splitIntoChunks(dataEntriesProtobuf, int(natsMaxPayloadSize-1)/2)

	for i, chunk := range chunkedPayload {
		var msg []byte

		if i == len(chunkedPayload)-1 {
			msg = append(msg, EndPaging)
			msg = append(msg, chunk...)
			err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
			if err != nil {
				zap.S().Errorf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
				return err
			}
			zap.S().Infof("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
			break
		}
		msg = append(msg, StartPaging)
		msg = append(msg, chunk...)
		err = nc.Publish(ConcatenateContractTopics(bu.l2ContractAddress), msg)
		if err != nil {
			zap.S().Errorf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return err
		}
		zap.S().Infof("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
		time.Sleep(publisherWaitingTime)
	}

	return nil
}

func (bu *BUpdatesExtensionState) publishBlockUpdates(updates proto.BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
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
		zap.S().Errorf("failed to publish message on topic %s", BlockUpdates)
		return err
	}
	zap.S().Infof("Published on topic: %s\n", BlockUpdates)
	return nil
}

func (bu *BUpdatesExtensionState) publishUpdates(updates proto.BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
	/* first publish block data */
	err := bu.publishBlockUpdates(updates, nc, scheme)
	if err != nil {
		zap.S().Errorf("failed to publish message on topic %s", BlockUpdates)
		return err
	}

	/* second publish contract data entries */
	if updates.ContractUpdatesInfo.AllDataEntries != nil {
		pblshErr := bu.publishContractUpdates(updates.ContractUpdatesInfo, nc)
		if pblshErr != nil {
			zap.S().Errorf("failed to publish message on topic %s", ConcatenateContractTopics(bu.l2ContractAddress))
			return pblshErr
		}
		zap.S().Infof("Published on topic: %s\n", ConcatenateContractTopics(bu.l2ContractAddress))
	}

	return nil
}

// initHistoryJournal with 100 past entries from State when the node starts.
func (bu *BUpdatesExtensionState) initHistoryJournal(updates proto.BUpdatesInfo) {
	// TODO implement this.
}

func (bu *BUpdatesExtensionState) addEntriesToHistoryJournalAndCache(updates proto.BUpdatesInfo) {
	height := updates.BlockUpdatesInfo.Height
	blockID := updates.BlockUpdatesInfo.BlockID

	historyEntry := HistoryEntry{
		height:  height,
		blockID: blockID,
		entries: updates.ContractUpdatesInfo.AllDataEntries,
	}
	bu.historyJournal.Push(historyEntry)
	bu.stateCache.AddCacheRecord(height, updates.ContractUpdatesInfo.AllDataEntries)
}

func (bu *BUpdatesExtensionState) rollbackHappened(updates proto.BUpdatesInfo, previousState proto.BUpdatesInfo) bool {
	if _, _, blockIDFound := bu.historyJournal.SearchByBlockID(updates.BlockUpdatesInfo.BlockHeader.Parent); blockIDFound {
		return true
	}
	if updates.BlockUpdatesInfo.Height < previousState.BlockUpdatesInfo.Height {
		return true
	}
	return false
}

func (bu *BUpdatesExtensionState) generatePatch(latestUpdates proto.BUpdatesInfo) (proto.BUpdatesInfo, error) {
	keysForAnalysis, found := bu.historyJournal.FetchKeysUntilBlockID(latestUpdates.BlockUpdatesInfo.BlockID)
	if !found {
		// TODO the rollback is too deep. Emergency restart.
	}

	// If the key is found in the state, fetch it. If it is not found, it creates a DeleteEntry.
	patchDataEntries, err := bu.buildPatch(keysForAnalysis, latestUpdates.BlockUpdatesInfo.Height)
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}

	// Clean the journal after rollback
	historyJournalLatestHeight, err := bu.historyJournal.TopHeight()
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}
	err = bu.CleanRecordsAfterRollback(historyJournalLatestHeight, latestUpdates.BlockUpdatesInfo.Height)
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}

	patch := proto.BUpdatesInfo{
		BlockUpdatesInfo: latestUpdates.BlockUpdatesInfo,
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: patchDataEntries,
			Height:         latestUpdates.ContractUpdatesInfo.Height,
		},
	}
	return patch, nil
}

func (bu *BUpdatesExtensionState) buildPatch(keysForPatch []string, targetHeight uint64) (proto.DataEntries, error) {
	l2WavesAddress, cnvrtErr := proto.NewAddressFromString(bu.l2ContractAddress)
	if cnvrtErr != nil {
		return nil, errors.Wrapf(cnvrtErr, "failed to convert L2 contract address %q", bu.l2ContractAddress)
	}
	var patch proto.DataEntries
	for _, dataEntryKey := range keysForPatch {
		recipient := proto.NewRecipientFromAddress(l2WavesAddress)
		dataEntry, ok, err := bu.stateCache.SearchValue(dataEntryKey, targetHeight)
		if err != nil {
			// height is too deep
			dataEntry, err = bu.st.RetrieveEntry(recipient, dataEntryKey)
		}
		if !ok {
			dataEntry = &proto.DeleteDataEntry{Key: dataEntryKey}
		}
		patch = append(patch, dataEntry)
	}
	return patch, nil
}

func (bu *BUpdatesExtensionState) CleanRecordsAfterRollback(latestHeightFromHistory uint64, heightAfterRollback uint64) error {
	// TODO should it be historyJournalLatestHeight - latestUpdates.BlockUpdatesInfo.Height + 1?
	err := bu.historyJournal.CleanAfterRollback(latestHeightFromHistory, heightAfterRollback)
	if err != nil {
		return err
	}

	// This should never happen
	if latestHeightFromHistory < heightAfterRollback {
		return errors.New("the height after rollback is bigger than the last saved height")
	}
	for i := latestHeightFromHistory; i >= heightAfterRollback; i-- {
		bu.stateCache.RemoveCacheRecord(i)
	}
	return nil
}

func handleBlockchainUpdate(updates proto.BUpdatesInfo, bu *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn) {
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
			zap.S().Errorf("failed to publish updates, %v", pblshErr)
			return
		}
		bu.previousState = &updates
		return
	}
	if bu.rollbackHappened(updates, *bu.previousState) {
		patch, err := bu.generatePatch(updates)
		if err != nil {

		}
		pblshErr := bu.publishUpdates(patch, nc, scheme)
		if pblshErr != nil {
			zap.S().Errorf("failed to publish updates, %v", pblshErr)
			return
		}
		bu.addEntriesToHistoryJournalAndCache(patch)
		bu.previousState = &patch
	}
	// compare the current state to the previous state
	stateChanged, changes, cmprErr := bu.hasStateChanged()
	if cmprErr != nil {
		zap.S().Errorf("failed to compare current and previous states, %v", cmprErr)
		return
	}
	// if there is any diff, send the update
	if stateChanged {
		pblshErr := bu.publishUpdates(changes, nc, scheme)
		if pblshErr != nil {
			zap.S().Errorf("failed to publish changes, %v", pblshErr)
		}
		bu.addEntriesToHistoryJournalAndCache(changes)
		bu.previousState = &updates
	}
}

func runPublisher(ctx context.Context, extension *BlockchainUpdatesExtension, scheme proto.Scheme, nc *nats.Conn) {
	for {
		select {
		case updates, ok := <-extension.BUpdatesChannel:
			if !ok {
				zap.S().Errorf("the updates channel for publisher was closed")
				return
			}
			handleBlockchainUpdate(updates, extension.blockchainExtensionState, scheme, nc)
		case <-ctx.Done():
			return
		}
	}
}

func runReceiver(nc *nats.Conn, bu *BlockchainUpdatesExtension) error {
	_, subErr := nc.Subscribe(L2RequestsTopic, func(request *nats.Msg) {
		signal := string(request.Data)
		switch signal {
		case RequestRestartSubTopic:
			notNilResponse := "ok"
			err := request.Respond([]byte(notNilResponse))
			if err != nil {
				zap.S().Errorf("failed to respond to a restart signal, %v", err)
				return
			}
			bu.EmptyPreviousState()
		default:
			zap.S().Errorf("nats receiver received an unknown signal, %s", signal)
		}
	})
	return subErr
}

func (e *BlockchainUpdatesExtension) RunBlockchainUpdatesPublisher(ctx context.Context,
	scheme proto.Scheme) error {
	opts := &server.Options{
		MaxPayload: natsMaxPayloadSize,
		Host:       hostDefault,
		Port:       portDefault,
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

	zap.S().Infof("NATS Server is running on port %d", portDefault)

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to NATS server")
	}
	defer nc.Close()

	receiverErr := runReceiver(nc, e)
	if receiverErr != nil {
		return receiverErr
	}
	runPublisher(ctx, e, scheme, nc)
	return nil
}
