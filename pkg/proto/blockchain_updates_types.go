package proto

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

const ChannelWriteTimeout = 10 * time.Second

type BUpdatesInfo struct {
	BlockUpdatesInfo    BlockUpdatesInfo
	ContractUpdatesInfo L2ContractDataEntries
}

// L2ContractDataEntries L2 contract data entries.
type L2ContractDataEntries struct {
	AllDataEntries DataEntries `json:"all_data_entries"`
	Height         uint64      `json:"height"`
	BlockID        BlockID     `json:"block_id"`
	BlockTimestamp int64       `json:"block_timestamp"`
}

// BlockUpdatesInfo Block updates.
type BlockUpdatesInfo struct {
	Height      uint64      `json:"height"`
	VRF         B58Bytes    `json:"vrf"`
	BlockID     BlockID     `json:"block_id"`
	BlockHeader BlockHeader `json:"block_header"`
}

type BlockchainUpdatesPluginInfo struct {
	enableBlockchainUpdatesPlugin bool
	L2ContractAddress             WavesAddress
	FirstBlock                    *bool
	Lock                          sync.Mutex
	Ready                         bool
	BUpdatesChannel               chan<- BUpdatesInfo
	ctx                           context.Context
}

func NewBlockchainUpdatesPluginInfo(ctx context.Context,
	l2Address WavesAddress, bUpdatesChannel chan<- BUpdatesInfo,
	firstBlock *bool,
	enableBlockchainUpdatesPlugin bool) *BlockchainUpdatesPluginInfo {
	return &BlockchainUpdatesPluginInfo{
		L2ContractAddress:             l2Address,
		FirstBlock:                    firstBlock,
		BUpdatesChannel:               bUpdatesChannel,
		ctx:                           ctx,
		enableBlockchainUpdatesPlugin: enableBlockchainUpdatesPlugin,
		Ready:                         false,
	}
}

func (e *BlockchainUpdatesPluginInfo) IsBlockchainUpdatesEnabled() bool {
	return e.enableBlockchainUpdatesPlugin
}

func (e *BlockchainUpdatesPluginInfo) IsReady() bool {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	return e.enableBlockchainUpdatesPlugin && e.Ready
}

func (e *BlockchainUpdatesPluginInfo) MakeExtensionReady() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	e.Ready = true
}

func (e *BlockchainUpdatesPluginInfo) Ctx() context.Context {
	return e.ctx
}

func (e *BlockchainUpdatesPluginInfo) FirstBlockDone() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	*e.FirstBlock = false
}

func (e *BlockchainUpdatesPluginInfo) IsFirstBlockDone() bool {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	return *e.FirstBlock
}

func (e *BlockchainUpdatesPluginInfo) WriteBUpdates(bUpdates BUpdatesInfo) {
	if e.BUpdatesChannel == nil || !e.IsReady() {
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
