package blockchaininfo

import (
	"context"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const ChannelWriteTimeout = 10 * time.Second

type BlockchainUpdatesExtension struct {
	Ctx                           context.Context
	enableBlockchainUpdatesPlugin bool
	l2ContractAddress             proto.WavesAddress
	BUpdatesChannel               chan BUpdatesInfo
	firstBlock                    bool
	blockchainExtensionState      *BUpdatesExtensionState
	Lock                          sync.Mutex
	BuPatchChannel                chan proto.DataEntries
	BuPatchRequestChannel         chan []string
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan BUpdatesInfo,
	buPatchChannel chan proto.DataEntries,
	buPatchRequestChannel chan []string,
	blockchainExtensionState *BUpdatesExtensionState,
) *BlockchainUpdatesExtension {

	return &BlockchainUpdatesExtension{
		Ctx:                           ctx,
		enableBlockchainUpdatesPlugin: true,
		l2ContractAddress:             l2ContractAddress,
		BUpdatesChannel:               bUpdatesChannel,
		firstBlock:                    true,
		blockchainExtensionState:      blockchainExtensionState,
		BuPatchChannel:                buPatchChannel,
		BuPatchRequestChannel:         buPatchRequestChannel,
	}
}

func (e *BlockchainUpdatesExtension) EnableBlockchainUpdatesPlugin() bool {
	return e != nil && e.enableBlockchainUpdatesPlugin
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesExtension) IsFirstRequestedBlock() bool {
	return e.firstBlock
}

func (e *BlockchainUpdatesExtension) FirstBlockDone() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	e.firstBlock = false
}

func (e *BlockchainUpdatesExtension) EmptyPreviousState() {
	e.Lock.Lock()
	e.firstBlock = true
	e.blockchainExtensionState.previousState = nil
	defer e.Lock.Unlock()
}

func (e *BlockchainUpdatesExtension) WriteBUpdates(bUpdates BUpdatesInfo) {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	if e.BUpdatesChannel == nil {
		return
	}
	select {
	case e.BUpdatesChannel <- bUpdates:
	case <-time.After(ChannelWriteTimeout):
		zap.S().Errorf("failed to write into the blockchain updates channel, out of time")
		return
	case <-e.Ctx.Done():
		e.Close()
		return
	}
}

func (e *BlockchainUpdatesExtension) WriteBUPatch(dataEntriesPatch proto.DataEntries) {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	if e.BuPatchChannel == nil {
		return
	}
	select {
	case e.BuPatchChannel <- dataEntriesPatch:
	case <-time.After(ChannelWriteTimeout):
		zap.S().Errorf("failed to write into the bu patch channel, out of time")
		return
	case <-e.Ctx.Done():
		e.Close()
		return
	}
}

func (e *BlockchainUpdatesExtension) Close() {
	if e.BUpdatesChannel != nil {
		close(e.BUpdatesChannel)
	}
	if e.BuPatchChannel != nil {
		close(e.BuPatchChannel)
	}
	e.BUpdatesChannel = nil
	e.BuPatchChannel = nil
}
