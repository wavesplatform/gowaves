package blockchaininfo

import (
	"bytes"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Block updates.
type BlockUpdatesInfo struct {
	Height      uint64             `json:"height"`
	VRF         proto.B58Bytes     `json:"vrf"`
	BlockID     proto.BlockID      `json:"block_id"`
	BlockHeader *proto.BlockHeader `json:"block_header"`
}

// L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries []proto.DataEntry `json:"all_data_entries"`
}

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

// TODO wrap errors.

func compareBUpdatesInfo(a, b BUpdatesInfo, scheme proto.Scheme) (bool, error) {
	if a.BlockUpdatesInfo.Height != b.BlockUpdatesInfo.Height {
		return false, nil
	}
	if !bytes.Equal(a.BlockUpdatesInfo.VRF, b.BlockUpdatesInfo.VRF) {
		return false, nil
	}
	if !bytes.Equal(a.BlockUpdatesInfo.BlockID.Bytes(), b.BlockUpdatesInfo.BlockID.Bytes()) {
		return false, nil
	}
	equalHeaders, err := compareBlockHeader(a.BlockUpdatesInfo.BlockHeader, b.BlockUpdatesInfo.BlockHeader, scheme)
	if err != nil {
		return false, err
	}
	if !equalHeaders {
		return false, nil
	}

	equalEntries, err := compareDataEntries(a.ContractUpdatesInfo.AllDataEntries, b.ContractUpdatesInfo.AllDataEntries)
	if err != nil {
		return false, err
	}
	if !equalEntries {
		return false, nil
	}

	return true, nil
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

func compareDataEntries(a, b []proto.DataEntry) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	for i := range a {
		equal, err := areEntriesEqual(a[i], b[i])
		if err != nil {
			return false, err
		}
		if !equal {
			return false, nil
		}
	}
	return true, nil
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

func statesEqual(state BUpdatesExtensionState, scheme proto.Scheme) (bool, error) {
	if state.currentState == nil || state.previousState == nil {
		return state.currentState == state.previousState, nil // both nil or one of them is nil
	}
	return compareBUpdatesInfo(*state.currentState, *state.previousState, scheme)
}
