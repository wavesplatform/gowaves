package blockchaininfo

import "github.com/wavesplatform/gowaves/pkg/proto"

/* Topics */
const (
	BlockUpdates      = "block_topic"
	microblockUpdates = "microblock_topic"
	ContractUpdates   = "contract_topi"
)

var Topics = []string{BlockUpdates, microblockUpdates, ContractUpdates}

// block updates

type BlockUpdatesInfo struct {
	Height      uint64             `json:"height"`
	VRF         proto.B58Bytes     `json:"vrf"`
	BlockID     proto.BlockID      `json:"block_id"`
	BlockHeader *proto.BlockHeader `json:"block_header"`
}

// l2 contract data entries

type L2ContractDataEntries struct {
	AllDataEntries []proto.DataEntry `json:"all_data_entries"`
}
