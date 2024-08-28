package blockchaininfo

import (
	"bytes"
	"sort"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Block updates.
type BlockUpdatesInfo struct {
	Height      *uint64            `json:"height"`
	VRF         *proto.B58Bytes    `json:"vrf"`
	BlockID     *proto.BlockID     `json:"block_id"`
	BlockHeader *proto.BlockHeader `json:"block_header"`
}

// L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries *[]proto.DataEntry `json:"all_data_entries"`
	Height         *uint64            `json:"height"`
}

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

// TODO wrap errors.

type infoChanges map[string]interface{}

const (
	heightKey      = "heightChange"
	VRFkey         = "vrfChange"
	blockIDkey     = "blockIDChange"
	blockHeaderKey = "blockHeaderChange"
	dataEntriesKey = "dataEntriesKey"
)

func compareBUpdatesInfo(current, previous BUpdatesInfo, scheme proto.Scheme) (bool, BUpdatesInfo, error) {
	changes := BUpdatesInfo{
		BlockUpdatesInfo:    BlockUpdatesInfo{},
		ContractUpdatesInfo: L2ContractDataEntries{},
	}

	equal := true

	if current.BlockUpdatesInfo.Height != previous.BlockUpdatesInfo.Height {
		equal = false
		//changes[heightKey] = current.BlockUpdatesInfo.Height
		changes.BlockUpdatesInfo.Height = current.BlockUpdatesInfo.Height
	}
	if !bytes.Equal(*current.BlockUpdatesInfo.VRF, *previous.BlockUpdatesInfo.VRF) {
		equal = false
		changes.BlockUpdatesInfo.VRF = current.BlockUpdatesInfo.VRF
		//changes[VRFkey] = current.BlockUpdatesInfo.VRF
	}
	if !bytes.Equal(current.BlockUpdatesInfo.BlockID.Bytes(), previous.BlockUpdatesInfo.BlockID.Bytes()) {
		equal = false
		changes.BlockUpdatesInfo.BlockID = current.BlockUpdatesInfo.BlockID
		//changes[blockIDkey] = current.BlockUpdatesInfo.BlockID
	}
	equalHeaders, err := compareBlockHeader(current.BlockUpdatesInfo.BlockHeader, previous.BlockUpdatesInfo.BlockHeader, scheme)
	if err != nil {
		return false, BUpdatesInfo{}, err
	}
	if !equalHeaders {
		equal = false
		changes.BlockUpdatesInfo.BlockHeader = current.BlockUpdatesInfo.BlockHeader
		//changes[blockHeaderKey] = current.BlockUpdatesInfo.BlockHeader
	}

	equalEntries, dataEntryChanges, err := compareDataEntries(*current.ContractUpdatesInfo.AllDataEntries, *previous.ContractUpdatesInfo.AllDataEntries)
	if err != nil {
		return false, BUpdatesInfo{}, err
	}
	if !equalEntries {
		equal = false
		changes.ContractUpdatesInfo.AllDataEntries = &dataEntryChanges
		changes.ContractUpdatesInfo.Height = current.BlockUpdatesInfo.Height
		//changes[dataEntriesKey] = dataEntryChanges
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
	sort.Sort(current)
	sort.Sort(previous)
	var changes []proto.DataEntry
	equal := true
	if len(current) != len(previous) {
		equal = false
	}

	minLength := min(len(current), len(previous))
	maxLength := max(len(current), len(previous))

	for i := 0; i < minLength; i++ {
		entryEqual, err := areEntriesEqual(current[i], previous[i])
		if err != nil {
			return false, nil, err
		}
		if !entryEqual {
			equal = false
			changes = append(changes, current[i])
		}
	}
	// iterating through the rest
	restIndex := maxLength - minLength
	if len(current) > len(previous) { // this means that some keys were added
		for i := restIndex; i < maxLength; i++ {
			changes = append(changes, current[i])
		}
	} else { // this means that some keys in the current map were deleted
		for i := restIndex; i < maxLength; i++ {
			changes = append(changes, &proto.DeleteDataEntry{Key: previous[i].GetKey()})
		}
	}
	return equal, changes, nil
}

func areEntriesEqual(a, b proto.DataEntry) (bool, error) {
	if a.GetKey() != b.GetKey() {
		return false, nil
	}
	aValueBytes, err := a.MarshalValue()
	if err != nil {
		return false, err
	}
	bValue, err := b.MarshalValue()
	if err != nil {
		return false, err
	}

	return bytes.Equal(aValueBytes, bValue), nil
}

func statesEqual(state BUpdatesExtensionState, scheme proto.Scheme) (bool, BUpdatesInfo, error) {
	return compareBUpdatesInfo(*state.currentState, *state.previousState, scheme)
}
