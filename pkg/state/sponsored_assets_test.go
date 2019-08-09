package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type sponsoredAssetsTestObjects struct {
	stor            *storageObjects
	features        *features
	sponsoredAssets *sponsoredAssets
}

func createSponsoredAssets() (*sponsoredAssetsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	features, err := newFeatures(stor.db, stor.dbBatch, stor.hs, stor.stateDB, settings.MainNetSettings, settings.FeaturesInfo)
	if err != nil {
		return nil, path, err
	}
	sponsoredAssets, err := newSponsoredAssets(features, stor.stateDB, stor.hs, settings.MainNetSettings)
	if err != nil {
		return nil, path, err
	}
	return &sponsoredAssetsTestObjects{stor, features, sponsoredAssets}, path, nil
}

func TestSponsorAsset(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	properCost := uint64(100500)
	id := testGlobal.asset0.asset.ID
	err = to.sponsoredAssets.sponsorAsset(id, properCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	newestIsSponsored := to.sponsoredAssets.newestIsSponsored(id, true)
	assert.Equal(t, newestIsSponsored, true)
	isSponsored := to.sponsoredAssets.isSponsored(id, true)
	assert.Equal(t, isSponsored, false)
	newestCost, err := to.sponsoredAssets.newestAssetCost(id, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	_, err = to.sponsoredAssets.assetCost(id, true)
	assert.Error(t, err, "assetCost() did not fail witn new asset before flushing")
	// Flush.
	to.stor.flush(t)
	newestIsSponsored = to.sponsoredAssets.newestIsSponsored(id, true)
	assert.Equal(t, newestIsSponsored, true)
	isSponsored = to.sponsoredAssets.isSponsored(id, true)
	assert.Equal(t, isSponsored, true)
	newestCost, err = to.sponsoredAssets.newestAssetCost(id, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	cost, err := to.sponsoredAssets.assetCost(id, true)
	assert.NoError(t, err, "assetCost() failed")
	assert.Equal(t, cost, properCost)
}

func TestSponsoredAssetToWaves(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	cost := uint64(2)
	assetAmount := uint64(100500)
	properWavesAmount := assetAmount / cost * FeeUnit
	id := testGlobal.asset0.asset.ID
	err = to.sponsoredAssets.sponsorAsset(id, cost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	wavesAmount, err := to.sponsoredAssets.sponsoredAssetToWaves(id, assetAmount)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	assert.Equal(t, wavesAmount, properWavesAmount)
}

func TestWavesToSponsoredAsset(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	cost := uint64(2)
	wavesAmount := uint64(100500)
	properAssetAmount := wavesAmount / FeeUnit * cost
	id := testGlobal.asset0.asset.ID
	err = to.sponsoredAssets.sponsorAsset(id, cost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	assetAmount, err := to.sponsoredAssets.wavesToSponsoredAsset(id, wavesAmount)
	assert.NoError(t, err, "wavesToSponsoredAsset() failed")
	assert.Equal(t, assetAmount, properAssetAmount)
}

func TestSponsoredFeesSwitchHeight(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	activateFeature(t, to.features, to.stor, int16(settings.FeeSponsorship))
	switchHeight, err := to.sponsoredAssets.sponsoredFeesSwitchHeight()
	assert.NoError(t, err, "sponsoredFeesSwitchHeight() failed")
	activationHeight, err := to.features.activationHeight(int16(settings.FeeSponsorship))
	assert.NoError(t, err, "activationHeight() failed")
	properSwitchHeight := activationHeight + to.sponsoredAssets.settings.ActivationWindowSize(activationHeight)
	assert.Equal(t, switchHeight, properSwitchHeight)
}
