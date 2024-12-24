package blockchaininfo

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainUpdatesExtension struct {
	ctx                           context.Context
	enableBlockchainUpdatesPlugin bool
	l2ContractAddress             proto.WavesAddress
	bUpdatesChannel               chan<- BUpdatesInfo
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan<- BUpdatesInfo,
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		ctx:                           ctx,
		enableBlockchainUpdatesPlugin: true,
		l2ContractAddress:             l2ContractAddress,
		bUpdatesChannel:               bUpdatesChannel,
	}
}

func (e *BlockchainUpdatesExtension) EnableBlockchainUpdatesPlugin() bool {
	return e != nil && e.enableBlockchainUpdatesPlugin
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesExtension) WriteBUpdates(bUpdates BUpdatesInfo) {
	if e.bUpdatesChannel == nil {
		return
	}
	select {
	case e.bUpdatesChannel <- bUpdates:
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
