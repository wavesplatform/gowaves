package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"testing"
)

func TestBlockInfoBinaryRoundTrip(t *testing.T) {
	var s crypto.Signature
	bi := BlockInfo{
		Empty:             false,
		ID:                s,
		EarliestTimeFrame: 12345,
	}
	b := bi.marshalBinary()
	var ab BlockInfo
	err := ab.unmarshalBinary(b)
	assert.NoError(t, err)
	assert.Equal(t, bi.EarliestTimeFrame, ab.EarliestTimeFrame)
	assert.ElementsMatch(t, bi.ID, ab.ID)
	assert.Equal(t, bi.Empty, ab.Empty)
}
