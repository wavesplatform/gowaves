package state

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type assetsTestObjects struct {
	stor   *testStorageObjects
	assets *assets
}

func createAssets() (*assetsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	assets := newAssets(stor.db, stor.dbBatch, stor.hs)
	return &assetsTestObjects{stor, assets}, path, nil
}

func TestIssueAsset(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := defaultAssetInfo(false)
	err = to.assets.issueAsset(assetID, asset, blockID0)
	assert.NoError(t, err, "failed to issue asset")
	inf, err := to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get newest asset info")
	if !inf.equal(asset) {
		t.Errorf("Assets differ.")
	}
	to.stor.flush(t)
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
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := defaultAssetInfo(true)
	err = to.assets.issueAsset(assetID, asset, blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.assets.reissueAsset(assetID, &assetReissueChange{false, 1}, blockID0, true)
	assert.NoError(t, err, "failed to reissue asset")
	asset.reissuable = false
	asset.quantity.Add(&asset.quantity, big.NewInt(1))
	to.stor.flush(t)
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
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := defaultAssetInfo(false)
	err = to.assets.issueAsset(assetID, asset, blockID0)
	assert.NoError(t, err, "failed to issue asset")
	err = to.assets.burnAsset(assetID, &assetBurnChange{1}, blockID0, true)
	assert.NoError(t, err, "failed to burn asset")
	asset.quantity.Sub(&asset.quantity, big.NewInt(1))
	to.stor.flush(t)
	resAsset, err := to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("Assets after burn differ.")
	}
}

func TestUpdateAssetInfo(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := defaultAssetInfo(false)
	err = to.assets.issueAsset(assetID, asset, blockID0)
	assert.NoError(t, err, "failed to issue asset")
	to.stor.flush(t)

	to.stor.addBlock(t, blockID1)
	ch := &assetInfoChange{newName: "newName", newDescription: "newDescription", newHeight: 1}
	err = to.assets.updateAssetInfo(assetID, ch, blockID1, true)
	assert.NoError(t, err, "failed to update asset info")

	asset.name = ch.newName
	asset.description = ch.newDescription

	resAsset, err := to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	assert.Equal(t, asset, resAsset)

	to.stor.flush(t)

	resAsset, err = to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	assert.Equal(t, asset, resAsset)
}

func TestNewestLastUpdateHeight(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := defaultAssetInfo(false)
	err = to.assets.issueAsset(assetID, asset, blockID0)
	assert.NoError(t, err, "failed to issue asset")

	lastUpdateHeight, err := to.assets.newestLastUpdateHeight(assetID, true)
	assert.NoError(t, err, "failed to get last update height")
	assert.Equal(t, uint64(1), lastUpdateHeight)

	to.stor.flush(t)

	to.stor.addBlock(t, blockID1)
	ch := &assetInfoChange{newName: "newName", newDescription: "newDescription", newHeight: 2}
	err = to.assets.updateAssetInfo(assetID, ch, blockID1, true)
	assert.NoError(t, err, "failed to update asset info")

	lastUpdateHeight, err = to.assets.newestLastUpdateHeight(assetID, true)
	assert.NoError(t, err, "failed to get last update height")
	assert.Equal(t, uint64(2), lastUpdateHeight)
}

func TestAssetsUncertain(t *testing.T) {
	to, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")

	// Issue uncertain asset and check it can be retrieved with newestAssetInfo().
	asset := defaultAssetInfo(false)
	to.assets.issueAssetUncertain(assetID, asset)
	inf, err := to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get newest asset info")
	if !inf.equal(asset) {
		t.Errorf("uncertain asset was not created properly")
	}
	// Asset should not be present after dropUncertain().
	to.assets.dropUncertain()
	_, err = to.assets.newestAssetInfo(assetID, true)
	assert.Error(t, err)
	// Issue uncertain asset and commit.
	to.stor.addBlock(t, blockID0)
	to.assets.issueAssetUncertain(assetID, asset)
	err = to.assets.commitUncertain(blockID0)
	assert.NoError(t, err)
	inf, err = to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get newest asset info")
	if !inf.equal(asset) {
		t.Errorf("uncertain asset was not created properly after commit")
	}
	// Reissue and burn uncertainly.
	err = to.assets.burnAssetUncertain(assetID, &assetBurnChange{1}, true)
	assert.NoError(t, err, "failed to burn asset")
	asset.quantity.Sub(&asset.quantity, big.NewInt(1))
	resAsset, err := to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("assets after burn differ.")
	}
	err = to.assets.reissueAssetUncertain(assetID, &assetReissueChange{false, 1}, true)
	assert.NoError(t, err, "failed to reissue asset")
	asset.reissuable = false
	asset.quantity.Add(&asset.quantity, big.NewInt(1))
	resAsset, err = to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("assets after reissue differ.")
	}
	// Test commit and flush.
	err = to.assets.commitUncertain(blockID0)
	assert.NoError(t, err)
	to.assets.dropUncertain()
	resAsset, err = to.assets.newestAssetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("assets after commit differ.")
	}
	to.stor.flush(t)
	resAsset, err = to.assets.assetInfo(assetID, true)
	assert.NoError(t, err, "failed to get asset info")
	if !resAsset.equal(asset) {
		t.Errorf("assets after flush differ.")
	}
}
