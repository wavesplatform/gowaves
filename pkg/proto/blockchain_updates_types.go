package proto

import (
	"context"
	"log/slog"
	"sync"
	"time"
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
	l2ContractAddress             WavesAddress
	lock                          sync.Mutex
	ready                         bool
	bUpdatesChannel               chan<- BUpdatesInfo
	ctx                           context.Context
}

func NewBlockchainUpdatesPluginInfo(ctx context.Context,
	l2Address WavesAddress, bUpdatesChannel chan<- BUpdatesInfo,
	enableBlockchainUpdatesPlugin bool) *BlockchainUpdatesPluginInfo {
	return &BlockchainUpdatesPluginInfo{
		l2ContractAddress:             l2Address,
		bUpdatesChannel:               bUpdatesChannel,
		ctx:                           ctx,
		enableBlockchainUpdatesPlugin: enableBlockchainUpdatesPlugin,
		ready:                         false,
	}
}

func (e *BlockchainUpdatesPluginInfo) IsBlockchainUpdatesEnabled() bool {
	return e.enableBlockchainUpdatesPlugin
}

func (e *BlockchainUpdatesPluginInfo) L2ContractAddress() WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesPluginInfo) IsReady() bool {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.enableBlockchainUpdatesPlugin && e.ready
}

func (e *BlockchainUpdatesPluginInfo) MakeExtensionReady() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.ready = true
}

func (e *BlockchainUpdatesPluginInfo) WriteBUpdates(bUpdates BUpdatesInfo) {
	if e.bUpdatesChannel == nil || !e.IsReady() {
		return
	}
	select {
	case e.bUpdatesChannel <- bUpdates:
	case <-time.After(ChannelWriteTimeout):
		slog.Error("failed to write into the blockchain updates channel, out of time")
		return
	case <-e.ctx.Done():
		e.Close()
		return
	}
}

func (e *BlockchainUpdatesPluginInfo) Close() {
	if e.bUpdatesChannel != nil {
		close(e.bUpdatesChannel)
	}
}
