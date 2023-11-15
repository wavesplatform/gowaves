package state

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type extendedSnapshotApplier interface {
	SetApplierInfo(info *blockSnapshotsApplierInfo)
	proto.SnapshotApplier
	internalSnapshotApplier
}

type txSnapshot struct {
	regular  []proto.AtomicSnapshot
	internal []internalSnapshot
}

func (ts txSnapshot) Apply(a extendedSnapshotApplier) error {
	// internal snapshots must be applied at the end
	for _, rs := range ts.regular {
		if !rs.IsGeneratedByTxDiff() {
			err := rs.Apply(a)
			if err != nil {
				return errors.Wrap(err, "failed to apply regular transaction snapshot")
			}
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
