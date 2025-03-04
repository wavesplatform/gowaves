package proto

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"time"
)

const ChannelWriteTimeout = 10 * time.Second

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

// BlockUpdatesInfo Block updates.
type BlockUpdatesInfo struct {
	Height      uint64      `json:"height"`
	VRF         B58Bytes    `json:"vrf"`
	BlockID     BlockID     `json:"block_id"`
	BlockHeader BlockHeader `json:"block_header"`
}

// L2ContractDataEntries L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries []DataEntry `json:"all_data_entries"`
	Height         uint64      `json:"height"`
}

// For appender
type BlockchainUpdatesPluginInfo struct {
	EnableBlockchainUpdatesPlugin bool
	L2ContractAddress             WavesAddress
	FirstBlock                    *bool
	Lock                          sync.Mutex
	BUpdatesChannel               chan<- BUpdatesInfo
	ctx                           context.Context
}

func NewBlockchainUpdatesPluginInfo(l2Address WavesAddress, bUpdatesChannel chan<- BUpdatesInfo, ctx context.Context, firstBlock *bool,
	enableBlockchainUpdatesPlugin bool) *BlockchainUpdatesPluginInfo {
	return &BlockchainUpdatesPluginInfo{
		L2ContractAddress:             l2Address,
		FirstBlock:                    firstBlock,
		BUpdatesChannel:               bUpdatesChannel,
		ctx:                           ctx,
		EnableBlockchainUpdatesPlugin: enableBlockchainUpdatesPlugin,
	}
}

func (e *BlockchainUpdatesPluginInfo) Ctx() context.Context {
	return e.ctx
}

func (e *BlockchainUpdatesPluginInfo) FirstBlockDone() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	*e.FirstBlock = false
}

func (e *BlockchainUpdatesPluginInfo) WriteBUpdates(bUpdates BUpdatesInfo) {
	if e.BUpdatesChannel == nil {
		return
	}
	select {
	case e.BUpdatesChannel <- bUpdates:
	case <-time.After(ChannelWriteTimeout):
		zap.S().Errorf("failed to write into the blockchain updates channel, out of time")
		return
	case <-e.ctx.Done():
		e.Close()
		return
	}
}

func (e *BlockchainUpdatesPluginInfo) Close() {
	if e.BUpdatesChannel != nil {
		close(e.BUpdatesChannel)
	}
	e.BUpdatesChannel = nil
}
