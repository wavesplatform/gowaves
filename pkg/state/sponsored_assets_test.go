package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type sponsoredAssetsTestObjects struct {
	stor            *testStorageObjects
	features        *features
	sponsoredAssets *sponsoredAssets
}

func createSponsoredAssets() (*sponsoredAssetsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	features, err := newFeatures(stor.rw, stor.db, stor.hs, settings.MainNetSettings, settings.FeaturesInfo)
	if err != nil {
		return nil, path, err
	}
	sponsoredAssets, err := newSponsoredAssets(stor.rw, features, stor.hs, settings.MainNetSettings)
	if err != nil {
		return nil, path, err
	}
	return &sponsoredAssetsTestObjects{stor, features, sponsoredAssets}, path, nil
}

func TestSponsorAsset(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	properCost := uint64(100500)
	id := testGlobal.asset0.asset.ID
	err = to.sponsoredAssets.sponsorAsset(id, properCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	newestIsSponsored, err := to.sponsoredAssets.newestIsSponsored(id, true)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, true)
	isSponsored, err := to.sponsoredAssets.isSponsored(id, true)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)
	newestCost, err := to.sponsoredAssets.newestAssetCost(id, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	_, err = to.sponsoredAssets.assetCost(id, true)
	assert.Error(t, err, "assetCost() did not fail witn new asset before flushing")
	// Flush.
	to.stor.flush(t)
	newestIsSponsored, err = to.sponsoredAssets.newestIsSponsored(id, true)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, true)
	isSponsored, err = to.sponsoredAssets.isSponsored(id, true)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, true)
	newestCost, err = to.sponsoredAssets.newestAssetCost(id, true)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	cost, err := to.sponsoredAssets.assetCost(id, true)
	assert.NoError(t, err, "assetCost() failed")
	assert.Equal(t, cost, properCost)
	// Check that asset with 0 cost is no longer considered sponsored.
	err = to.sponsoredAssets.sponsorAsset(id, 0, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	newestIsSponsored, err = to.sponsoredAssets.newestIsSponsored(id, true)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, false)
	to.stor.flush(t)
	isSponsored, err = to.sponsoredAssets.isSponsored(id, true)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)
}

func TestSponsoredAssetToWaves(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

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
		to.stor.close(t)

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

func TestIsSponsorshipActivated(t *testing.T) {
	to, path, err := createSponsoredAssets()
	assert.NoError(t, err, "createSponsoredAssets() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// False before activation.
	isSponsorshipActivated, err := to.sponsoredAssets.isSponsorshipActivated()
	assert.NoError(t, err, "isSponsorshipActivated() failed")
	assert.Equal(t, false, isSponsorshipActivated)

	// False after activation.
	to.stor.activateFeature(t, int16(settings.FeeSponsorship))
	isSponsorshipActivated, err = to.sponsoredAssets.isSponsorshipActivated()
	assert.NoError(t, err, "isSponsorshipActivated() failed")
	assert.Equal(t, false, isSponsorshipActivated)

	// True after windowSize blocks after activation.
	to.stor.activateSponsorship(t)
	isSponsorshipActivated, err = to.sponsoredAssets.isSponsorshipActivated()
	assert.NoError(t, err, "isSponsorshipActivated() failed")
	assert.Equal(t, true, isSponsorshipActivated)
}
