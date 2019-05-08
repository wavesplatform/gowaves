package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var (
	rangeSize = rollbackMaxBlocks
)

func genIds(t *testing.T, size int) []crypto.Signature {
	res := make([]crypto.Signature, size)
	for i := 0; i < size; i++ {
		id := make([]byte, crypto.SignatureSize)
		_, err := rand.Read(id)
		blockID, err := crypto.NewSignatureFromBytes(id)
		assert.NoError(t, err, "crypto.NewSignatureFromBytes() failed")
		res[i] = blockID
	}
	return res
}

type idChecker = func(crypto.Signature) (uint64, error)

func checkHeights(t *testing.T, ids []crypto.Signature, properHeights []uint64, check idChecker) {
	for i, id := range ids {
		height, err := check(id)
		assert.NoError(t, err, "blockIDToHeight failed")
		assert.Equal(t, properHeights[i], height, "blockIDToHeight() returned incorrect result")
	}
}

func TestIsInRange(t *testing.T) {
	rb, err := newRecentBlocks(rangeSize)
	assert.NoError(t, err, "newRecentBlocks() failed")
	assert.Equal(t, true, rb.isEmpty())
	ids := genIds(t, rangeSize)
	heights := make([]uint64, rangeSize)
	// Test indirect addition of IDs.
	for i, id := range ids {
		err := rb.addNewBlockID(id)
		assert.NoError(t, err, "addNewBlockID() failed")
		heights[i] = uint64(i)
	}
	assert.Equal(t, true, rb.isEmpty())
	checkHeights(t, ids, heights, rb.newBlockIDToHeight)
	rb.flush()
	assert.Equal(t, false, rb.isEmpty())
	checkHeights(t, ids, heights, rb.blockIDToHeight)
	rb.reset()
	assert.Equal(t, true, rb.isEmpty())
	// Now test direct addition of IDs.
	for _, id := range ids {
		err := rb.addBlockID(id)
		assert.NoError(t, err, "addBlockID() failed")
	}
	assert.Equal(t, false, rb.isEmpty())
	checkHeights(t, ids, heights, rb.blockIDToHeight)
}
