package state

import (
	"bytes"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type assetsTestObjects struct {
	rb      *recentBlocks
	assets  *assets
	stateDB *stateDB
}

func flushAssets(t *testing.T, to *assetsTestObjects) {
	to.rb.flush()
	err := to.assets.flush(false)
	assert.NoError(t, err, "assets.flush() failed")
	to.assets.reset()
	err = to.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	to.stateDB.reset()
}

func createAssets() (*assetsTestObjects, []string, error) {
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, nil, err
	}
	res := []string{dbDir0}
	db, err := keyvalue.NewKeyVal(dbDir0, defaultTestBloomFilterParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	rb, err := newRecentBlocks(rollbackMaxBlocks, nil)
	if err != nil {
		return nil, res, err
	}
	assets, err := newAssets(db, dbBatch, stateDB, rb)
	if err != nil {
		return nil, res, err
	}
	return &assetsTestObjects{rb, assets, stateDB}, res, nil
}

func createAssetInfo(t *testing.T, reissuable bool, blockID0 crypto.Signature, assetID crypto.Digest) *assetInfo {
	asset := &assetInfo{
		assetConstInfo: assetConstInfo{
			name:        "asset",
			description: "description",
			decimals:    2,
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   *big.NewInt(10000000),
			reissuable: reissuable,
			blockID:    blockID0,
		},
	}
	return asset
}

func TestIssueAsset(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := createAssetInfo(t, false, blockID0, assetID)
	err = to.assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	record, err := to.assets.newestAssetRecord(assetID, true)
	assert.NoError(t, err, "failed to get newest asset record")
	if !record.equal(&asset.assetHistoryRecord) {
		t.Errorf("Assets differ.")
	}
	flushAssets(t, to)
	resAsset, err := to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("Assets differ.")
	}
}

func TestReissueAsset(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := createAssetInfo(t, true, blockID0, assetID)
	err = to.assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	err = to.assets.reissueAsset(assetID, &assetReissueChange{false, 1, blockID0}, true)
	assert.NoError(t, err, "failed to reissue asset")
	asset.reissuable = false
	asset.quantity.Add(&asset.quantity, big.NewInt(1))
	flushAssets(t, to)
	resAsset, err := to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("Assets after reissue differ.")
	}
}

func TestBurnAsset(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = to.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, to.stateDB, to.rb, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := createAssetInfo(t, false, blockID0, assetID)
	err = to.assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	err = to.assets.burnAsset(assetID, &assetBurnChange{1, blockID0}, true)
	assert.NoError(t, err, "failed to burn asset")
	asset.quantity.Sub(&asset.quantity, big.NewInt(1))
	flushAssets(t, to)
	resAsset, err := to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("Assets after burn differ.")
	}
}
