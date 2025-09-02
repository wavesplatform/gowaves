package blockchaininfo

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const StoreBlocksLimit = 200

const UpdatesBufferedChannelSize = 256

const portDefault = 4222
const hostDefault = "127.0.0.1"
const natsMaxPayloadSize int32 = 1024 * 1024 // 1 MB
const publisherWaitingTime = 100 * time.Millisecond

func ConcatenateContractTopics(contractAddress string) string {
	return ContractUpdates + contractAddress
}

type BUpdatesExtensionState struct {
	CurrentState         *proto.BUpdatesInfo
	PreviousState        *proto.BUpdatesInfo // this information is what was just published
	Limit                uint64
	Scheme               proto.Scheme
	constantContractKeys []string
	L2ContractAddress    string
	HistoryJournal       *HistoryJournal
	St                   state.State
}

type UpdatesPublisher struct {
	l2ContractAddress  string
	obsolescencePeriod time.Duration
	ntpTime            types.Time
}

func NewBUpdatesExtensionState(limit uint64, scheme proto.Scheme, l2ContractAddress string,
	st state.State) (*BUpdatesExtensionState, error) {
	stateCache := NewStateCache()
	currentHeight, err := st.Height()
	if err != nil {
		return nil, err
	}
	l2address, cnvrtErr := proto.NewAddressFromString(l2ContractAddress)
	if cnvrtErr != nil {
		return nil, errors.Wrapf(cnvrtErr, "failed to convert L2 contract address %s", l2ContractAddress)
	}
	historyJournal := NewHistoryJournal()
	for targetHeight := currentHeight - HistoryJournalLengthMax; targetHeight <= currentHeight; targetHeight++ {
		blockSnapshot, retrieveErr := st.SnapshotsAtHeight(targetHeight)
		if retrieveErr != nil {
			return nil, retrieveErr
		}
		blockInfo, pullErr := st.NewestBlockInfoByHeight(targetHeight)
		if pullErr != nil {
			return nil, errors.Wrap(pullErr, "failed to get newest block info")
		}
		blockHeader, blockErr := st.NewestHeaderByHeight(targetHeight)
		if blockErr != nil {
			return nil, errors.Wrap(blockErr, "failed to get newest block info")
		}
		bUpdatesInfo, snapshotErr := state.BuildBlockUpdatesInfoFromSnapshot(blockInfo, blockHeader, blockSnapshot, l2address)
		if snapshotErr != nil {
			return nil, errors.Wrap(snapshotErr, "failed to build blocks from snapshot")
		}

		filteredDataEntries, filtrErr := filterDataEntries(targetHeight-limit,
			bUpdatesInfo.ContractUpdatesInfo.AllDataEntries)
		if filtrErr != nil {
			return nil, errors.Wrap(filtrErr, "failed to initialize state cache, failed to filter data entries")
		}

		stateCache.AddCacheRecord(targetHeight, filteredDataEntries, bUpdatesInfo.BlockUpdatesInfo)

		historyEntry := HistoryEntry{
			Height:      targetHeight,
			BlockID:     bUpdatesInfo.BlockUpdatesInfo.BlockID,
			Entries:     filteredDataEntries,
			VRF:         bUpdatesInfo.BlockUpdatesInfo.VRF,
			BlockHeader: bUpdatesInfo.BlockUpdatesInfo.BlockHeader,
		}
		historyJournal.Push(historyEntry)
	}
	historyJournal.SetStateCache(stateCache)
	return &BUpdatesExtensionState{Limit: limit, Scheme: scheme,
		L2ContractAddress: l2ContractAddress, HistoryJournal: historyJournal, St: st}, nil
}

func (bu *BUpdatesExtensionState) SetPreviousState(updates proto.BUpdatesInfo) {
	bu.PreviousState = &updates
}

func (bu *BUpdatesExtensionState) HasStateChanged() (bool, proto.BUpdatesInfo, error) {
	statesAreEqual, changes, err := bu.StatesEqual(bu.Scheme)
	if err != nil {
		return false, proto.BUpdatesInfo{}, err
	}
	if statesAreEqual {
		return false, proto.BUpdatesInfo{}, nil
	}
	return true, changes, nil
}

func (bu *BUpdatesExtensionState) StatesEqual(scheme proto.Scheme) (bool, proto.BUpdatesInfo, error) {
	return CompareBUpdatesInfo(*bu.CurrentState, *bu.PreviousState, scheme)
}

func splitIntoChunks(array []byte, maxChunkSize int) [][]byte {
	if maxChunkSize <= 0 {
		return nil
	}
	var chunkedArray [][]byte

	for i := 0; i < len(array); i += maxChunkSize {
		end := min(i+maxChunkSize, len(array))
		chunkedArray = append(chunkedArray, array[i:end])
	}

	return chunkedArray
}

func PublishContractUpdates(ctx context.Context,
	contractUpdates proto.L2ContractDataEntries, nc *nats.Conn,
	l2ContractAddress string) error {
	dataEntriesProtobuf, err := L2ContractDataEntriesToProto(contractUpdates).MarshalVTStrict()
	if err != nil {
		return err
	}

	if len(dataEntriesProtobuf) <= int(natsMaxPayloadSize-1) {
		var msg []byte
		msg = append(msg, NoPaging)
		msg = append(msg, dataEntriesProtobuf...)
		err = nc.Publish(ConcatenateContractTopics(l2ContractAddress), msg)
		if err != nil {
			slog.Error("failed to publish message on topic",
				slog.Any("topic", ConcatenateContractTopics(l2ContractAddress)),
				logging.Error(err))
			return err
		}
		return nil
	}

	chunkedPayload := splitIntoChunks(dataEntriesProtobuf, int(natsMaxPayloadSize-1)/2)

	// Select, ctx and time.After

	for i, chunk := range chunkedPayload {
		select {
		case <-time.After(publisherWaitingTime):
			var msg []byte

			if i == len(chunkedPayload)-1 {
				msg = append(msg, EndPaging)
				msg = append(msg, chunk...)
				err = nc.Publish(ConcatenateContractTopics(l2ContractAddress), msg)
				if err != nil {
					slog.Error("failed to publish message on topic",
						slog.Any("topic", ConcatenateContractTopics(l2ContractAddress)),
						logging.Error(err))
					return err
				}
				break
			}
			msg = append(msg, StartPaging)
			msg = append(msg, chunk...)
			err = nc.Publish(ConcatenateContractTopics(l2ContractAddress), msg)
			if err != nil {
				slog.Error("failed to publish message on topic",
					slog.Any("topic", ConcatenateContractTopics(l2ContractAddress)),
					logging.Error(err))
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func PublishBlockUpdates(updates proto.BUpdatesInfo, nc *nats.Conn, scheme proto.Scheme) error {
	blockInfo, err := BlockUpdatesInfoToProto(updates.BlockUpdatesInfo, scheme)
	if err != nil {
		return err
	}
	blockInfoProtobuf, err := blockInfo.MarshalVTStrict()
	if err != nil {
		return err
	}
	err = nc.Publish(BlockUpdates, blockInfoProtobuf)
	if err != nil {
		slog.Error("failed to publish message on topic",
			slog.Any("topic", BlockUpdates),
			logging.Error(err))
		return err
	}
	return nil
}

func (p *UpdatesPublisher) isObsolete(lastReceivedBlock proto.BlockHeader) (bool, error) {
	isObsolete, _, err := scheduler.IsBlockObsolete(p.ntpTime, p.obsolescencePeriod, lastReceivedBlock.Timestamp)
	if err != nil {
		return isObsolete, errors.Errorf("failed to check if block is obsolete, %v", err)
	}
	return isObsolete, nil
}

func (p *UpdatesPublisher) PublishUpdates(ctx context.Context,
	updates proto.BUpdatesInfo,
	nc *nats.Conn, scheme proto.Scheme, l2ContractAddress string) error {
	isObsolete, errObsolete := p.isObsolete(updates.BlockUpdatesInfo.BlockHeader)
	if errObsolete != nil {
		return errObsolete
	}
	if isObsolete {
		/* No updates should be published yet */
		return nil
	}

	/* first publish block data */
	err := PublishBlockUpdates(updates, nc, scheme)
	if err != nil {
		slog.Error("failed to publish message on topic",
			slog.Any("topic", BlockUpdates),
			logging.Error(err))
		return err
	}
	/* second publish contract data entries */
	pblshErr := PublishContractUpdates(ctx, updates.ContractUpdatesInfo, nc, l2ContractAddress)
	if pblshErr != nil {
		slog.Error("failed to publish message on topic",
			slog.Any("topic", ConcatenateContractTopics(p.L2ContractAddress())),
			logging.Error(pblshErr))
		return pblshErr
	}
	return nil
}

func (p *UpdatesPublisher) L2ContractAddress() string {
	return p.l2ContractAddress
}

func (bu *BUpdatesExtensionState) AddEntriesToHistoryJournalAndCache(updates proto.BUpdatesInfo) {
	height := updates.BlockUpdatesInfo.Height
	blockID := updates.BlockUpdatesInfo.BlockID

	historyEntry := HistoryEntry{
		Height:      height,
		BlockID:     blockID,
		Entries:     updates.ContractUpdatesInfo.AllDataEntries,
		VRF:         updates.BlockUpdatesInfo.VRF,
		BlockHeader: updates.BlockUpdatesInfo.BlockHeader,
	}
	bu.HistoryJournal.Push(historyEntry)
	bu.HistoryJournal.stateCache.AddCacheRecord(height, updates.ContractUpdatesInfo.AllDataEntries,
		updates.BlockUpdatesInfo)
}

func (bu *BUpdatesExtensionState) RollbackHappened(updates proto.BUpdatesInfo, previousState proto.BUpdatesInfo) bool {
	if _, blockIDFound := bu.HistoryJournal.SearchByBlockID(updates.BlockUpdatesInfo.BlockHeader.Parent); blockIDFound {
		return false
	}
	if updates.BlockUpdatesInfo.Height < previousState.BlockUpdatesInfo.Height {
		return true
	}
	return false
}

func (bu *BUpdatesExtensionState) GeneratePatch(latestUpdates proto.BUpdatesInfo) (proto.BUpdatesInfo, error) {
	keysForAnalysis, found := bu.HistoryJournal.FetchKeysUntilBlockID(latestUpdates.BlockUpdatesInfo.BlockID)
	if !found {
		previousHeight := bu.PreviousState.BlockUpdatesInfo.Height
		newHeight := latestUpdates.BlockUpdatesInfo.Height
		return proto.BUpdatesInfo{}, errors.Errorf("failed to fetch keys after rollback, the rollback is too deep."+
			"Previous height %d, new height %d", previousHeight, newHeight)
	}

	// If the key is found in the state, fetch it. If it is not found, it creates a DeleteEntry.
	patchDataEntries, patchBlockInfo, err := bu.BuildPatch(keysForAnalysis,
		latestUpdates.BlockUpdatesInfo.Height-1) // Height.
	// -1 because the current height is from the new block updates,
	// and we need to return the client's state to the previous block
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}

	// Clean the journal and cache after rollback
	historyJournalLatestHeight, err := bu.HistoryJournal.TopHeight()
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}
	err = bu.CleanRecordsAfterRollback(historyJournalLatestHeight, latestUpdates.BlockUpdatesInfo.Height)
	if err != nil {
		return proto.BUpdatesInfo{}, err
	}

	patch := proto.BUpdatesInfo{
		BlockUpdatesInfo: patchBlockInfo, // wrong, must be the previous block
		ContractUpdatesInfo: proto.L2ContractDataEntries{
			AllDataEntries: patchDataEntries,
			Height:         latestUpdates.ContractUpdatesInfo.Height - 1,
			BlockID:        patchBlockInfo.BlockID,
		},
	}
	return patch, nil
}

func (bu *BUpdatesExtensionState) IsKeyConstant(keyDataEntry string) bool {
	return slices.Contains(bu.constantContractKeys, keyDataEntry)
}

func (bu *BUpdatesExtensionState) BuildPatch(keysForPatch []string, targetHeight uint64) (proto.DataEntries,
	proto.BlockUpdatesInfo, error) {
	l2WavesAddress, cnvrtErr := proto.NewAddressFromString(bu.L2ContractAddress)
	if cnvrtErr != nil {
		return nil, proto.BlockUpdatesInfo{}, errors.Wrapf(cnvrtErr,
			"failed to convert L2 contract address %q", bu.L2ContractAddress)
	}
	recipient := proto.NewRecipientFromAddress(l2WavesAddress)
	patch := make(map[string]proto.DataEntry)
	for _, dataEntryKey := range keysForPatch {
		dataEntry, ok := bu.HistoryJournal.stateCache.SearchValue(dataEntryKey, targetHeight)
		if !ok {
			// If the key is constant, we will go to State, if not, consider it a DeleteDataEntry
			if bu.IsKeyConstant(dataEntryKey) {
				var err error
				// TODO handle not found error separately. For other errors, return with it.
				dataEntry, err = bu.St.RetrieveEntry(recipient, dataEntryKey)
				if err != nil {
					dataEntry = &proto.DeleteDataEntry{Key: dataEntryKey}
				}
			} else {
				dataEntry = &proto.DeleteDataEntry{Key: dataEntryKey}
			}
		}
		patch[dataEntry.GetKey()] = dataEntry
	}
	patchArray := slices.Collect(maps.Values(patch))
	blockInfo, err := bu.HistoryJournal.stateCache.SearchBlockInfo(targetHeight)
	if err != nil {
		return nil, proto.BlockUpdatesInfo{}, err
	}
	return patchArray, blockInfo, nil
}

func (bu *BUpdatesExtensionState) CleanRecordsAfterRollback(latestHeightFromHistory uint64,
	heightAfterRollback uint64) error {
	err := bu.HistoryJournal.CleanAfterRollback(latestHeightFromHistory, heightAfterRollback)
	if err != nil {
		return err
	}

	// This should never happen
	if latestHeightFromHistory < heightAfterRollback {
		return errors.New("the height after rollback is bigger than the last saved height")
	}
	for i := latestHeightFromHistory; i >= heightAfterRollback; i-- {
		bu.HistoryJournal.stateCache.RemoveCacheRecord(i)
	}
	return nil
}

func HandleRollback(ctx context.Context, be *BUpdatesExtensionState, updates proto.BUpdatesInfo,
	updatesPublisher UpdatesPublisherInterface,
	nc *nats.Conn, scheme proto.Scheme) proto.BUpdatesInfo {
	patch, err := be.GeneratePatch(updates)
	if err != nil {
		slog.Error("failed to generate a patch", logging.Error(err))
	}
	pblshErr := updatesPublisher.PublishUpdates(ctx, patch, nc, scheme, updatesPublisher.L2ContractAddress())
	if pblshErr != nil {
		slog.Error("failed to publish updates", logging.Error(pblshErr))
		return proto.BUpdatesInfo{}
	}
	be.AddEntriesToHistoryJournalAndCache(patch)
	be.SetPreviousState(patch)
	return patch
}

func handleBlockchainUpdate(ctx context.Context,
	updates proto.BUpdatesInfo, be *BUpdatesExtensionState, scheme proto.Scheme, nc *nats.Conn,
	updatesPublisher UpdatesPublisher, handleRollback bool) {
	// update current state
	be.CurrentState = &updates
	if be.PreviousState == nil {
		// publish initial updates
		filteredDataEntries, err := filterDataEntries(updates.BlockUpdatesInfo.Height-be.Limit,
			updates.ContractUpdatesInfo.AllDataEntries)
		if err != nil {
			return
		}
		updates.ContractUpdatesInfo.AllDataEntries = filteredDataEntries
		pblshErr := updatesPublisher.PublishUpdates(ctx,
			updates, nc, scheme, updatesPublisher.L2ContractAddress())
		if pblshErr != nil {
			slog.Error("failed to publish updates", logging.Error(pblshErr))
			return
		}
		be.PreviousState = &updates
		return
	}
	if handleRollback {
		if be.RollbackHappened(updates, *be.PreviousState) {
			HandleRollback(ctx, be, updates, &updatesPublisher, nc, scheme)
		}
	}
	// compare the current state to the previous state
	stateChanged, changes, cmprErr := be.HasStateChanged()
	if cmprErr != nil {
		slog.Error("failed to compare current and previous states", logging.Error(cmprErr))
		return
	}
	// if there is any diff, send the update
	if stateChanged {
		pblshErr := updatesPublisher.PublishUpdates(ctx, updates, nc, scheme, updatesPublisher.L2ContractAddress())
		if pblshErr != nil {
			slog.Error("failed to publish changes", logging.Error(pblshErr))
		}
		be.AddEntriesToHistoryJournalAndCache(changes)
		be.PreviousState = &updates
	}
}

func runPublisher(ctx context.Context, extension *BlockchainUpdatesExtension, scheme proto.Scheme, nc *nats.Conn,
	updatesPublisher UpdatesPublisher, updatesChannel <-chan proto.BUpdatesInfo) {
	for {
		select {
		case updates, ok := <-updatesChannel:
			if !ok {
				slog.Error("the updates channel for publisher was closed")
				return
			}
			handleBlockchainUpdate(ctx, updates, extension.blockchainExtensionState, scheme, nc,
				updatesPublisher, true)
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
				slog.Error("failed to respond to a restart signal", logging.Error(err))
				return
			}
			bu.ClearPreviousState()
		default:
			slog.Error("nats receiver received an unknown signal", slog.Any("signal", signal))
		}
	})
	return subErr
}

func publishHistoryBlocks(ctx context.Context,
	e *BlockchainUpdatesExtension, scheme proto.Scheme,
	nc *nats.Conn, updatesPublisher UpdatesPublisher) {
	for _, historyEntry := range e.blockchainExtensionState.HistoryJournal.historyJournal {
		updates := proto.BUpdatesInfo{
			BlockUpdatesInfo: proto.BlockUpdatesInfo{
				Height:      historyEntry.Height,
				BlockID:     historyEntry.BlockID,
				VRF:         historyEntry.VRF,
				BlockHeader: historyEntry.BlockHeader,
			},
			ContractUpdatesInfo: proto.L2ContractDataEntries{
				Height:         historyEntry.Height,
				AllDataEntries: historyEntry.Entries,
				BlockID:        historyEntry.BlockID,
			},
		}
		handleBlockchainUpdate(ctx, updates, e.blockchainExtensionState, scheme,
			nc, updatesPublisher, false)
	}
}
