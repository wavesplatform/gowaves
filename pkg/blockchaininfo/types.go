package blockchaininfo

import (
	"bytes"
	"context"
	"io"
	"math"
	"slices"
	"sync"

	"github.com/ccoveille/go-safecast"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var ErrNotInCache = errors.New("the target height is not in cache")

const (
	RootHashSize = 32

	HistoryJournalLengthMax = 100
)

type UpdatesPublisherInterface interface {
	PublishUpdates(ctx context.Context, updates proto.BUpdatesInfo,
		nc *nats.Conn, scheme proto.Scheme, l2ContractAddress string) error
	L2ContractAddress() string
}

type StateCacheRecord struct {
	dataEntries map[string]proto.DataEntry
	blockInfo   proto.BlockUpdatesInfo
}

func NewStateCacheRecord(dataEntries []proto.DataEntry, blockInfo proto.BlockUpdatesInfo) StateCacheRecord {
	var stateCacheRecord StateCacheRecord
	stateCacheRecord.dataEntries = make(map[string]proto.DataEntry, len(dataEntries))

	for _, dataEntry := range dataEntries {
		stateCacheRecord.dataEntries[dataEntry.GetKey()] = dataEntry
	}
	stateCacheRecord.blockInfo = blockInfo
	return stateCacheRecord
}

type StateCache struct {
	lock    sync.Mutex
	records map[proto.Height]StateCacheRecord
	heights []uint64
}

func NewStateCache() *StateCache {
	return &StateCache{
		records: make(map[proto.Height]StateCacheRecord),
	}
}

func (sc *StateCache) SearchValue(key string, height uint64) (proto.DataEntry, bool) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	record, found := sc.records[height]
	if !found {
		return nil, false
	}
	entry, ok := record.dataEntries[key]
	return entry, ok
}

func (sc *StateCache) SearchBlockInfo(height uint64) (proto.BlockUpdatesInfo, error) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	if _, ok := sc.records[height]; !ok {
		return proto.BlockUpdatesInfo{}, ErrNotInCache
	}
	return sc.records[height].blockInfo, nil
}

func (sc *StateCache) AddCacheRecord(height uint64, dataEntries []proto.DataEntry, blockInfo proto.BlockUpdatesInfo) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	// clean the oldest record if the cache is too big
	if len(sc.heights) > HistoryJournalLengthMax {
		minHeight := slices.Min(sc.heights)
		delete(sc.records, minHeight)
	}
	stateCacheRecord := NewStateCacheRecord(dataEntries, blockInfo)
	sc.records[height] = stateCacheRecord
	sc.heights = append(sc.heights, height)
}

func (sc *StateCache) RemoveCacheRecord(targetHeight uint64) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	delete(sc.records, targetHeight)

	if i := slices.Index(sc.heights, targetHeight); i != -1 {
		sc.heights = append(sc.heights[:i], sc.heights[i+1:]...)
	}
}

type HistoryEntry struct {
	Height      uint64
	BlockID     proto.BlockID
	VRF         proto.B58Bytes
	BlockHeader proto.BlockHeader

	Entries proto.DataEntries
}

type HistoryJournal struct {
	lock           sync.Mutex
	stateCache     *StateCache
	historyJournal [HistoryJournalLengthMax]HistoryEntry
	top            int
	size           int
}

func NewHistoryJournal() *HistoryJournal {
	return &HistoryJournal{
		top:  0,
		size: 0,
	}
}

func (hj *HistoryJournal) SetStateCache(stateCache *StateCache) {
	hj.stateCache = stateCache
}

// FetchKeysUntilBlockID TODO write tests.
// FetchKeysUntilBlockID goes from top to bottom and fetches all keys.
// If the blockID is found, it returns the keys up to and including that element and true.
// If the blockID is not found - nil and false.
func (hj *HistoryJournal) FetchKeysUntilBlockID(blockID proto.BlockID) ([]string, bool) {
	hj.lock.Lock()
	defer hj.lock.Unlock()

	var keys []string
	for i := 0; i < hj.size; i++ {
		idx := (hj.top - 1 - i + HistoryJournalLengthMax) % HistoryJournalLengthMax
		historyEntry := hj.historyJournal[idx]

		dataEntries := historyEntry.Entries
		for _, dataEntry := range dataEntries {
			keys = append(keys, dataEntry.GetKey())
		}
		if historyEntry.BlockID == blockID {
			return keys, true
		}
	}

	return nil, false
}

// SearchByBlockID TODO write tests.
func (hj *HistoryJournal) SearchByBlockID(blockID proto.BlockID) (HistoryEntry, bool) {
	hj.lock.Lock()
	defer hj.lock.Unlock()

	// Iterate over the elements from the top (latest) to the bottom.
	for i := 0; i < hj.size; i++ {
		idx := (hj.top - 1 - i + HistoryJournalLengthMax) % HistoryJournalLengthMax
		if hj.historyJournal[idx].BlockID == blockID {
			return hj.historyJournal[idx], true
		}
	}
	return HistoryEntry{}, false
}

// SearchByBlockID TODO write tests.
func (hj *HistoryJournal) TopHeight() (uint64, error) {
	hj.lock.Lock()
	defer hj.lock.Unlock()

	if hj.size == 0 {
		return 0, errors.New("failed to pull the top height, history journal is empty")
	}

	// Shift "top" back.
	hj.top = (hj.top - 1 + HistoryJournalLengthMax) % HistoryJournalLengthMax
	topHeight := hj.historyJournal[hj.top].Height
	return topHeight, nil
}

// CleanAfterRollback TODO write tests.
func (hj *HistoryJournal) CleanAfterRollback(latestHeightFromHistory uint64, heightAfterRollback uint64) error {
	hj.lock.Lock()
	defer hj.lock.Unlock()

	distance := latestHeightFromHistory - heightAfterRollback
	if distance > math.MaxInt64 {
		return errors.New("distance too large to fit in an int64")
	}
	dist, err := safecast.ToInt(distance)
	if err != nil {
		return errors.Wrapf(err, "failed to convert int64 to int")
	}
	if dist > hj.size {
		return errors.New("distance out of range")
	}

	// Remove the number of elements from the top to `distance`.
	hj.top = (hj.top - dist + HistoryJournalLengthMax) % HistoryJournalLengthMax
	hj.size -= int(distance)
	return nil
}

func (hj *HistoryJournal) Push(v HistoryEntry) {
	hj.lock.Lock()
	defer hj.lock.Unlock()
	hj.historyJournal[hj.top] = v // Add to top or rewrite the oldest element.

	hj.top = (hj.top + 1) % HistoryJournalLengthMax

	if hj.size < HistoryJournalLengthMax {
		hj.size++
	}
}

func (hj *HistoryJournal) Pop() (HistoryEntry, error) {
	hj.lock.Lock()
	defer hj.lock.Unlock()

	if hj.size == 0 {
		return HistoryEntry{}, errors.New("failed to pop from the history journal, it's empty")
	}

	// Shift "top" back.
	hj.top = (hj.top - 1 + HistoryJournalLengthMax) % HistoryJournalLengthMax
	entry := hj.historyJournal[hj.top]
	hj.size--
	return entry, nil
}

func CompareBUpdatesInfo(current, previous proto.BUpdatesInfo,
	scheme proto.Scheme) (bool, proto.BUpdatesInfo, error) {
	changes := proto.BUpdatesInfo{
		BlockUpdatesInfo:    proto.BlockUpdatesInfo{},
		ContractUpdatesInfo: proto.L2ContractDataEntries{},
	}

	equal := true
	if current.BlockUpdatesInfo.Height != previous.BlockUpdatesInfo.Height {
		equal = false
		changes.BlockUpdatesInfo.Height = current.BlockUpdatesInfo.Height
	}
	if !bytes.Equal(current.BlockUpdatesInfo.VRF, previous.BlockUpdatesInfo.VRF) {
		equal = false
		changes.BlockUpdatesInfo.VRF = current.BlockUpdatesInfo.VRF
	}
	if !bytes.Equal(current.BlockUpdatesInfo.BlockID.Bytes(), previous.BlockUpdatesInfo.BlockID.Bytes()) {
		equal = false
		changes.BlockUpdatesInfo.BlockID = current.BlockUpdatesInfo.BlockID
	}
	equalHeaders, err := compareBlockHeader(current.BlockUpdatesInfo.BlockHeader,
		previous.BlockUpdatesInfo.BlockHeader, scheme)
	if err != nil {
		return false, proto.BUpdatesInfo{}, err
	}
	if !equalHeaders {
		equal = false
		changes.BlockUpdatesInfo.BlockHeader = current.BlockUpdatesInfo.BlockHeader
	}

	equalEntries, dataEntryChanges, err := compareDataEntries(current.ContractUpdatesInfo.AllDataEntries,
		previous.ContractUpdatesInfo.AllDataEntries)
	if err != nil {
		return false, proto.BUpdatesInfo{}, err
	}
	if !equalEntries {
		equal = false
		changes.ContractUpdatesInfo.AllDataEntries = dataEntryChanges
		changes.ContractUpdatesInfo.Height = current.BlockUpdatesInfo.Height
	}

	return equal, changes, nil
}

func compareBlockHeader(a, b proto.BlockHeader, scheme proto.Scheme) (bool, error) {
	blockAbytes, err := a.MarshalHeader(scheme)
	if err != nil {
		return false, err
	}

	blockBbytes, err := b.MarshalHeader(scheme)
	if err != nil {
		return false, err
	}

	return bytes.Equal(blockAbytes, blockBbytes), nil
}

func compareDataEntries(current, previous proto.DataEntries) (bool, []proto.DataEntry, error) {
	currentMap := make(map[string][]byte)
	previousMap := make(map[string][]byte)

	for _, dataEntry := range current {
		value, err := dataEntry.MarshalValue()
		if err != nil {
			return false, nil, err
		}
		currentMap[dataEntry.GetKey()] = value
	}

	for _, dataEntry := range previous {
		value, err := dataEntry.MarshalValue()
		if err != nil {
			return false, nil, err
		}
		previousMap[dataEntry.GetKey()] = value
	}
	var changes []proto.DataEntry

	for key, valueCur := range currentMap {
		// Existing keys, not found in the previous state. This means that these keys were added.
		if valuePrev, found := previousMap[key]; !found {
			entryChange, err := proto.NewDataEntryFromValueBytes(valueCur)
			if err != nil {
				return false, nil, err
			}
			entryChange.SetKey(key)
			changes = append(changes, entryChange)
			// Existing keys, found in the previous state, different values. This means that data changed.
		} else if !bytes.Equal(valuePrev, valueCur) {
			entryChange, err := proto.NewDataEntryFromValueBytes(valueCur)
			if err != nil {
				return false, nil, err
			}
			entryChange.SetKey(key)
			changes = append(changes, entryChange)
		}
	} // Else, the keys were deleted, the flow goes on.

	// Keys existing in the previous state, not found in the current state. This means that these keys were deleted.
	for key := range previousMap {
		if _, found := currentMap[key]; !found {
			deleteEntry := &proto.DeleteDataEntry{}
			deleteEntry.SetKey(key)
			changes = append(changes, deleteEntry)
		}
	}

	equal := len(changes) == 0
	return equal, changes, nil
}

type BlockMeta struct {
	BlockHeight          int64  `json:"blockHeight"`
	BlockEpoch           int64  `json:"blockEpoch"`
	BlockParent          []byte `json:"blockParent"`
	ChainID              int64  `json:"chainId"`
	E2CTransfersRootHash []byte `json:"e2cTransfersRootHash"`
	LastC2ETransferIndex int64  `json:"lastC2ETransferIndex"`
}

func readBytes(reader *bytes.Reader, length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (bm *BlockMeta) UnmarshalBinary(value []byte) error {
	var err error
	binaryData := value

	reader := bytes.NewReader(binaryData)
	// Step 1: Extract blockHeight, 8 bytes
	bm.BlockHeight, err = readInt64(reader)
	if err != nil {
		return errors.Errorf("failed to read block height from blockMeta: %v", err)
	}
	// Step 2: Extract blockEpoch, 8 bytes
	bm.BlockEpoch, err = readInt64(reader)
	if err != nil {
		return errors.Errorf("failed to read block epoch from blockMeta: %v", err)
	}
	// Step 3: Extract blockParent, 32 bytes
	bm.BlockParent, err = readBytes(reader, crypto.DigestSize)
	if err != nil {
		return errors.Errorf("failed to read block parent from blockMeta: %v", err)
	}
	// Step 4: Extract chainId, 8 bytes
	bm.ChainID, err = readInt64(reader)
	if err != nil {
		return errors.Errorf("failed to read chain ID from blockMeta: %v", err)
	}
	// How many bytes are left to read
	remainingBytes := reader.Len()
	// Step 5: Extract e2cTransfersRootHash
	if remainingBytes >= RootHashSize {
		bm.E2CTransfersRootHash, err = readBytes(reader, RootHashSize)
		if err != nil {
			return errors.Errorf("failed to read E2CTransfersRootHash from blockMeta: %v", err)
		}
	} else {
		bm.E2CTransfersRootHash = nil // Represents base58''
	}
	// Step 6: Extract lastC2ETransferIndex
	if remainingBytes == 8 || remainingBytes > RootHashSize {
		index, readErr := readInt64(reader)
		if readErr != nil {
			return errors.Errorf("failed to read lastC2ETransferIndex from blockMeta: %v", readErr)
		}
		bm.LastC2ETransferIndex = index
	} else {
		bm.LastC2ETransferIndex = -1
	}
	return nil
}
