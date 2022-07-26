package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type scriptsComplexityStorageObjects struct {
	stor              *testStorageObjects
	scriptsComplexity *scriptsComplexity
}

func createScriptsComplexityStorageObjects(t *testing.T) *scriptsComplexityStorageObjects {
	stor := createStorageObjects(t, true)
	scriptsComplexity := newScriptsComplexity(stor.hs)
	return &scriptsComplexityStorageObjects{stor, scriptsComplexity}
}

func TestSaveComplexityForAddr(t *testing.T) {
	to := createScriptsComplexityStorageObjects(t)

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
	err := to.scriptsComplexity.saveComplexitiesForAddr(addr, estimations, blockID0)
	assert.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 1)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 2)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err := to.scriptsComplexity.newestScriptComplexityByAddr(addr, 3)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)
	res, err := to.scriptsComplexity.newestOriginalScriptComplexityByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, est1, *res)

	to.stor.flush(t)

	res1, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 1)
	require.NoError(t, err)
	assert.Equal(t, est1, *res1)
	res2, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 2)
	require.NoError(t, err)
	assert.Equal(t, est2, *res2)
	res3, err = to.scriptsComplexity.newestScriptComplexityByAddr(addr, 3)
	require.NoError(t, err)
	assert.Equal(t, est3, *res3)
	res, err = to.scriptsComplexity.newestOriginalScriptComplexityByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, est1, *res)
}

func TestSaveComplexityForAsset(t *testing.T) {
	to := createScriptsComplexityStorageObjects(t)

	to.stor.addBlock(t, blockID0)
	asset := testGlobal.asset0.asset.ID
	assetID := proto.AssetIDFromDigest(asset)
	est := ride.TreeEstimation{Estimation: 500, Verifier: 500}
	err := to.scriptsComplexity.saveComplexitiesForAsset(asset, est, blockID0)
	assert.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	require.NoError(t, err)
	assert.Equal(t, est, *res1)

	to.stor.flush(t)

	res1, err = to.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	require.NoError(t, err)
	assert.Equal(t, est, *res1)
}
