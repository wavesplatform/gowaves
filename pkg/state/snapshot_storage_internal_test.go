package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSaveSnapshots(t *testing.T) {
	storage := createStorageObjects(t, true)
	snapshotStor := newSnapshotsAtHeight(storage.hs)
	ids := genRandBlockIds(t, 1)
	snapshots := proto.TransactionSnapshot{
		proto.WavesBalanceSnapshot{Address: *generateRandomRecipient(t).Address(), Balance: 100},
		proto.WavesBalanceSnapshot{Address: *generateRandomRecipient(t).Address(), Balance: 100},
		proto.WavesBalanceSnapshot{Address: *generateRandomRecipient(t).Address(), Balance: 100},
		proto.WavesBalanceSnapshot{Address: *generateRandomRecipient(t).Address(), Balance: 100},
	}
	storage.addBlock(t, ids[0])
	err := snapshotStor.saveSnapshots(ids[0], 10, snapshots)
	assert.NoError(t, err)

	fromStorage, err := snapshotStor.shapshots(10)
	assert.NoError(t, err)

	assert.Equal(t, len(fromStorage.Balances), len(snapshots))
}
