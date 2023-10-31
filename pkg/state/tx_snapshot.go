package state

import (
	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type extendedSnapshotApplier interface {
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

func (ts txSnapshot) ToProtobuf() (*g.TransactionStateSnapshot, error) {
	var res g.TransactionStateSnapshot
	for _, atomicSnapshot := range ts.regular {
		if err := atomicSnapshot.AppendToProtobuf(&res); err != nil {
			return nil, errors.Wrap(err, "failed to marshall TransactionSnapshot to proto")
		}
	}
	return &res, nil
}
