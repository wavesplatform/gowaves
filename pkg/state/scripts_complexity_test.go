package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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
	scriptsComplexity := newScriptsComplexity(stor.hs)
	return &scriptsComplexityStorageObjects{stor, scriptsComplexity}, path, nil
}

func TestSaveComplexityForAddr(t *testing.T) {
	to, path, err := createScriptsComplexityStorageObjects()
	require.NoError(t, err, "createScriptsComplexityStorageObjects() failed")

	defer func() {
		to.stor.close(t)
		err = common.CleanTemporaryDirs(path)
		require.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr

	est1 := ride.TreeEstimation{
		Estimation: 1234567890,
		Verifier:   11111,
		Functions:  map[string]int{"lightFunc": 123, "heavyFunc": 1000, "superHeavyFunc": 1234567890},
	}
	est2 := ride.TreeEstimation{
		Estimation: 5647382910,
		Verifier:   22222,
		Functions:  map[string]int{"lightFunc": 456, "heavyFunc": 2000, "superHeavyFunc": 5647382910},
	}
	est3 := ride.TreeEstimation{
		Estimation: 9876543210,
		Verifier:   33333,
		Functions:  map[string]int{"lightFunc": 789, "heavyFunc": 3000, "superHeavyFunc": 9876543210},
	}
	estimations := map[int]ride.TreeEstimation{1: est1, 2: est2, 3: est3}
	err = to.scriptsComplexity.saveComplexitiesForAddr(addr, estimations, blockID0)
	assert.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 1, true)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 2, true)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 3, true)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)

	to.stor.flush(t)

	res1, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 1, true)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 2, true)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 3, true)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)
}

func TestSaveComplexityForAsset(t *testing.T) {
	to, path, err := createScriptsComplexityStorageObjects()
	assert.NoError(t, err, "createScriptsComplexityStorageObjects() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	asset := testGlobal.asset0.asset.ID
	est1 := ride.TreeEstimation{Estimation: 500, Verifier: 500}
	est2 := ride.TreeEstimation{Estimation: 600, Verifier: 600}
	est3 := ride.TreeEstimation{Estimation: 700, Verifier: 700}
	estimations := map[int]ride.TreeEstimation{1: est1, 2: est2, 3: est3}
	err = to.scriptsComplexity.saveComplexitiesForAsset(asset, estimations, blockID0)
	assert.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptComplexityByAsset(asset, 1, true)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err := to.scriptsComplexity.newestScriptComplexityByAsset(asset, 2, true)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err := to.scriptsComplexity.newestScriptComplexityByAsset(asset, 3, true)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)

	to.stor.flush(t)

	res1, err = to.scriptsComplexity.newestScriptComplexityByAsset(asset, 1, true)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err = to.scriptsComplexity.newestScriptComplexityByAsset(asset, 2, true)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err = to.scriptsComplexity.newestScriptComplexityByAsset(asset, 3, true)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)
}
