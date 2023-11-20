package state

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type ordersVolumesStorageObjects struct {
	stor          *testStorageObjects
	ordersVolumes *ordersVolumes
}

func createOrdersVolumeStorageObjects(t *testing.T) *ordersVolumesStorageObjects {
	stor := createStorageObjects(t, true)
	ordersVolumes := newOrdersVolumes(stor.hs)
	return &ordersVolumesStorageObjects{stor, ordersVolumes}
}

func TestIncreaseFilled(t *testing.T) {
	to := createOrdersVolumeStorageObjects(t)

	to.stor.addBlock(t, blockID0)
	orderID := bytes.Repeat([]byte{0xff}, crypto.DigestSize)

	const (
		firstFee     = uint64(1)
		secondFee    = uint64(100500)
		firstAmount  = uint64(111)
		secondAmount = uint64(500100)
	)

	err := to.ordersVolumes.increaseFilled(orderID, firstAmount, firstFee, blockID0)
	assert.NoError(t, err)
	filledAmount, filledFee, err := to.ordersVolumes.newestFilled(orderID)
	assert.NoError(t, err)
	assert.Equal(t, firstFee, filledFee)
	assert.Equal(t, firstAmount, filledAmount)

	err = to.ordersVolumes.increaseFilled(orderID, secondAmount, secondFee, blockID0)
	assert.NoError(t, err)
	filledAmount, filledFee, err = to.ordersVolumes.newestFilled(orderID)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)

	to.stor.flush(t)
	filledAmount, filledFee, err = to.ordersVolumes.newestFilled(orderID)
	assert.NoError(t, err)
	assert.Equal(t, firstFee+secondFee, filledFee)
	assert.Equal(t, firstAmount+secondAmount, filledAmount)
}
