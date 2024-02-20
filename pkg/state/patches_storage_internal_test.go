package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func Test_patchesStorage(t *testing.T) {
	var (
		blockIDs      = genRandBlockIds(t, 2)
		fistBlockID   = blockIDs[0]
		secondBlockID = blockIDs[1]
	)

	addr, err := proto.NewAddressFromString("3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi")
	require.NoError(t, err)
	assetID, err := crypto.NewDigestFromBase58("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	require.NoError(t, err)

	to := createStorageObjects(t, true)
	to.addBlock(t, fistBlockID)

	patch := []proto.AtomicSnapshot{
		proto.WavesBalanceSnapshot{Address: addr, Balance: 100500},
		proto.AssetBalanceSnapshot{Address: addr, AssetID: assetID, Balance: 100500},
	}

	patchesStor := newPatchesStorage(to.hs, to.settings.AddressSchemeCharacter)

	err = patchesStor.savePatch(fistBlockID, patch)
	require.NoError(t, err)

	actual, err := patchesStor.newestPatch(fistBlockID)
	require.NoError(t, err)
	assert.Equal(t, patch, actual)

	to.flush(t)
	// check after flush
	actual, err = patchesStor.newestPatch(fistBlockID)
	require.NoError(t, err)
	assert.Equal(t, patch, actual)

	// get patch for non-existing block
	actual, err = patchesStor.newestPatch(secondBlockID)
	require.NoError(t, err)
	assert.Empty(t, actual)
	assert.Nil(t, actual)

	// save empty patch for second block
	// should be no-op, so we don't initialize block intentionally
	err = patchesStor.savePatch(secondBlockID, nil)
	require.NoError(t, err)
}
