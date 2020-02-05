package node

import (
	"bytes"
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
)

type HistoryBlockApplier interface {
	ApplyBlocksBytes(blocks [][]byte) error
}

type HistoryBlockApplierImpl struct {
	services    services.Services
	interrupter types.MinerInterrupter
	scoreSender types.ScoreSender
}

func NewHistoryBlockApplier(services services.Services, interrupter types.MinerInterrupter, scoreSender types.ScoreSender) *HistoryBlockApplierImpl {
	return &HistoryBlockApplierImpl{
		services:    services,
		interrupter: interrupter,
		scoreSender: scoreSender,
	}
}

func (a *HistoryBlockApplierImpl) ApplyBlocksBytes(blocks [][]byte) error {
	a.interrupter.Interrupt()
	locked := a.services.State.Mutex().Lock()
	defer locked.Unlock()
	h, err := a.services.State.Height()
	if err != nil {
		return err
	}
	id, err := a.services.State.HeightToBlockID(h)
	if err != nil {
		return err
	}
	parent, err := proto.BlockGetParent(blocks[0])
	if err != nil {
		return err
	}
	sig, err := proto.BlockGetSignature(blocks[0])
	if err != nil {
		return err
	}
	rollback := false
	if !bytes.Equal(id[:], parent[:]) {
		err := a.services.State.RollbackTo(parent)
		if err != nil {
			return err
		}
		rollback = true
	}
	size := 0
	groupIndex := 0
	for i, block := range blocks {
		blocksNumber := i + 1
		size += len(block)
		if (size < importer.MaxTotalBatchSizeForNetworkSync) && (blocksNumber != len(blocks)) {
			continue
		}
		blocksToApply := blocks[groupIndex:blocksNumber]
		groupIndex = blocksNumber
		if err := a.services.State.AddNewBlocks(blocksToApply); err != nil {
			zap.S().Debugf("[*] BlockDownloader: error on adding new blocks: %q, sig: %s, parent sig %s, rollback: %v", err, sig, parent, rollback)
			return err
		}
		if err := MaybeEnableExtendedApi(a.services.State, a.services.Time); err != nil {
			panic(fmt.Sprintf("[*] BlockDownloader: MaybeEnableExtendedApi(): %v. Failed to persist address transactions for API after successfully applying valid blocks.", err))
		}
		size = 0
	}
	go a.services.BlockAddedNotifier.Handle()
	a.scoreSender.NonPriority()
	return nil
}
