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
	var (
		se1 = scriptEstimation{
			currentEstimatorVersion: 1,
			scriptIsEmpty:           false,
			estimation: ride.TreeEstimation{
				Estimation: 1234567890,
				Verifier:   11111,
				Functions:  map[string]int{"lightFunc": 123, "heavyFunc": 1000, "superHeavyFunc": 1234567890},
			},
		}
		se2 = scriptEstimation{
			currentEstimatorVersion: 2,
			scriptIsEmpty:           false,
			estimation: ride.TreeEstimation{
				Estimation: 5647382910,
				Verifier:   22222,
				Functions:  map[string]int{"lightFunc": 456, "heavyFunc": 2000, "superHeavyFunc": 5647382910},
			},
		}
		se3 = scriptEstimation{
			currentEstimatorVersion: 3,
			scriptIsEmpty:           false,
			estimation: ride.TreeEstimation{
				Estimation: 9876543210,
				Verifier:   33333,
				Functions:  map[string]int{"lightFunc": 789, "heavyFunc": 3000, "superHeavyFunc": 9876543210},
			},
		}
		seEmpty = scriptEstimation{
			currentEstimatorVersion: 3,
			scriptIsEmpty:           true,
			estimation:              ride.TreeEstimation{},
		}
	)
	err := to.scriptsComplexity.saveComplexitiesForAddr(addr, se1, blockID0)
	require.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, se1.currentEstimatorVersion, int(res1.EstimatorVersion))
	assert.Equal(t, se1.estimation, res1.Estimation)

	err = to.scriptsComplexity.saveComplexitiesForAddr(addr, se2, blockID0)
	require.NoError(t, err)
	res2, err := to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, se2.currentEstimatorVersion, int(res2.EstimatorVersion))
	assert.Equal(t, se2.estimation, res2.Estimation)

	to.stor.flush(t)

	resFlushed, err := to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, se2.currentEstimatorVersion, int(resFlushed.EstimatorVersion))
	assert.Equal(t, se2.estimation, resFlushed.Estimation)

	err = to.scriptsComplexity.updateCallableComplexitiesForAddr(addr, se3, blockID0)
	require.NoError(t, err)
	res3, err := to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, se3.currentEstimatorVersion, int(res3.EstimatorVersion))
	assert.Equal(t, se3.estimation.Functions, res3.Estimation.Functions)
	assert.Equal(t, se3.estimation.Estimation, res3.Estimation.Estimation)
	assert.Equal(t, se2.estimation.Verifier, res3.Estimation.Verifier)

	to.stor.flush(t)

	resFlushed, err = to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	require.NoError(t, err)
	assert.Equal(t, se3.currentEstimatorVersion, int(resFlushed.EstimatorVersion))
	assert.Equal(t, se3.estimation.Functions, resFlushed.Estimation.Functions)
	assert.Equal(t, se3.estimation.Estimation, res3.Estimation.Estimation)
	assert.Equal(t, se2.estimation.Verifier, resFlushed.Estimation.Verifier)

	err = to.scriptsComplexity.saveComplexitiesForAddr(addr, seEmpty, blockID0)
	require.NoError(t, err)
	_, err = to.scriptsComplexity.newestScriptEstimationRecordByAddr(addr)
	assert.EqualError(t, err,
		"failed to unmarshal account script complexities record: empty binary data, estimation doesn't exist",
	)

	failAddr := testGlobal.recipientInfo.addr
	err = to.scriptsComplexity.updateCallableComplexitiesForAddr(failAddr, se1, blockID0)
	assert.EqualError(t, err, "failed to update callable complexities for addr '"+failAddr.String()+"': "+
		"not found")
}

func TestSaveComplexityForAsset(t *testing.T) {
	to := createScriptsComplexityStorageObjects(t)

	to.stor.addBlock(t, blockID0)
	asset := testGlobal.asset0.asset.ID
	assetID := proto.AssetIDFromDigest(asset)
	est := scriptEstimation{
		currentEstimatorVersion: maxEstimatorVersion,
		scriptIsEmpty:           false,
		estimation:              ride.TreeEstimation{Estimation: 500, Verifier: 500},
	}
	err := to.scriptsComplexity.saveComplexitiesForAsset(asset, est, blockID0)
	assert.NoError(t, err)
	res1, err := to.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	require.NoError(t, err)
	assert.Equal(t, est.estimation, *res1)

	to.stor.flush(t)

	res1, err = to.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	require.NoError(t, err)
	assert.Equal(t, est.estimation, *res1)
}
