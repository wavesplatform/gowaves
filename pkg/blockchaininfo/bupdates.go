package blockchaininfo

import (
	"context"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockchainUpdatesExtension struct {
	Ctx                      context.Context
	l2ContractAddress        proto.WavesAddress
	BUpdatesChannel          chan proto.BUpdatesInfo
	firstBlock               *bool
	blockchainExtensionState *BUpdatesExtensionState
	Lock                     sync.Mutex
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan proto.BUpdatesInfo,
	blockchainExtensionState *BUpdatesExtensionState,
	firstBlock *bool,
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		Ctx:                      ctx,
		l2ContractAddress:        l2ContractAddress,
		BUpdatesChannel:          bUpdatesChannel,
		firstBlock:               firstBlock,
		blockchainExtensionState: blockchainExtensionState,
	}
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
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
