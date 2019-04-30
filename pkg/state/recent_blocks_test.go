package state

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

var (
	bottomLimit = rollbackMaxBlocks
)

func createRecentBlocks(t *testing.T) *recentBlocks {
	rb, err := newRecentBlocks()
	assert.NoError(t, err, "newRecentBlocks() failed")
	return rb
}

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

type recentTest struct {
	bottomLimit int
	isRecent    bool
}

func checkRecent(t *testing.T, rb *recentBlocks, ids []crypto.Signature, tc *recentTest) {
	for _, id := range ids {
		isRecent, err := rb.blockIsRecent(id, tc.bottomLimit)
		assert.NoError(t, err, "blockIsRecent() failed")
		assert.Equal(t, tc.isRecent, isRecent, "blockIsRecent() returned incorrect result")
	}
}

func TestBlockIsRecent(t *testing.T) {
	rb := createRecentBlocks(t)
	assert.Equal(t, true, rb.isEmpty())
	ids := genIds(t, rollbackMaxBlocks*2)
	for _, id := range ids {
		err := rb.addBlockID(id)
		assert.NoError(t, err, "addBlockID() failed")
	}
	assert.Equal(t, false, rb.isEmpty())
	tc0 := &recentTest{bottomLimit, false}
	checkRecent(t, rb, ids[:len(ids)-bottomLimit], tc0)
	tc1 := &recentTest{bottomLimit, true}
	checkRecent(t, rb, ids[len(ids)-bottomLimit:], tc1)
	rb.reset()
	assert.Equal(t, true, rb.isEmpty())
	tc2 := &recentTest{0, false}
	checkRecent(t, rb, ids, tc2)
}
