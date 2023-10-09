package state

import (
	protobuf "google.golang.org/protobuf/proto"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotsAtHeight struct {
	hs *historyStorage
}

func newSnapshotsAtHeight(hs *historyStorage) *snapshotsAtHeight {
	return &snapshotsAtHeight{hs: hs}
}

func (s *snapshotsAtHeight) saveSnapshots(
	blockID proto.BlockID,
	blockHeight uint64,
	txSnapshots proto.TransactionSnapshot,
) error {
	key := snapshotsKey{height: blockHeight}
	recordBytes, err := protobuf.Marshal(txSnapshots.ToProtobuf())
	if err != nil {
		return err
	}
	return s.hs.addNewEntry(snapshots, key.bytes(), recordBytes, blockID)
}

func (s *snapshotsAtHeight) shapshots(height uint64) (*g.TransactionStateSnapshot, error) {
	key := snapshotsKey{height: height}
	snapshotsBytes, err := s.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return nil, err
	}
	var res g.TransactionStateSnapshot
	if err = protobuf.Unmarshal(snapshotsBytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
