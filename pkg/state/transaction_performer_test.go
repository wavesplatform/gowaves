package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type performerTestObjects struct {
	stor     *storageObjects
	entities *blockchainEntitiesStorage
	tp       *transactionPerformer
}

func createPerformerTestObjects(t *testing.T) (*performerTestObjects, []string) {
	stor, path, err := createStorageObjects()
	assert.NoError(t, err, "createStorageObjects() failed")
	entities, err := newBlockchainEntitiesStorage(stor.hs, settings.MainNetSettings)
	assert.NoError(t, err, "newBlockchainEntitiesStorage() failed")
	tp, err := newTransactionPerformer(entities, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionPerformer() failed")
	return &performerTestObjects{stor, entities, tp}, path
}

func defaultPerformerInfo(t *testing.T) *performerInfo {
	return &performerInfo{false, blockID0}
}

func TestPerformIssueV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV1(t)
	err := to.tp.performIssueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueV1() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
			blockID:    blockID0,
		},
	}

	// Check asset info.
	info, err := to.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func TestPerformIssueV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV2(t)
	err := to.tp.performIssueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performIssueV2() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
			blockID:    blockID0,
		},
	}

	// Check asset info.
	info, err := to.entities.assets.assetInfo(*tx.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func TestPerformReissueV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	tx := createReissueV1(t)
	err := to.tp.performReissueV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV1() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func TestPerformReissueV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	tx := createReissueV2(t)
	err := to.tp.performReissueV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performReissueV2() failed")
	to.stor.flush(t)
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Check asset info.
	info, err := to.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func TestPerformBurnV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	tx := createBurnV1(t)
	err := to.tp.performBurnV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnV1() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV1 transaction")
}

func TestPerformBurnV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetInfo := createAsset(t, to.entities, to.stor, testGlobal.asset0.asset.ID)
	tx := createBurnV2(t)
	err := to.tp.performBurnV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performBurnV2() failed")
	to.stor.flush(t)
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	// Check asset info.
	info, err := to.entities.assets.assetInfo(testGlobal.asset0.asset.ID, true)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV2 transaction")
}

func TestPerformLeaseV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV1(t)
	err := to.tp.performLeaseV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseV1 transaction")
}

func TestPerformLeaseV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV2(t)
	err := to.tp.performLeaseV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    true,
		leaseAmount: tx.Amount,
		recipient:   *tx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}

	info, err := to.entities.leases.leasingInfo(*tx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseV1 transaction")
}

func TestPerformLeaseCancelV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV1(t)
	err := to.tp.performLeaseV1(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV1() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelV1(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV1() failed")
	to.stor.flush(t)
	info, err := to.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelV1 transaction")
}

func TestPerformLeaseCancelV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV2(t)
	err := to.tp.performLeaseV2(leaseTx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseV2() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	leasingInfo := &leasing{
		isActive:    false,
		leaseAmount: leaseTx.Amount,
		recipient:   *leaseTx.Recipient.Address,
		sender:      testGlobal.senderInfo.addr,
	}
	tx := createLeaseCancelV2(t, *leaseTx.ID)
	err = to.tp.performLeaseCancelV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performLeaseCancelV2() failed")
	to.stor.flush(t)
	info, err := to.entities.leases.leasingInfo(*leaseTx.ID, true)
	assert.NoError(t, err, "leasingInfo() failed")
	assert.Equal(t, *leasingInfo, *info, "invalid leasing info after performing LeaseCancelV2 transaction")
}

func TestPerformCreateAliasV1(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV1(t)
	err := to.tp.performCreateAliasV1(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV1() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	addr, err := to.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasV2 transaction")
}

func TestPerformCreateAliasV2(t *testing.T) {
	to, path := createPerformerTestObjects(t)

	defer func() {
		err := util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createCreateAliasV2(t)
	err := to.tp.performCreateAliasV2(tx, defaultPerformerInfo(t))
	assert.NoError(t, err, "performCreateAliasV2() failed")
	to.stor.addBlock(t, blockID0)
	to.stor.flush(t)
	addr, err := to.entities.aliases.addrByAlias(tx.Alias.Alias, true)
	assert.NoError(t, err, "addrByAlias failed")
	assert.Equal(t, testGlobal.senderInfo.addr, *addr, "invalid address by alias after performing CreateAliasV2 transaction")
}
