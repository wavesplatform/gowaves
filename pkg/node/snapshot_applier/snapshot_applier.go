package snapshot_applier

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type SnapshotApplier struct {
}

func NewSnapshotApplier() *SnapshotApplier {
	return &SnapshotApplier{}
}

func (a *SnapshotApplier) BlockSnapshotExists(state state.State, blockID proto.BlockID) (bool, error) {
	return true, nil
}

func (a *SnapshotApplier) Apply(state state.State, snapshots []state.TransactionSnapshot, block []proto.BlockID) (proto.Height, error) {
	return 0, nil
}

func (a *SnapshotApplier) ApplyMicro(state state.State, snapshots []state.TransactionSnapshot, block []proto.BlockID) (proto.Height, error) {
	return 0, nil
}
