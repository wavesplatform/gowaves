package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type sponsoredAssetsTestObjects struct {
	stor            *testStorageObjects
	features        *features
	sponsoredAssets *sponsoredAssets
}

func createSponsoredAssets(t *testing.T, doubleActivation bool) *sponsoredAssetsTestObjects {
	stor := createStorageObjects(t, true)
	sets := settings.MainNetSettings
	sets.SponsorshipSingleActivationPeriod = !doubleActivation
	features, ok := stor.entities.features.(*features)
	require.True(t, ok)
	sponsoredAssets := newSponsoredAssets(stor.rw, features, stor.hs, sets, true)
	return &sponsoredAssetsTestObjects{stor, features, sponsoredAssets}
}

func TestSponsorAsset(t *testing.T) {
	to := createSponsoredAssets(t, true)

	to.stor.addBlock(t, blockID0)
	properCost := uint64(100500)

	id := testGlobal.asset0.asset.ID
	assetIDDigest := proto.AssetIDFromDigest(id)
	err := to.sponsoredAssets.sponsorAsset(id, properCost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	newestIsSponsored, err := to.sponsoredAssets.newestIsSponsored(assetIDDigest)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, true)
	isSponsored, err := to.sponsoredAssets.isSponsored(assetIDDigest)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)

	newestCost, err := to.sponsoredAssets.newestAssetCost(assetIDDigest)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	_, err = to.sponsoredAssets.assetCost(assetIDDigest)
	assert.Error(t, err, "assetCost() did not fail witn new asset before flushing")
	// Flush.
	to.stor.flush(t)
	newestIsSponsored, err = to.sponsoredAssets.newestIsSponsored(assetIDDigest)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, true)
	isSponsored, err = to.sponsoredAssets.isSponsored(assetIDDigest)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, true)
	newestCost, err = to.sponsoredAssets.newestAssetCost(assetIDDigest)
	assert.NoError(t, err, "newestAssetCost() failed")
	assert.Equal(t, newestCost, properCost)
	cost, err := to.sponsoredAssets.assetCost(assetIDDigest)
	assert.NoError(t, err, "assetCost() failed")
	assert.Equal(t, cost, properCost)
	// Check that asset with 0 cost is no longer considered sponsored.
	err = to.sponsoredAssets.sponsorAsset(id, 0, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	newestIsSponsored, err = to.sponsoredAssets.newestIsSponsored(assetIDDigest)
	assert.NoError(t, err, "newestIsSponsored() failed")
	assert.Equal(t, newestIsSponsored, false)
	to.stor.flush(t)
	isSponsored, err = to.sponsoredAssets.isSponsored(assetIDDigest)
	assert.NoError(t, err, "isSponsored() failed")
	assert.Equal(t, isSponsored, false)
}

func TestSponsorAssetUncertain(t *testing.T) {
	to := createSponsoredAssets(t, true)

	properCost := uint64(100500)
	assetIDDigest := testGlobal.asset0.asset.ID
	assetID := proto.AssetIDFromDigest(assetIDDigest)
	test := func() {
		to.stor.addBlock(t, blockID0)
		to.sponsoredAssets.sponsorAssetUncertain(assetIDDigest, properCost)
		newestIsSponsored, err := to.sponsoredAssets.newestIsSponsored(assetID)

		assert.NoError(t, err, "newestIsSponsored() failed")
		assert.Equal(t, newestIsSponsored, true)
		isSponsored, err := to.sponsoredAssets.isSponsored(assetID)
		assert.NoError(t, err, "isSponsored() failed")
		assert.Equal(t, isSponsored, false)
		newestCost, err := to.sponsoredAssets.newestAssetCost(assetID)
		assert.NoError(t, err, "newestAssetCost() failed")
		assert.Equal(t, newestCost, properCost)
		_, err = to.sponsoredAssets.assetCost(assetID)
		assert.Error(t, err, "assetCost() did not fail witn new asset before flushing")
	}
	tests := []struct {
		drop, commit bool
	}{
		{true, false},
		{false, true},
	}
	for _, tc := range tests {
		test()
		if tc.drop {
			to.sponsoredAssets.dropUncertain()

			_, err := to.sponsoredAssets.newestAssetCost(assetID)
			assert.Error(t, err)
			newestIsSponsored, err := to.sponsoredAssets.newestIsSponsored(assetID)
			assert.NoError(t, err)
			assert.Equal(t, false, newestIsSponsored)
		} else if tc.commit {
			err := to.sponsoredAssets.commitUncertain(blockID0)
			assert.NoError(t, err)

			cost, err := to.sponsoredAssets.newestAssetCost(assetID)
			assert.NoError(t, err)
			assert.Equal(t, properCost, cost)
			newestIsSponsored, err := to.sponsoredAssets.newestIsSponsored(assetID)
			assert.NoError(t, err)
			assert.Equal(t, true, newestIsSponsored)
		}
		to.sponsoredAssets.dropUncertain()
	}
}

func TestSponsoredAssetToWaves(t *testing.T) {
	to := createSponsoredAssets(t, true)

	to.stor.addBlock(t, blockID0)
	cost := uint64(2)
	assetAmount := uint64(100500)
	properWavesAmount := assetAmount / cost * FeeUnit
	id := testGlobal.asset0.asset.ID
	assetID := proto.AssetIDFromDigest(id)
	err := to.sponsoredAssets.sponsorAsset(id, cost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	wavesAmount, err := to.sponsoredAssets.sponsoredAssetToWaves(assetID, assetAmount)
	assert.NoError(t, err, "sponsoredAssetToWaves() failed")
	assert.Equal(t, wavesAmount, properWavesAmount)
}

func TestWavesToSponsoredAsset(t *testing.T) {
	to := createSponsoredAssets(t, true)

	to.stor.addBlock(t, blockID0)
	cost := uint64(2)
	wavesAmount := uint64(100500)
	properAssetAmount := wavesAmount / FeeUnit * cost
	id := testGlobal.asset0.asset.ID
	assetID := proto.AssetIDFromDigest(id)
	err := to.sponsoredAssets.sponsorAsset(id, cost, blockID0)
	assert.NoError(t, err, "sponsorAsset() failed")
	assetAmount, err := to.sponsoredAssets.wavesToSponsoredAsset(assetID, wavesAmount)
	assert.NoError(t, err, "wavesToSponsoredAsset() failed")
	assert.Equal(t, assetAmount, properAssetAmount)
}

func TestIsSponsorshipActivated_Double(t *testing.T) {
	to := createSponsoredAssets(t, true)

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

func TestIsSponsorshipActivated_Single(t *testing.T) {
	to := createSponsoredAssets(t, false)

	// False before activation.
	isSponsorshipActivated, err := to.sponsoredAssets.isSponsorshipActivated()
	assert.NoError(t, err, "isSponsorshipActivated() failed")
	assert.Equal(t, false, isSponsorshipActivated)

	// True after activation.
	to.stor.activateFeature(t, int16(settings.FeeSponsorship))
	isSponsorshipActivated, err = to.sponsoredAssets.isSponsorshipActivated()
	assert.NoError(t, err, "isSponsorshipActivated() failed")
	assert.Equal(t, true, isSponsorshipActivated)
}
