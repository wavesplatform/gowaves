package blockchaininfo

import (
	"context"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainUpdatesExtension struct {
	ctx                      context.Context
	l2ContractAddress        proto.WavesAddress
	bUpdatesChannel          chan proto.BUpdatesInfo
	firstBlock               *bool
	blockchainExtensionState *BUpdatesExtensionState
	lock                     sync.Mutex
	makeExtensionReadyFunc   func()
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan proto.BUpdatesInfo,
	blockchainExtensionState *BUpdatesExtensionState,
	firstBlock *bool,
	makeExtensionReadyFunc func(),
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		ctx:                      ctx,
		l2ContractAddress:        l2ContractAddress,
		bUpdatesChannel:          bUpdatesChannel,
		firstBlock:               firstBlock,
		blockchainExtensionState: blockchainExtensionState,
		makeExtensionReadyFunc:   makeExtensionReadyFunc,
	}
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesExtension) MarkExtensionReady() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.makeExtensionReadyFunc()
}

func (e *BlockchainUpdatesExtension) IsFirstRequestedBlock() bool {
	return *e.firstBlock
}

func (e *BlockchainUpdatesExtension) EmptyPreviousState() {
	e.lock.Lock()
	defer e.lock.Unlock()
	*e.firstBlock = true
	e.blockchainExtensionState.PreviousState = nil
}

func (e *BlockchainUpdatesExtension) Close() {
	if e.bUpdatesChannel != nil {
		close(e.bUpdatesChannel)
	}
	e.bUpdatesChannel = nil
}
