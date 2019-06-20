package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	defaultTimestamp = uint64(1465742577614)
	defaultAmount    = uint64(100)
	defaultFee       = uint64(1)
)

type differTestObjects struct {
	stor *storageObjects
	td   *transactionDiffer
}

func createDifferTestObjects(t *testing.T) (*differTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	td, err := newTransactionDiffer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionDiffer() failed")
	return &differTestObjects{stor, td}, path
}

func defaultDifferInfo(t *testing.T) *differInfo {
	return &differInfo{false, testGlobal.minerInfo.pk}
}

func createGenesis(t *testing.T) *proto.Genesis {
	return proto.NewUnsignedGenesis(testGlobal.recipientInfo.addr, defaultAmount, defaultTimestamp)
}

func TestCreateDiffGenesis(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t)
	diff, err := to.td.createDiffGenesis(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffGenesis() failed")
	correctDiff := txDiff{testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(tx.Amount)}}
	assert.Equal(t, correctDiff, diff)
}

func createPayment(t *testing.T) *proto.Payment {
	return proto.NewUnsignedPayment(testGlobal.senderInfo.pk, testGlobal.recipientInfo.addr, defaultAmount, defaultFee, defaultTimestamp)
}

func TestCreateDiffPayment(t *testing.T) {
	to, path := createDifferTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	diff, err := to.td.createDiffPayment(tx, defaultDifferInfo(t))
	assert.NoError(t, err, "createDiffPayment() failed")

	correctDiff := txDiff{
		testGlobal.senderInfo.wavesKey:    balanceDiff{balance: int64(-tx.Amount - tx.Fee)},
		testGlobal.recipientInfo.wavesKey: balanceDiff{balance: int64(tx.Amount)},
		testGlobal.minerInfo.wavesKey:     balanceDiff{balance: int64(tx.Fee)},
	}
	assert.Equal(t, correctDiff, diff)
}
