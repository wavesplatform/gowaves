package blockchaininfo

import (
	"context"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlockchainUpdatesExtension struct {
	ctx                      context.Context
	l2ContractAddress        proto.WavesAddress
	bUpdatesChannel          chan proto.BUpdatesInfo
	firstBlock               *bool
	blockchainExtensionState *BUpdatesExtensionState
	lock                     sync.Mutex
	makeExtensionReadyFunc   func()
	obsolescencePeriod       time.Duration
	ntpTime                  types.Time
}

func NewBlockchainUpdatesExtension(
	ctx context.Context,
	l2ContractAddress proto.WavesAddress,
	bUpdatesChannel chan proto.BUpdatesInfo,
	blockchainExtensionState *BUpdatesExtensionState,
	firstBlock *bool,
	makeExtensionReadyFunc func(),
	obsolescencePeriod time.Duration,
	ntpTime types.Time,
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		ctx:                      ctx,
		l2ContractAddress:        l2ContractAddress,
		bUpdatesChannel:          bUpdatesChannel,
		firstBlock:               firstBlock,
		blockchainExtensionState: blockchainExtensionState,
		makeExtensionReadyFunc:   makeExtensionReadyFunc,
		obsolescencePeriod:       obsolescencePeriod,
		ntpTime:                  ntpTime,
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
