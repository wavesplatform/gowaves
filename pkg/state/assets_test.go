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
			reissuable: false,
			blockID:    blockID,
		},
	}
	err = assets.issueAsset(assetID, asset)
	assert.NoError(t, err, "failed to issue asset")
	flushAssets(t, assets)
	resAsset, err := assets.assetInfo(assetID)
	assert.NoError(t, err, "failed to get asset info")
	if *resAsset != *asset {
		t.Errorf("Assets differ.")
	}
}

func TestReissueAsset(t *testing.T) {

}

func TestBurnAsset(t *testing.T) {

}
