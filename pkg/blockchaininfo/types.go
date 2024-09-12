package blockchaininfo

import (
	"bytes"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// BlockUpdatesInfo Block updates.
type BlockUpdatesInfo struct {
	Height      *uint64            `json:"height"`
	VRF         *proto.B58Bytes    `json:"vrf"`
	BlockID     *proto.BlockID     `json:"block_id"`
	BlockHeader *proto.BlockHeader `json:"block_header"`
}

// L2ContractDataEntries L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries *[]proto.DataEntry `json:"all_data_entries"`
	Height         *uint64            `json:"height"`
}

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

// TODO wrap errors.

func compareBUpdatesInfo(current, previous BUpdatesInfo, scheme proto.Scheme) (bool, BUpdatesInfo, error) {
	changes := BUpdatesInfo{
		BlockUpdatesInfo:    BlockUpdatesInfo{},
		ContractUpdatesInfo: L2ContractDataEntries{},
	}

	equal := true

	if current.BlockUpdatesInfo.Height != previous.BlockUpdatesInfo.Height {
		equal = false
		changes.BlockUpdatesInfo.Height = current.BlockUpdatesInfo.Height
	}
	if !bytes.Equal(*current.BlockUpdatesInfo.VRF, *previous.BlockUpdatesInfo.VRF) {
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

	equalEntries, dataEntryChanges, err := compareDataEntries(*current.ContractUpdatesInfo.AllDataEntries,
		*previous.ContractUpdatesInfo.AllDataEntries)
	if err != nil {
		return false, BUpdatesInfo{}, err
	}
	if !equalEntries {
		equal = false
		changes.ContractUpdatesInfo.AllDataEntries = &dataEntryChanges
		changes.ContractUpdatesInfo.Height = current.BlockUpdatesInfo.Height
	}
	return equal, changes, nil
}

func compareBlockHeader(a, b *proto.BlockHeader, scheme proto.Scheme) (bool, error) {
	if a == nil || b == nil {
		return a == b, nil // both nil or one of them is nil
	}
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

	equal := changes == nil
	return equal, changes, nil
}

func statesEqual(state BUpdatesExtensionState, scheme proto.Scheme) (bool, BUpdatesInfo, error) {
	return compareBUpdatesInfo(*state.currentState, *state.previousState, scheme)
}
