package blockchaininfo

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	RootHashSize = 32
)

// BlockUpdatesInfo Block updates.
type BlockUpdatesInfo struct {
	Height      uint64            `json:"height"`
	VRF         proto.B58Bytes    `json:"vrf"`
	BlockID     proto.BlockID     `json:"block_id"`
	BlockHeader proto.BlockHeader `json:"block_header"`
}

// L2ContractDataEntries L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries []proto.DataEntry `json:"all_data_entries"`
	Height         uint64            `json:"height"`
}

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

type L2Requests struct {
	Restart bool
}

func CompareBUpdatesInfo(current, previous BUpdatesInfo,
	scheme proto.Scheme) (bool, BUpdatesInfo, error) {
	changes := BUpdatesInfo{
		BlockUpdatesInfo:    BlockUpdatesInfo{},
		ContractUpdatesInfo: L2ContractDataEntries{},
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
		return false, BUpdatesInfo{}, err
	}
	if !equalHeaders {
		equal = false
		changes.BlockUpdatesInfo.BlockHeader = current.BlockUpdatesInfo.BlockHeader
	}

	equalEntries, dataEntryChanges, err := compareDataEntries(current.ContractUpdatesInfo.AllDataEntries,
		previous.ContractUpdatesInfo.AllDataEntries)
	if err != nil {
		return false, BUpdatesInfo{}, err
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
	currentMap := make(map[string][]byte)  // Data entries.
	previousMap := make(map[string][]byte) // Data entries.

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
	}

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
	n, err := reader.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != length {
		return nil, errors.Errorf("expected to read %d bytes, but read %d bytes", length, n)
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
