package blockchaininfo

import (
	"context"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainUpdatesExtension struct {
	ctx                      context.Context
	l2ContractAddress        proto.WavesAddress
	BUpdatesChannel          chan proto.BUpdatesInfo
	firstBlock               *bool
	blockchainExtensionState *BUpdatesExtensionState
	Lock                     sync.Mutex
	extensionReady           chan<- struct{}
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan proto.BUpdatesInfo,
	blockchainExtensionState *BUpdatesExtensionState,
	firstBlock *bool,
	extensionReady chan<- struct{},
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		ctx:                      ctx,
		l2ContractAddress:        l2ContractAddress,
		BUpdatesChannel:          bUpdatesChannel,
		firstBlock:               firstBlock,
		blockchainExtensionState: blockchainExtensionState,
		extensionReady:           extensionReady,
	}
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesExtension) MarkExtensionReady() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	e.extensionReady <- struct{}{}
}

func (e *BlockchainUpdatesExtension) IsFirstRequestedBlock() bool {
	return *e.firstBlock
}

func (e *BlockchainUpdatesExtension) EmptyPreviousState() {
	e.Lock.Lock()
	*e.firstBlock = true
	e.blockchainExtensionState.PreviousState = nil
	defer e.Lock.Unlock()
}

func (e *BlockchainUpdatesExtension) Close() {
	if e.BUpdatesChannel != nil {
		close(e.BUpdatesChannel)
	}
	e.BUpdatesChannel = nil
}
