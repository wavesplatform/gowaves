package state

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type mock struct {
}

func (m *mock) IsValidBlock(blockID crypto.Signature) (bool, error) {
	return true, nil
}

func (m *mock) Height() (uint64, error) {
	return 0, nil
}

func (m *mock) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	return 0, nil
}

func (m *mock) RollbackMax() uint64 {
	return 1
}

func flushAssets(t *testing.T, assets *assets) {
	if err := assets.flush(); err != nil {
		t.Fatalf("flush(): %v\n", err)
	}
	assets.reset()
	if err := assets.db.Flush(assets.dbBatch); err != nil {
		t.Fatalf("db.Flush(): %v\n", err)
	}
}

func createAssets() (*assets, []string, error) {
	res := make([]string, 1)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, res, err
	}
	db, err := keyvalue.NewKeyVal(dbDir0)
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stor, err := newAssets(db, dbBatch, &mock{}, &mock{})
	if err != nil {
		return nil, res, err
	}
	res = []string{dbDir0}
	return stor, res, nil
}

func createAssetInfo(t *testing.T, reissuable bool, blockID crypto.Signature) (*assetInfo, crypto.Digest) {
	assetID, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	assert.NoError(t, err, "failed to create digest from bytes")
	asset := &assetInfo{
		assetConstInfo: assetConstInfo{
			name:        "asset",
			description: "description",
			decimals:    2,
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   10000000,
			reissuable: reissuable,
			blockID:    blockID,
		},
	}
	return asset, assetID
}

func TestIssueAsset(t *testing.T) {
	assets, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
	asset, assetID := createAssetInfo(t, false, blockID)
	err = assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	record, err := assets.newestAssetRecord(assetID)
	assert.NoError(t, err, "failed to get newest asset record")
	if *record != asset.assetHistoryRecord {
		t.Errorf("Assets differ.")
	}
	flushAssets(t, assets)
	resAsset, err := assets.assetInfo(assetID)
	assert.NoError(t, err, "failed to get asset info")
	if *resAsset != *asset {
		t.Errorf("Assets differ.")
	}
}

func TestReissueAsset(t *testing.T) {
	assets, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
	asset, assetID := createAssetInfo(t, true, blockID)
	err = assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	err = assets.reissueAsset(assetID, &assetReissueChange{false, 1, blockID})
	assert.NoError(t, err, "failed to reissue asset")
	asset.reissuable = false
	asset.quantity += 1
	flushAssets(t, assets)
	resAsset, err := assets.assetInfo(assetID)
	assert.NoError(t, err, "failed to get asset info")
	if *resAsset != *asset {
		t.Errorf("Assets after reissue differ.")
	}
}

func TestBurnAsset(t *testing.T) {
	assets, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")

	defer func() {
		err = assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	blockID, err := crypto.NewSignatureFromBytes(bytes.Repeat([]byte{0xff}, crypto.SignatureSize))
	assert.NoError(t, err, "failed to create signature from bytes")
	asset, assetID := createAssetInfo(t, false, blockID)
	err = assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	err = assets.burnAsset(assetID, &assetBurnChange{1, blockID})
	assert.NoError(t, err, "failed to burn asset")
	asset.quantity -= 1
	flushAssets(t, assets)
	resAsset, err := assets.assetInfo(assetID)
	assert.NoError(t, err, "failed to get asset info")
	if *resAsset != *asset {
		t.Errorf("Assets after burn differ.")
	}
}
