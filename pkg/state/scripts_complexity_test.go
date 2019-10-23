package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type scriptsComplexityStorageObjects struct {
	stor              *testStorageObjects
	scriptsComplexity *scriptsComplexity
}

func createScriptsComplexityStorageObjects() (*scriptsComplexityStorageObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	scriptsComplexity, err := newScriptsComplexity(stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &scriptsComplexityStorageObjects{stor, scriptsComplexity}, path, nil
}

func TestSaveComplexityForAddr(t *testing.T) {
	to, path, err := createScriptsComplexityStorageObjects()
	assert.NoError(t, err, "createScriptsComplexityStorageObjects() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	byFuncs := make(map[string]uint64)
	byFuncs["lightyFunc"] = 123
	byFuncs["heavyFunc"] = 1500
	byFuncs["superHeavyFunc"] = 4000
	r := &accountScriptComplexityRecord{verifierComplexity: 100500, byFuncs: byFuncs, estimator: 1}
	err = to.scriptsComplexity.saveComplexityForAddr(addr, r, blockID0)
	assert.NoError(t, err)
	res, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, true)
	assert.NoError(t, err)
	assert.Equal(t, r, res)

	to.stor.flush(t)

	res, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, true)
	assert.NoError(t, err)
	assert.Equal(t, r, res)
}

func TestSaveComplexityForAsset(t *testing.T) {
	to, path, err := createScriptsComplexityStorageObjects()
	assert.NoError(t, err, "createScriptsComplexityStorageObjects() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	asset := testGlobal.asset0.asset.ID
	r := &assetScriptComplexityRecord{500, 2}
	err = to.scriptsComplexity.saveComplexityForAsset(asset, r, blockID0)
	assert.NoError(t, err)
	res, err := to.scriptsComplexity.newestScriptComplexityByAsset(asset, true)
	assert.NoError(t, err)
	assert.Equal(t, r, res)

	to.stor.flush(t)

	res, err = to.scriptsComplexity.newestScriptComplexityByAsset(asset, true)
	assert.NoError(t, err)
	assert.Equal(t, r, res)
}
