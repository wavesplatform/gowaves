package proto

// L2ContractDataEntries L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries DataEntries `json:"all_data_entries"`
	Height         uint64      `json:"height"`
	BlockID        BlockID     `json:"block_id"`
}

// BlockUpdatesInfo Block updates.
type BlockUpdatesInfo struct {
	Height      uint64      `json:"height"`
	VRF         B58Bytes    `json:"vrf"`
	BlockID     BlockID     `json:"block_id"`
	BlockHeader BlockHeader `json:"block_header"`
}
