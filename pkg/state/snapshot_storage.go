package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotsAtHeight struct {
	hs     *historyStorage
	scheme proto.Scheme
}

func newSnapshotsAtHeight(hs *historyStorage, scheme proto.Scheme) *snapshotsAtHeight {
	return &snapshotsAtHeight{hs: hs, scheme: scheme}
}

func (s *snapshotsAtHeight) saveSnapshots(
	blockID proto.BlockID,
	blockHeight uint64,
	txSnapshots proto.BlockSnapshot,
) error {
	key := snapshotsKey{height: blockHeight}
	blockSnapshotsBytes, err := txSnapshots.MarshallBinary()
	if err != nil {
		return err
	}
	return s.hs.addNewEntry(snapshots, key.bytes(), blockSnapshotsBytes, blockID)
}

func (s *snapshotsAtHeight) getSnapshots(height uint64) (proto.BlockSnapshot, error) {
	key := snapshotsKey{height: height}
	snapshotsBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return proto.BlockSnapshot{}, err
	}
	var res proto.BlockSnapshot
	if err = res.UnmarshalBinary(snapshotsBytes, s.scheme); err != nil {
		return proto.BlockSnapshot{}, err
	}
	return res, nil
}
