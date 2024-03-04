package state

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotApplierHooks interface {
	BeforeTxSnapshotApply(tx proto.Transaction, validatingUTX bool) error
	AfterTxSnapshotApply() error
}

type extendedSnapshotApplierInfo interface {
	BlockID() proto.BlockID
	BlockchainHeight() proto.Height
	CurrentBlockHeight() proto.Height
	EstimatorVersion() int
	Scheme() proto.Scheme
	StateActionsCounter() *proto.StateActionsCounter
}

type extendedSnapshotApplier interface {
	ApplierInfo() extendedSnapshotApplierInfo
	SetApplierInfo(info extendedSnapshotApplierInfo)
	filterZeroDiffsSHOut(blockID proto.BlockID)
	proto.SnapshotApplier
	internalSnapshotApplier
	snapshotApplierHooks
}

type txSnapshot struct {
	regular  []proto.AtomicSnapshot
	internal []internalSnapshot
}

func (ts txSnapshot) ApplyFixSnapshot(a extendedSnapshotApplier) error {
	return ts.Apply(a, nil, false)
}

func (ts txSnapshot) Apply(a extendedSnapshotApplier, tx proto.Transaction, validatingUTX bool) error {
	if err := a.BeforeTxSnapshotApply(tx, validatingUTX); err != nil {
		return errors.Wrapf(err, "failed to execute before tx snapshot apply hook")
	}
	// internal snapshots must be applied at the end
	for _, rs := range ts.regular {
		err := rs.Apply(a)
		if err != nil {
			return errors.Wrap(err, "failed to apply regular transaction snapshot")
		}
	}
	for _, is := range ts.internal {
		err := is.ApplyInternal(a)
		if err != nil {
			return errors.Wrap(err, "failed to apply internal transaction snapshot")
		}
	}
	if err := a.AfterTxSnapshotApply(); err != nil {
		return errors.Wrapf(err, "failed to execute after tx snapshot apply hook")
	}
	return nil
}

func (ts txSnapshot) ApplyInitialSnapshot(a extendedSnapshotApplier) error {
	// internal snapshots must be applied at the end
	for _, rs := range ts.regular {
		err := rs.Apply(a)
		if err != nil {
			return errors.Wrap(err, "failed to apply regular transaction snapshot")
		}
	}
	for _, is := range ts.internal {
		err := is.ApplyInternal(a)
		if err != nil {
			return errors.Wrap(err, "failed to apply internal transaction snapshot")
		}
	}
	return nil
}
