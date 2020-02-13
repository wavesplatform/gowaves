package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type HistoryBlockApplier interface {
	Apply(blocks []*proto.Block) error
}

type HistoryBlockApplierImpl struct {
	services    services.Services
	scoreSender types.ScoreSender
	applier     types.BlocksApplier
}

func (a *HistoryBlockApplierImpl) Apply(blocks []*proto.Block) error {
	err := a.applier.Apply(blocks)
	if err != nil {
		return err
	}
	go a.services.BlockAddedNotifier.Handle()
	a.scoreSender.NonPriority()
	return nil
}

func NewHistoryBlockApplier(applier types.BlocksApplier, services services.Services, scoreSender types.ScoreSender) *HistoryBlockApplierImpl {
	return &HistoryBlockApplierImpl{
		services:    services,
		scoreSender: scoreSender,
		applier:     applier,
	}
}
