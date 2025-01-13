package blockchaininfo

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const ChannelWriteTimeout = 10 * time.Second

type BlockchainUpdatesExtension struct {
	ctx                           context.Context
	enableBlockchainUpdatesPlugin bool
	l2ContractAddress             proto.WavesAddress
	bUpdatesChannel               chan<- BUpdatesInfo
	l2RequestsChannel             <-chan L2Requests
	firstBlock                    bool
	blockchainExtensionState      *BUpdatesExtensionState
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan<- BUpdatesInfo,
	requestChannel <-chan L2Requests,
	blockchainExtensionState *BUpdatesExtensionState,
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		ctx:                           ctx,
		enableBlockchainUpdatesPlugin: true,
		l2ContractAddress:             l2ContractAddress,
		bUpdatesChannel:               bUpdatesChannel,
		l2RequestsChannel:             requestChannel,
		firstBlock:                    true,
		blockchainExtensionState:      blockchainExtensionState,
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
	e.firstBlock = false
}

func (e *BlockchainUpdatesExtension) ReceiveSignals() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case l2Request := <-e.l2RequestsChannel:
			if l2Request.Restart {
				e.firstBlock = true
				e.blockchainExtensionState.previousState = nil
			}
		}
	}
}

func (e *BlockchainUpdatesExtension) WriteBUpdates(bUpdates BUpdatesInfo) {
	if e.bUpdatesChannel == nil {
		return
	}
	select {
	case e.bUpdatesChannel <- bUpdates:
	case <-time.After(ChannelWriteTimeout):
		zap.S().Errorf("failed to write into the blockchain updates channel, out of time")
		return
	case <-e.ctx.Done():
		e.close()
		return
	}
}

func (e *BlockchainUpdatesExtension) close() {
	if e.bUpdatesChannel == nil {
		return
	}
	close(e.bUpdatesChannel)
	e.bUpdatesChannel = nil
}
